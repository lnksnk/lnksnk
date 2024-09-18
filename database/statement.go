package database

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"reflect"

	//"github.com/lnksnk/lnksnk/caching"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/concurrent"
	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
	"github.com/lnksnk/lnksnk/parameters"
)

type Statement struct {
	ctx       context.Context
	cn        *Connection
	isRemote  bool
	prepstmnt *sql.Stmt
	prms      *parameters.Parameters
	rdr       *Reader
	args      *sync.Map
	stmntlck  *sync.RWMutex
	stmnt     string
	argnames  []string
	argtypes  []int
	//parseSqlParam func(totalArgs int) (s string)
}

func NewStatement(cn *Connection) (stmnt *Statement) {
	if cn != nil {
		stmnt = &Statement{cn: cn, isRemote: cn.isRemote(), stmntlck: &sync.RWMutex{}, args: &sync.Map{}}
	}
	return
}

func parseParam(parseSqlParam func(totalArgs int) (s string), totalArgs int) (s string) {
	if parseSqlParam != nil {
		s = parseSqlParam(totalArgs)
	} else {
		s = "?"
	}

	return
}

type StatementHandler interface {
	Prepair(...interface{}) []interface{}
}

type StatementHandlerFunc func(a ...interface{}) []interface{}

func (stmnthndlfnc StatementHandlerFunc) Prepair(a ...interface{}) []interface{} {
	return stmnthndlfnc(a...)
}

