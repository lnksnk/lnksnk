package main

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/dbms"
	"github.com/lnksnk/lnksnk/dbms/dlv"
	"github.com/lnksnk/lnksnk/dbms/mssql"
	"github.com/lnksnk/lnksnk/dbms/ora"
	"github.com/lnksnk/lnksnk/dbms/postgres"
	"github.com/lnksnk/lnksnk/dbms/sqlite"
	"github.com/lnksnk/lnksnk/es"
	"github.com/lnksnk/lnksnk/es/fieldmapping"
	"github.com/lnksnk/lnksnk/fonts"
	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/ui"

	"github.com/lnksnk/lnksnk/fs/active"
	"github.com/lnksnk/lnksnk/listen"
	"github.com/lnksnk/lnksnk/mimes"
	sessioning "github.com/lnksnk/lnksnk/serve/serveio/session"
)

func main() {
	chn := make(chan bool, 1)
	var mltyfsys fs.MultiFileSystem = nil
	var compileprogram = func(fsys fs.MultiFileSystem, cde ...interface{}) (prgm interface{}, err error) {
		return es.CompileProgram(fsys, func(refscriptormod interface{}, modspecifier string) (rlsvdmodrec interface{}, rslvderr error) {
			if chdapi, _ := fsys.(interface {
				CachedInfo(path string) (chdfi active.CachedInfo, err error)
			}); chdapi != nil {
				chfi, chdfierr := chdapi.CachedInfo(modspecifier)
				if chdfierr != nil {
					rslvderr = chdfierr
					return
				}
				rlsvdmodrec = chfi.Program()
			}
			return
		}, cde...)
	}
	mltyfsys = active.AciveFileSystem(compileprogram)
	mltyfsys.CacheExtensions(".html", ".js", ".css", ".svg", ".woff2", ".woff", ".ttf", ".eot", ".sql")
	mltyfsys.DefaultExtensions(".html", ".js", ".json", ".css")
	mltyfsys.ActiveExtensions(".html", ".js", ".svg", ".json", ".xml", ".sql")
	mltyfsys.Map("/etl", "C:/projects/cim", true)
	mltyfsys.Map("/media", "C:/movies", true)
	//mltyfsys.Map("/datafiles", "C:/projects/datafiles", true)
	glbldbms := dbms.NewDBMS(mltyfsys)
	dbdrivers := map[string][]interface{}{}
	dbdrivers["postgres"] = []interface{}{dbms.InvokeDBFunc(postgres.InvokeDB), dbms.ParseSqlArgFunc(postgres.ParseSqlParam)}
	dbdrivers["mssql"] = []interface{}{dbms.InvokeDBFunc(mssql.InvokeDB), dbms.ParseSqlArgFunc(mssql.ParseSqlParam)}
	dbdrivers["sqlserver"] = []interface{}{dbms.InvokeDBFunc(mssql.InvokeDB), dbms.ParseSqlArgFunc(mssql.ParseSqlParam)}
	dbdrivers["azuresql"] = []interface{}{dbms.InvokeDBFunc(mssql.InvokeDBAzure), dbms.ParseSqlArgFunc(mssql.ParseSqlParam)}
	dbdrivers["oracle"] = []interface{}{dbms.InvokeDBFunc(ora.InvokeDB), dbms.ParseSqlArgFunc(ora.ParseSqlParam)}
	dbdrivers["sqlite"] = []interface{}{dbms.InvokeDBFunc(sqlite.InvokeDB), dbms.ParseSqlArgFunc(sqlite.ParseSqlParam)}
	dbdrivers["csv"] = []interface{}{dbms.InvokeDBFunc(dlv.InvokeCSVDB), dbms.ParseSqlArgFunc(dlv.ParseSqlParam)}
	dbdrivers["dlv"] = []interface{}{dbms.InvokeDBFunc(dlv.InvokeDLVDB), dbms.ParseSqlArgFunc(dlv.ParseSqlParam)}

	glbldbms.Drivers().DefaultInvokable(func(driver string) (InvokeDB dbms.InvokeDBFunc, ParseSqlParam dbms.ParseSqlArgFunc) {
		if dbdvrapi, dbdrvapik := dbdrivers[driver]; dbdrvapik {
			InvokeDB, _ = dbdvrapi[0].(dbms.InvokeDBFunc)
			ParseSqlParam, _ = dbdvrapi[1].(dbms.ParseSqlArgFunc)
		}
		return
	})
	glbldbms.Drivers().Register("csv", mltyfsys)
	if mltyfsys.Exist("/embedding/embed.html") {
		glbldbms.Connections().Register("datafiles", "csv", "/datafiles")
		fmt.Println(time.Now())
		if rdr, rdrerr := glbldbms.Query("datafiles", "simulations.csv", map[string]interface{}{"ColDelim": ",", "Trim": true}, mltyfsys); rdrerr == nil {
			defer rdr.Close()
			for rc := range rdr.Records() {
				if rc.First() {
					fmt.Println(rc.Columns())
				}
				if rc.Last() {
					fmt.Println(rc.Data())
					fmt.Println(rc.RowNR())
				}
			}
		}
		fmt.Println(time.Now())
	}
	fonts.ImportFonts(mltyfsys)
	ui.ImportUiJS(mltyfsys)
	var lstn listen.Listening

	var mainsession = sessioning.NewSession(nil, mltyfsys)
	var mainsessions = mainsession.Sessions()
	var ssnhndlr sessioning.SessionHttpFunc
	ssnhndlr = sessioning.SessionHttpFunc(func(w http.ResponseWriter, r *http.Request) {
		ssn := ssnhndlr.Session(mainsession, w, r, mltyfsys)
		mainsessions.Set(mainsessions.UniqueKey(), ssn)
		defer func() {
			if ssn != nil {
				ssn.Close()
			}
		}()
		ssn.API(func(sa *sessioning.SessionAPI) {
			sa.InvokeVM = func(s sessioning.Session) sessioning.SessionVM {
				var vm = es.New()
				vm.SetFieldNameMapper(fieldmapping.NewFieldMapper(es.UncapFieldNameMapper()))
				vm.SetImportModule(func(modname string, namedimports ...[][]string) (imported bool) {
					if modfi := mltyfsys.Stat(modname); modfi != nil {
						active.ProcessActiveFile(mltyfsys, modfi, nil, nil, func(pgrm interface{}, w io.Writer) {
							imported = es.ImportModule(pgrm, vm, namedimports...)
						})
					}
					return imported
				})
				vm.SetRequire(func(modname string) (obj *es.Object) {
					obj = es.RequireModuleExports(nil, vm)
					return
				})
				vm.Set("$", ssn)

				ssnvm := sessioning.InvokeVM(vm, ssn)
				vm.Set("print", ssnvm.Print)
				vm.Set("println", ssnvm.Println)
				return ssnvm
			}
			sa.RunProgram = func(prgrm interface{}, prgout io.Writer) {
				if vm := ssn.VM(); vm != nil {
					if eprg, _ := prgrm.(*es.Program); eprg != nil {
						if evm, _ := vm.VM().(*es.Runtime); evm != nil {
							if prgout != nil {
								vm.SetWriter(prgout)
							}
							defer func() {
								if prgout != nil {
									vm.SetWriter(nil)
								}
							}()

							rslt, err := evm.RunProgram(eprg)
							if err != nil {
								if out := ssn.Out(); out != nil {
									out.Println("err:" + err.Error())
									for lnr, ln := range strings.Split(eprg.Src(), "\n") {
										out.Println(fmt.Sprintf("%d. %s", lnr+1, strings.TrimRightFunc(ln, ioext.IsSpace)))
									}
								}
							} else {
								if rslt != nil {
									if exp := rslt.Export(); exp != nil {
										if prgout != nil {
											vm.Print(exp)
										}
									}
								}
							}
						}
					}
				}
			}

			sa.InvodeDB = func() dbms.DBMSHandler {
				return glbldbms.Handler(ssn.Params(), func(dbfsys fs.MultiFileSystem, dbfi fs.FileInfo, qryout io.Writer) {
					if dbfi = active.ProcessActiveFile(
						dbfsys,
						dbfi,
						qryout,
						nil,
						sa.RunProgram); dbfi != nil {
						if qryout != nil {
							ioext.Fprint(qryout, dbfi)
						}
					}
				})
			}

			sa.MarshalEval = func(arg interface{}, a ...interface{}) (result interface{}, err error) {
				bf := ioext.NewBuffer()
				defer bf.Close()
				if err = sa.Eval(arg, append(a, bf)...); err == nil {
					result, err = bf.Marshal()
				}
				return
			}

			sa.Eval = func(arg interface{}, a ...interface{}) (err error) {
				var argmps []map[string]interface{}
				var altout io.Writer
				ai := 0
				al := len(a)
				for ai < al {
					if altoutd, altoutk := a[ai].(io.Writer); altoutk {
						if altout == nil && altoutd != nil {
							altout = altoutd
						}
						a = append(a[:ai], a[ai+1:]...)
						al--
						continue
					}
					if argmp, argmpk := a[ai].(map[string]interface{}); argmpk {
						if len(argmp) > 0 {
							argmps = append(argmps, argmp)
						}
						a = append(a[:ai], a[ai+1:]...)
						al--
						continue
					}
					ai++
				}
				var fi, _ = arg.(fs.FileInfo)
				if fi == nil {
					if s, sk := arg.(string); sk && s != "" {
						if fi = mltyfsys.Stat(s); fi == nil {
							if ext := filepath.Ext(s); ext != "" {
								if fi = mltyfsys.Stat(s); fi == nil {
									return
								}
							} else {
								if fi = mltyfsys.Stat(s + ext); fi == nil {
									return
								}
							}
						}
					}
				}
				if altout == nil {
					if vm := ssn.VM(); vm != nil {
						if altout = vm.Writer(); altout == nil {
							altout = ssn.Out()
						}
					} else {
						altout = ssn.Out()
					}
				}
				if fi != nil {
					if len(argmps) == 0 {
						if prcfi := active.ProcessActiveFile(mltyfsys, fi, altout, nil, sa.RunProgram); prcfi != nil && altout != nil {
							if prnt, prtk := altout.(interface{ Print(...interface{}) error }); prtk {
								if prnt != nil {
									prnt.Print(prcfi)
								}
								return
							}
							ioext.Fprint(altout, prcfi)
						}
						return
					}
					cntnt, cde, _, _, prserr := active.ParseOnly(mltyfsys, fi, argmps...)
					defer func() {
						if !cntnt.Empty() {
							cntnt.Close()
						}
						if !cde.Empty() {
							cde.Close()
						}
					}()
					if prserr != nil {
						err = prserr
					}

					if altout != nil && !cntnt.Empty() {
						cntnt.WriteTo(altout)
					}
					if !cde.Empty() {
						prgm, prgmerr := compileprogram(mltyfsys, cde)
						if prgmerr != nil {
							err = prgmerr
							return
						}
						if prgm != nil {
							sa.RunProgram(prgm, altout)
						}
					}
				}
				return
			}
		})

		path := ssn.Path()
		rqfi := ssn.Fsys().Stat(path)
		if rqfi == nil {
			return
		}
		if cls, _ := rqfi.(io.Closer); cls != nil {
			defer cls.Close()
		}
		if rqsize := rqfi.Size(); rqsize > 0 {
			out := ssn.Out()
			in := ssn.In()
			mimetype, texttype, media := mimes.FindMimeType(rqfi.Ext())
			if texttype {
				if out != nil {
					out.Header().Set("Expires", time.Now().Format(http.TimeFormat))
				}
			}
			if texttype || strings.Contains(mimetype, "text/plain") {
				mimetype += "; charset=utf-8"
			}
			if out != nil {
				out.Header().Set("Content-type", mimetype)
			}
			actv := texttype
			if !actv && rqfi.Active() {
				actv = true
			}
			if actv {
				if ssn != nil {
					if rqfi = active.ProcessActiveFile(
						mltyfsys,
						rqfi,
						out,
						nil,
						ssn.API().RunProgram); rqfi != nil && out != nil && in != nil {
						out.Print(rqfi.Reader(in.Context()))
					}
				}
				return
			}

			if !actv || (media && rqfi.Media()) {
				rdr := rqfi.Reader()
				if rdrsk, rangeOffset, rangeType := rdr.(io.ReadSeeker), in.RangeOffset(), in.RangeType(); rdrsk != nil && rangeOffset > -1 && rangeType == "bytes" {
					rdrsk.Seek(rangeOffset, 0)
					maxoffset := int64(0)
					maxlen := int64(0)
					if maxoffset = rangeOffset + (rqsize - rangeOffset); maxoffset > 0 {
						maxlen = maxoffset - rangeOffset
						maxoffset--
					}

					if maxoffset < rangeOffset {
						maxoffset = rangeOffset
						maxlen = 0
					}
					if maxlen > 1024*1024 {
						maxlen = 1024 * 1024
						maxoffset = rangeOffset + (maxlen - 1)
					}
					contentrange := fmt.Sprintf("%s %d-%d/%d", in.RangeType(), rangeOffset, maxoffset, rqsize)
					if out != nil {
						out.Header().Set("Content-Range", contentrange)
						out.Header().Set("Content-Length", fmt.Sprintf("%d", maxlen))
					}
					rdr = io.LimitReader(rdr, maxlen)
					out.MaxWriteSize(maxlen)
					if out != nil {
						out.WriteHeader(206)
					}
				} else {
					if out != nil {
						out.Header().Set("Accept-Ranges", "bytes")
						out.Header().Set("Content-Length", fmt.Sprintf("%d", rqsize))
					}
					out.MaxWriteSize(rqsize)
				}
				out.BPrint(rdr)
				return
			}
		}
	})
	lstn = listen.NewListen(ssnhndlr.ServeHttp)

	lstn.Serve("tcp", ":1089")
	<-chn
}