func (stmnt *Statement) Prepair(prms *parameters.Parameters, rdr *Reader, args map[string]interface{}, a ...interface{}) (preperr error) {
	if stmnt != nil {
		defer func() {
			if preperr != nil && stmnt != nil {
				stmnt.Close()
			}
		}()
		//var rnrr io.RuneReader = nil
		var qrybuf = iorw.NewBuffer()
		defer qrybuf.Close()
		var validNames []string
		var validNameType []int
		var fs *fsutils.FSUtils = nil
		var al = len(a)
		var ai = 0
		stmntref := &stmnt.stmnt
		var cchng *concurrent.Map = nil
		var ctx context.Context = nil
		var stmnthndlr StatementHandler = nil
		var parseSqlParam func(totalArgs int) (s string)
		if stmnt.cn != nil && stmnt.cn.dbParseSqlParam != nil {
			parseSqlParam = stmnt.cn.dbParseSqlParam
		}
		for ai < al {
			if d := a[ai]; d != nil {
				if stmnthndld, _ := d.(StatementHandler); stmnthndld != nil {
					if stmnthndlr == nil {
						stmnthndlr = stmnthndld
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				} else if fsd, _ := d.(*fsutils.FSUtils); fsd != nil {
					if fs == nil {
						fs = fsd
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				} else if ccnngd, _ := d.(*concurrent.Map); ccnngd != nil {
					if cchng == nil {
						cchng = ccnngd
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				} else if ctxd, _ := d.(context.Context); ctxd != nil {
					if ctx == nil {
						ctx = ctxd
					}
					a = append(a[:ai], a[ai+1:]...)
					al--
					continue
				}
			}
			ai++
		}
		if vqry, vqryfnd := cchng.Find(a...); vqryfnd && vqry != nil {
			qrybuf.Print(vqry)
		} else {
			qrybuf.Print(a...)
		}

		if fs != nil {
			if fi := func() fsutils.FileInfo {
				if tstsql := qrybuf.String() + func() string {
					if !qrybuf.HasSuffix(".sql") {
						return ".sql"
					}
					return ""
				}(); tstsql != "" {
					if fio := fs.LS(tstsql); len(fio) == 1 {
						return fio[0]
					}
					if fio := fs.LS(tstsql[:len(tstsql)-len(".sql")] + "." + stmnt.cn.driverName + ".sql"); len(fio) == 1 {
						return fio[0]
					}
				}
				return nil
			}(); fi != nil && stmnthndlr != nil {
				qrybuf.Clear()
				qrybuf.Print(stmnthndlr.Prepair(fi))
			}
		}

		if qrybuf.HasPrefix("#") && qrybuf.HasSuffix("#") {
			if substrqry, _ := qrybuf.SubString(1, qrybuf.Size()-1); substrqry != "" {
				subqryarr := strings.Split(substrqry, "=>")
				subqry := make([]interface{}, len(subqryarr))
				for subn, sub := range subqryarr {
					subqry[subn] = strings.TrimSpace(sub)
				}
				if valfnd, valfndok := cchng.Find(subqry...); valfndok && valfnd != nil {
					qrybuf.Clear()
					qrybuf.Print(valfnd)
				} else {
					qrybuf.Clear()
					qrybuf.Print(substrqry)
				}
			}
		}
		defer qrybuf.Close()

		var foundTxt = false

		var possibleArgName map[string]int = map[string]int{}
		var possibleArgSize map[string]int = map[string]int{}
		paramkeys := prms.StandardKeys()
		prmkschkd := map[string]bool{}
		if len(args) > 0 {
			for dfltk, dfltv := range args {
				for prmn, prmk := range paramkeys {
					if strings.EqualFold(prmk, dfltk) {
						if prms.StringParameter(prmk, "") == "" {
							paramkeys = append(paramkeys[:prmn], paramkeys[prmn+1:]...)
							prmkschkd[dfltk] = true
							possibleArgName[dfltk] = 0
							possibleArgSize[dfltk] = 1
							break
						}
						prmkschkd[dfltk] = true
						break
					}
				}
				if !prmkschkd[dfltk] {
					possibleArgName[dfltk] = 0
					if reflect.TypeOf(dfltv).Kind() == reflect.Array || reflect.TypeOf(dfltv).Kind() == reflect.Slice {
						possibleArgSize[dfltk] = reflect.ValueOf(dfltv).Len()
					} else {
						possibleArgSize[dfltk] = 1
					}
				}
			}
		}

		for _, dfltk := range paramkeys {
			prmv := prms.Parameter(dfltk)
			possibleArgName[dfltk] = 1
			possibleArgSize[dfltk] = len(prmv)
		}

		if rdr != nil {
			for _, dfltk := range rdr.Columns() {
				for prmk := range possibleArgName {
					if strings.EqualFold(prmk, dfltk) && prmk != dfltk {
						delete(possibleArgName, prmk)
						delete(possibleArgSize, prmk)
					}
				}
				possibleArgName[dfltk] = 2
				possibleArgSize[dfltk] = 1
			}
		}

		qrybdr := qrybuf.Clone(true).Reader(true)

		bsy := false
		qrybuf.Print(iorw.ReadRunesUntil(qrybdr, iorw.RunesUntilSliceFlushFunc(func(phrase string, untilrdr io.RuneReader, orgrd iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) (fnerr error) {
			if phrase == "@" {
				if foundTxt {
					flushrdr.PreAppendArgs(phrase)
					return
				}
				if bsy {
					return fmt.Errorf("%s", phrase)
				}
				if !bsy {
					bsy = true
				}
				defer func() {
					bsy = false
				}()
				argbf, argbferr := iorw.NewBufferError(untilrdr)
				if argbferr != nil {
					if argbferr.Error() == "@" {
						if argbf.Empty() {
							return
						}
						fndprm := true
						argbfl := argbf.Size()
						for mpvk, mpkv := range possibleArgName {
							if fndprm = int64(len(mpvk)) >= argbfl && strings.EqualFold(argbf.String(), mpvk); fndprm {
								if validNames == nil {
									validNames = []string{}
								}
								if validNameType == nil {
									validNameType = []int{}
								}
								argss := possibleArgSize[mpvk]
								if argss > 0 {
									tmpsql := ""
									for argi := range argss {
										tmpsql += parseParam(parseSqlParam, argi+len(validNames)) + func() string {
											if argi < argss-1 {
												return ","
											}
											return ""
										}()
									}
									flushrdr.PreAppendArgs(tmpsql)
									validNames = append(validNames, mpvk)
									validNameType = append(validNameType, mpkv)
								}
								argbferr = nil
								return
							}
						}
						argbferr = nil
						if !fndprm {
							flushrdr.PreAppendArgs("''")
						}
						return
					}
					if argbferr.Error() == "'" {
						if !argbf.Empty() {
							flushrdr.PreAppendArgs(argbf.Reader(true), argbferr.Error())
							argbferr = nil
							return
						}
						flushrdr.PreAppendArgs(argbferr.Error())
						argbferr = nil
						return
					}
				}
				flushrdr.PreAppendArgs(phrase)
				return
			}
			if phrase == "'" {
				if bsy {
					return fmt.Errorf("%s", phrase)
				}
				if !foundTxt {
					foundTxt = true
					flushrdr.PreAppendArgs(phrase)
					return
				}
				foundTxt = false
				flushrdr.PreAppendArgs(phrase)
				return
			}
			return
		}), "@", "'"))

		*stmntref = qrybuf.String()
		if refrdr := stmnt.rdr; rdr != nil && refrdr != rdr {
			stmnt.rdr = rdr
		}

		if len(args) > 0 {
			if argssnc := stmnt.args; argssnc != nil {
				for ak, av := range args {
					argssnc.Store(ak, av)
				}
			}
		}
		if len(validNames) > 0 {
			stmnt.argnames = validNames[:]
			stmnt.argtypes = validNameType[:]
		}

		if refprms := stmnt.prms; prms != nil && prms != refprms {
			stmnt.prms = prms
		}
		if stmnt.prepstmnt == nil && stmnt.cn.isRemote() {

		} else {
			if ctx != nil && stmnt.ctx != ctx {
				stmnt.ctx = ctx
			}
			if stmnt.prepstmnt == nil {
				if db, dberr := stmnt.cn.DbInvoke(); dberr == nil && db != nil {
					if stmnt.ctx != nil {
						if stmnt.prepstmnt, preperr = db.PrepareContext(stmnt.ctx, stmnt.stmnt); preperr != nil {
							return
						}
					} else if stmnt.prepstmnt, preperr = db.Prepare(stmnt.stmnt); preperr != nil {
						return
					}
				} else if dberr != nil {
					preperr = dberr
				}
			}
		}
	}
	return
}

func (stmnt *Statement) Arguments() (args []interface{}) {
	if stmnt != nil && stmnt.cn != nil && len(stmnt.argnames) > 0 {
		if argssnc, argnames, argtypes, rdr := stmnt.args, stmnt.argnames, stmnt.argtypes, stmnt.rdr; argssnc != nil && len(argnames) > 0 && len(argnames) == len(argtypes) {
			for argn, argnme := range argnames {
				if argtpe := argtypes[argn]; argtpe == 0 {
					if argv, argvok := argssnc.Load(argnme); argvok {
						if reflect.TypeOf(argv).Kind() == reflect.Array || reflect.TypeOf(argv).Kind() == reflect.Slice {
							if argvls, argvlsok := argv.([]interface{}); argvlsok {
								args = append(args, argvls...)
								continue
							}
							if argvls, argvlsok := argv.([]string); argvlsok {
								for _, av := range argvls {
									args = append(args, av)
								}
								continue
							}
							if argvls, argvlsok := argv.([]int); argvlsok {
								for _, av := range argvls {
									args = append(args, av)
								}
								continue
							}
							continue
						}
						args = append(args, argv)
					}
				} else if prms := stmnt.prms; prms != nil && argtpe == 1 {
					prmv := prms.Parameter(argnme)
					prmvl := len(prmv)
					if prmvl == 0 {
						args = append(args, "")
						continue
					}
					if prmvl == 1 {
						args = append(args, prmv[0])
						continue
					}
					for _, prmrgv := range prmv {
						args = append(args, prmrgv)
					}
				} else if rdr != nil && argtpe == 2 {
					if rows := rdr.rows; rows != nil {
						if clsi := rows.FieldIndex(argnme); clsi > -1 {
							args = append(args, rows.FieldByIndex(clsi))
						}
					}
				}
			}
		}
	}
	return
}

func (stmnt *Statement) Query() (rows RowsAPI, err error) {
	if stmnt != nil {
		if ctx, prep := stmnt.ctx, stmnt.prepstmnt; prep != nil {
			var sqlrw *sql.Rows = nil
			if ctx != nil {
				if sqlrw, err = prep.QueryContext(ctx, stmnt.Arguments()...); err == nil && sqlrw != nil {
					rows = newSqlRows(sqlrw, nil, nil)
				}
			} else {
				if sqlrw, err = prep.Query(stmnt.Arguments()...); err == nil && sqlrw != nil {
					rows = newSqlRows(sqlrw, nil, nil)
				}
			}
		}
	}
	return
}

func (stmnt *Statement) Close() (err error) {
	if stmnt != nil {
		if prms := stmnt.prms; prms != nil {
			stmnt.prms = nil
		}

		if args := stmnt.args; args != nil {
			stmnt.args = nil
		}

		if rdr := stmnt.rdr; rdr != nil {
			stmnt.rdr = nil
		}
		if prepstmnt := stmnt.prepstmnt; prepstmnt != nil {
			stmnt.prepstmnt = nil
			err = prepstmnt.Close()
		}
		if cn := stmnt.cn; cn != nil {
			stmnt.cn = nil
		}
	}
	return
}
