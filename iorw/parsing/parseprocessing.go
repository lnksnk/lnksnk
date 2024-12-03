package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnksnk/lnksnk/fsutils"
	"github.com/lnksnk/lnksnk/iorw"
)

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/' || cr == ':'
}

type contentelem struct {
	modified    time.Time
	fi          fsutils.FileInfo
	elemname    string
	elemroot    string
	elemext     string
	ctntbuf     *iorw.Buffer
	prebuf      *iorw.Buffer
	postbuf     *iorw.Buffer
	runerdr     io.RuneReader
	rawBuf      *iorw.Buffer
	eofevent    func(*contentelem, error)
	attrs       map[string]interface{}
	level       int
	prvctntelem *contentelem
	coresttngs  map[string]interface{}
	pgstngs     map[string]interface{}
}

func (ctntelm *contentelem) writeRune(r rune) {
	if ctntelm != nil {
		if ctntelm.rawBuf != nil {
			ctntelm.rawBuf.WriteRune(r)
			return
		}
		ctntelm.content().WriteRune(r)
	}
}

func (ctntelm *contentelem) content() (ctntbuf *iorw.Buffer) {
	if ctntelm != nil {
		if ctntbuf = ctntelm.ctntbuf; ctntbuf == nil {
			ctntbuf = iorw.NewBuffer()
			ctntelm.ctntbuf = ctntbuf
		}
	}
	return
}

// ReadRune implements io.RuneReader.
func (ctntelm *contentelem) ReadRune() (r rune, size int, err error) {
	if ctntelm != nil {
		if ctntelm.runerdr != nil {
			if r, size, err = ctntelm.runerdr.ReadRune(); err != nil {
				if eofevent := ctntelm.eofevent; eofevent != nil {
					ctntelm.eofevent = nil
					if err == io.EOF {
						eofevent(ctntelm, nil)
						return
					}
					eofevent(ctntelm, err)
				}
			}
			return
		}
		if err = prepairContentElem(ctntelm); err != nil {
			return
		}
		if ctntelm.runerdr != nil {
			if ctntelm.rawBuf == nil {
				ctntelm.rawBuf = iorw.NewBuffer()
			}
			if r, size, err = ctntelm.runerdr.ReadRune(); err != nil {
				if eofevent := ctntelm.eofevent; eofevent != nil {
					ctntelm.eofevent = nil
					if err == io.EOF {
						eofevent(ctntelm, nil)
						return
					}
					eofevent(ctntelm, err)
				}
			}
		}
	}
	if size == 0 && err == nil {
		err = io.EOF
	}
	return
}

func prepairContentElem(ctntelm *contentelem) (err error) {
	if ctntelm != nil && ctntelm.runerdr == nil && ctntelm.fi != nil {

		cntntbuf := ctntelm.ctntbuf
		ctntelm.ctntbuf = nil
		var rdr io.RuneReader = nil
		if r, rerr := ctntelm.fi.Open(); rerr == nil {
			if rdr, _ = (r).(io.RuneReader); rdr == nil {
				rdr = iorw.NewEOFCloseSeekReader(r)
			}
		}
		ctntstngs := map[string]interface{}{}

		//path := ctntelm.fi.Path()
		pathroot := ctntelm.fi.PathRoot()
		root := ctntelm.fi.Root()
		rsroot := ctntelm.fi.RSRoot()
		if rsroot != "" && rsroot[len(rsroot)-1] != '/' {
			rsroot += "/"
		}
		if len(root) < len(rsroot) {
			root = rsroot
		}
		/*fmt.Println("elem:", ctntelm.elemname)
		fmt.Println("path:", path)
		fmt.Println("pathroot:", pathroot)
		fmt.Println("root:", root)
		fmt.Println("rsroot:", rsroot)
		fmt.Println()*/
		/*pthexti, pathpthi := strings.LastIndex(pathroot, "."), strings.LastIndex(pathroot, "/")
		if pathpthi > -1 {
			if pthexti > pathpthi {
				pathroot = pathroot[:pathpthi+1]
			}
			pathroot = pathroot[:pathpthi+1]
		} else {
			pathroot = "/"
		}*/
		//path = path[len(pathroot):]
		/*if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
			root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
		}
		if strings.HasSuffix(ctntelm.elemname, ":") {
			path = ""
		}*/
		coresttngs := ctntelm.coresttngs
		if coresttngs == nil {
			coresttngs = map[string]interface{}{}
			coresttngs["path-root"] = pathroot
			coresttngs["root"] = root
			coresttngs["base-root"] = rsroot
			coresttngs["elem-root"] = func() (elmroot string) {
				/*if path == "" {
					if strings.HasSuffix(pathroot, "/") {
						if pthi := strings.LastIndex(pathroot[:len(pathroot)-1], "/"); pthi > -1 {
							elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
						} else {
							elmroot = ""
						}
					} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
						elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
					} else {
						elmroot = ""
					}
				} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
					elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
				} else {
					elmroot = ""
				}*/
				elmroot = strings.Replace(pathroot, "/", ":", -1)
				return
			}()
			/*coresttngs["elem-path"] = func() (elempath string) {
				elempath = ctntelm.elemname
				if elmpthi := strings.LastIndex(elempath, ":"); elmpthi > 0 {
					elempath = elempath[:elmpthi+1]
					return
				}
				return ":"
			}()*/
			coresttngs["elem-base"] = func() (elembase string) {
				/*elmbases := strings.Split(coresttngs["elem-root"].(string), ":")
				enajst := 0
				for en, elmb := range elmbases {
					if elmb == "" {
						if en == 0 {
							elembase = ":" + elembase
						}
						enajst++
						continue
					}
					if (en + enajst) < len(elmbases)-1 {
						elembase += elmb + ":"
					}
				}*/
				elembase = strings.Replace(rsroot, "/", ":", -1)
				return
			}()
			ctntelm.coresttngs = coresttngs
		}

		attrs := ctntelm.attrs

		var prpbf *iorw.Buffer
		prpbuffer := func() *iorw.Buffer {
			if prpbf != nil {
				return prpbf
			}
			prpbf = iorw.NewBuffer()
			return prpbf
		}

		preprdr := iorw.ReadRunesUntil(rdr, func(prevprasefnd, prasefnd string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, prpflushrdr iorw.SliceRuneReader) (prperr error) {
			if prasefnd == "[#" {
				if !prpbf.Empty() {
					prpbf.Clear()
				}
				if prperr = prpbuffer().Print(untilrdr); prperr != nil {
					if prperr.Error() == "#]" {
						prperr = nil
						if !prpbf.Empty() {
							for attk, attv := range attrs {
								if eql, _ := prpbf.Equals(attk); eql {
									prpflushrdr.PreAppend(valToRuneReader(attv, false))
									return
								}
							}
							for attk, attv := range coresttngs {
								if eql, _ := prpbf.Equals(attk); eql {
									prpflushrdr.PreAppend(valToRuneReader(attv, false))
									return
								}
							}
							for attk, attv := range ctntelm.pgstngs {
								if eql, _ := prpbf.Equals(attk); eql {
									prpflushrdr.PreAppend(valToRuneReader(attv, false))
									return
								}
							}
						}
						return
					}
					if prperr == io.EOF {
						prperr = nil
					}
				}
				return
			}
			if prasefnd == "#]" {
				return fmt.Errorf("%s", prasefnd)
			}
			return
		}, "[#", "#]")
		corefnd := false
		preprdr = iorw.ReadRunesUntil(preprdr, func(prpprevphrase, prpphrase string, prpuntilrdr io.RuneReader, prporgrdr iorw.SliceRuneReader, prporgerr error, prpflush iorw.SliceRuneReader) (prperr error) {
			if prpphrase == "<:_:" {
				corefnd = true
				defer func() {
					corefnd = false
				}()
				tmbbf := iorw.NewBuffer()
				if prperr = tmbbf.Print(prpuntilrdr); prperr != nil {
					if prperr.Error() == ":/>" {
						prperr = nil
						if !tmbbf.Empty() {
							for crk, crv := range coresttngs {
								if eql, _ := tmbbf.Equals(crk); eql {
									prpflush.PreAppend(valToRuneReader(crv, false))
									return
								}
							}
						}
						prperr = nil
					}
					if prperr == io.EOF {
						prperr = nil
					}
					return
				}
				return
			}
			if corefnd {
				return fmt.Errorf("%s", prpphrase)
			}
			prpflush.PreAppend(strings.NewReader(prpphrase))
			return
		}, "<:_:", ":/>")
		if !cntntbuf.Empty() {
			/*if err = cntntbuf.Print(iorw.ReadRunesUntil(cntntbuf.Clone(true).Reader(true), "[:", func(cntprevphrase, cntphrase string, cntuntilrdr io.RuneReader, cntorgrdr iorw.SliceRuneReader, cntorgerr error, cntflushrdr iorw.SliceRuneReader) (cnterr error) {
				if cntphrase == "[:" {
					cntbf, cntbferr := iorw.NewBufferError(iorw.ReadRunesUntil(cntuntilrdr), "::", func(tplprevphrase, tplphrase string, tpluntilrdr io.RuneReader, tplorgrdr iorw.SliceRuneReader, tplorgerr error, tplflushrdr iorw.SliceRuneReader) (tplerr error) {

						return
					})
					if cntbferr != nil {
						if !cntbf.Empty() {

						}
					}
					return
				}
				if cntphrase == ":]" {
					return fmt.Errorf("%s", cntphrase)
				}
				return
			})); err != nil {
				fmt.Println("cntnt-err" + err.Error())
			}*/
		}
		ctntstngs["cntnt"] = cntntbuf
		var ctntstngskeys []string

		for ctntk := range ctntstngs {
			ctntstngskeys = append(ctntstngskeys, "<:"+ctntk+":/>")
		}

		if len(ctntstngskeys) > 0 {
			preprdr = iorw.ReadRunesUntil(preprdr, ctntstngskeys, func(prpprevphrase, prpphrase string, prpuntilrdr io.RuneReader, prporgrdr iorw.SliceRuneReader, prporgerr error, prpflushrdr iorw.SliceRuneReader) (prperr error) {
				prpflushrdr.PreAppendArgs(valToRuneReader(ctntstngs[prpphrase[len("<:"):len(prpphrase)-len(":/>")]], false))
				return
			})
		}

		if ctntelm.elemname == ":etl:ui:calendars:layout" {
			ctntelm.runerdr = preprdr
			return
		}
		ctntelm.runerdr = preprdr
	}
	return
}

// Close implements io.Closer
func (ctntelm *contentelem) Close() (err error) {
	if ctntelm != nil {
		postbuf, prebuf, ctntbuf, rawBuf, attrs := ctntelm.postbuf, ctntelm.prebuf, ctntelm.ctntbuf, ctntelm.rawBuf, ctntelm.attrs
		ctntelm.postbuf = nil
		ctntelm.prebuf = nil
		ctntelm.runerdr = nil
		ctntelm.rawBuf = nil
		ctntelm.fi = nil
		ctntelm.eofevent = nil
		ctntelm.attrs = nil
		ctntelm.pgstngs = nil

		if postbuf != nil {
			postbuf.Close()
		}
		if prebuf != nil {
			prebuf.Close()
		}
		if ctntbuf != nil {
			ctntbuf.Close()
			ctntbuf = nil
		}
		if rawBuf != nil {
			rawBuf.Close()
		}

		for _, atv := range attrs {
			if atvbf, _ := atv.(*iorw.Buffer); atvbf != nil {
				atvbf.Close()
			}
		}
	}
	return
}

type ctntelemlevel int

const (
	ctntElemUnknown ctntelemlevel = iota
	ctntElemStart
	ctntElemSingle
	ctntElemEnd
)

func (ctntelmlvl ctntelemlevel) String() string {
	if ctntelmlvl == ctntElemStart {
		return "start"
	}
	if ctntelmlvl == ctntElemSingle {
		return "single"
	}
	if ctntelmlvl == ctntElemEnd {
		return "end"
	}
	return "unknown"
}

type codeReadMode int

const (
	codeReadingCode codeReadMode = iota
	codeReadingContent
)

func valToRuneReader(val interface{}, clear bool) io.RuneReader {
	if s, _ := val.(string); s != "" {
		return strings.NewReader(s)
	}
	if int32s, _ := val.([]int32); len(int32s) > 0 {
		rns := make([]rune, len(int32s))
		copy(rns, int32s)
		return iorw.NewRunesReader(rns...)
	}
	if bf, _ := val.(*iorw.Buffer); bf != nil {
		return bf.Clone(clear).Reader(true)
	}
	return nil
}
func internalProcessParsing(
	capturecache func(fullpath string, pathModified time.Time, cachedpaths map[string]time.Time, prsdpsv, prsdatv *iorw.Buffer, preppedatv interface{}) (cshderr error),
	pathModified time.Time,
	path, pathroot, pathext string,
	out io.Writer,
	fs *fsutils.FSUtils,
	invertActive bool,
	evalcode func(...interface{}) (interface{}, error),
	rnrdrs ...io.RuneReader) (prsngerr error) {
	fullpath := pathroot + path
	validelempaths := map[string]time.Time{}
	invalidelempaths := map[string]bool{}

	if len(rnrdrs) == 0 {
		return
	}

	root := pathroot
	if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
		root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
	}
	pgsttngs := map[string]interface{}{}
	pgsttngs["pg-path-root"] = pathroot
	pgsttngs["pg-root"] = root

	var elempath = func() (elmroot string) {
		if path == "" {
			if strings.HasSuffix(pathroot, "/") {
				if pthi := strings.LastIndex(pathroot[:len(pathroot)-1], "/"); pthi > -1 {
					elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
				} else {
					elmroot = ""
				}
			} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
				elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
			} else {
				elmroot = ""
			}
		} else if pthi := strings.LastIndex(pathroot, "/"); pthi > -1 {
			elmroot = strings.Replace(pathroot[:pthi+1], "/", ":", -1)
		} else {
			elmroot = ""
		}
		return
	}()
	pgsttngs["pg-elem-root"] = elempath
	pgsttngs["pg-elem-base"] = func() (elembase string) {
		elmbases := strings.Split(elempath, ":")
		enajst := 0
		for en, elmb := range elmbases {
			if elmb == "" {
				if en == 0 {
					elembase = ":" + elembase
				}
				enajst++
				continue
			}
			if (en + enajst) < len(elmbases)-1 {
				elembase += elmb + ":"
			}
		}
		return
	}()
	pgstngl := 0
	for pgk := range pgsttngs {
		if pgkl := len(pgk); pgkl > pgstngl {
			pgstngl = pgkl
		}
	}
	if invertActive {
		rnrdrs = append([]io.RuneReader{strings.NewReader("<@")}, append(rnrdrs, strings.NewReader("@>"))...)
	}

	var phrsbf *iorw.Buffer = nil
	var pgphrs = map[string]string{"<:_:": ":/>", "[#": "#]"}
	var ctntinitrplcrdr = iorw.ReadRunesUntil(iorw.NewSliceRuneReader(rnrdrs...), pgphrs, iorw.RunesUntilFunc(func(prevphrasefnd, phrasefnd string, untilrdr io.RuneReader, orgrd iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) error {
		if phrasefnd == "<:_:" || phrasefnd == "[#" {
			if phrsbf == nil {
				phrsbf = iorw.NewBuffer()
			}
			if prevphrasefnd == phrasefnd {
				flushrdr.PreAppendArgs(phrasefnd)
				return nil
			}
			phrsbf.Clear()
			if _, phrserr := phrsbf.ReadRunesFrom(untilrdr); phrserr != nil {
				if phrserr == io.EOF {
					if _, pgok := pgphrs[phrasefnd]; pgok {
						if !phrsbf.Empty() {
							flushrdr.PreAppendArgs(phrasefnd, phrsbf.Clone(true).Reader(true))
							return nil
						}
						flushrdr.PreAppendArgs(phrasefnd)
						return nil
					}
					if !phrsbf.Empty() {
						flushrdr.PreAppendArgs(phrasefnd)
						return nil
					}
					return phrserr
				}
				if prhse := phrserr.Error(); pgphrs[phrasefnd] == prhse {
					if phrsl := phrsbf.Size(); phrsl <= int64(pgstngl) {
						for thisk, thisv := range pgsttngs {
							if qkl, _ := phrsbf.Equals(thisk); qkl {
								flushrdr.PreAppend(valToRuneReader(thisv, true))
								return nil
							}
						}
					}
					flushrdr.PreAppendArgs(phrasefnd, phrsbf.Clone(true).Reader(true), prhse)
					return nil
				}
				if phrserr == orgerr {

					return orgerr
				}
			}
			return nil
		}
		if prevphrasefnd != phrasefnd && pgphrs[prevphrasefnd] == phrasefnd {
			return fmt.Errorf("%s", phrasefnd)
		}
		flushrdr.PreAppendArgs(phrasefnd)
		return nil
	}))

	var crntnextelm *contentelem = nil
	var elemlevels = []*contentelem{}

	var addelemlevel = func(fi fsutils.FileInfo, elemname string, elemext string) (elmnext *contentelem) {
		elmnext = &contentelem{
			modified: fi.ModTime(),
			fi:       fi,
			elemname: elemname,
			elemroot: elemname[:strings.LastIndex(elemname, ":")+1],
			elemext:  elemext,
			pgstngs:  pgsttngs,
		}
		validelempaths[fi.Path()] = fi.ModTime()
		elmnext.level = len(elemlevels)
		if elmnext.level > 0 {
			elmnext.prvctntelem = elemlevels[elmnext.level-1]
		}
		elemlevels = append([]*contentelem{elmnext}, elemlevels...)
		return
	}
	var nextfullname = func(elemname string, elmlvl ctntelemlevel) (fullname string) {
		if elemname[0:1] == ":" {
			return elemname
		}

		if crntnextelm != nil {
			if elmlvl == ctntElemEnd {
				if ctntelmpnti, elml, al := strings.LastIndex(crntnextelm.elemname, "."), len(elemname), len(elemlevels); al > 0 && elemlevels[0] == crntnextelm && ((ctntelmpnti == -1 && strings.HasSuffix(crntnextelm.elemname, elemname)) || (len(crntnextelm.elemname[:ctntelmpnti+1]) < elml && strings.HasSuffix(crntnextelm.elemname, elemname))) {
					return crntnextelm.elemname
				}
			}
			return crntnextelm.elemroot + elemname
		}
		return elempath + elemname
	}

	chkng := false
	rdngval := false
	var argbf *iorw.Buffer
	argbuffer := func() *iorw.Buffer {
		if argbf != nil {
			return argbf
		}
		argbf = iorw.NewBuffer()
		return argbf
	}

	ctntprsrdr := iorw.ReadRunesUntil(iorw.NewSliceRuneReader(ctntinitrplcrdr), "<", ">", func(prevphrase, phrase string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) (fnderr error) {
		if rdngval {
			orgrdr.PreAppend(strings.NewReader(phrase))
			return
		}
		if phrase == "<" {
			invalfnd := false
			var chkbf *iorw.Buffer
			var elmargs map[string]interface{}
			defer func() {
				for ark, arv := range elmargs {
					if arbv, _ := arv.(*iorw.Buffer); arbv != nil {
						arbv.Close()
						arbv = nil
					}
					delete(elmargs, ark)
				}
			}()
			formatElmArgVal := func(val *iorw.Buffer) *iorw.Buffer {
				var eofargs []interface{}
				for elmk := range func() map[string]interface{} {
					if crntnextelm != nil {
						return crntnextelm.attrs
					}
					return nil
				}() {
					eofargs = append(eofargs, "#"+elmk+"#")
				}
				if len(eofargs) > 0 {
					eofargs = append(eofargs, "#")
					srchng := false
					eofargs = append(eofargs, func(rplcprevphrase, rplcphrase string, rplcuntilrdr io.RuneReader, rplcorgrdr iorw.SliceRuneReader, rplcorgerr error, rplcflushrdr iorw.SliceRuneReader) (rplcfnderr error) {
						if rplcphrase == "#" {
							if srchng {
								return fmt.Errorf("%s", rplcphrase)
							}
							srchng = true
							var srchbf *iorw.Buffer
							if srchbf, rplcfnderr = iorw.NewBufferError(rplcuntilrdr); rplcfnderr != nil {
								if rplcfnderr.Error() == "#" {
									rplcfnderr = nil
									srchbf.Clear()
									return
								}
								if rplcfnderr != io.EOF {
									return
								}
								rplcfnderr = nil
							}
							if !srchbf.Empty() {
								rplcflushrdr.PreAppend(srchbf.Reader(true))
							}
							return
						}
						if rplcval, rplcok := crntnextelm.attrs[rplcphrase[1:len(rplcphrase)-2]]; rplcok {
							rplcflushrdr.PreAppend(valToRuneReader(rplcval, true))
							return
						}

						return
					})
					val.Print(iorw.ReadRunesUntil(val.Clone(true).Reader(true), eofargs...))
				}
				return val
			}
			setElmArgVal := func(argk string, argv interface{}) {
			setargv:
				if elmargs != nil {
					elmargs[argk] = argv
					return
				}
				elmargs = map[string]interface{}{}
				goto setargv
			}
			chkng = true
			defer func() {
				chkng = false
			}()
			chkbfrns := append([]rune{}, []rune(phrase)...)
			rdngval = false
			elmname := ""
			prpelmname := func() bool {
				if elmname != "" {
					if !strings.Contains(elmname, "::") {
						return true
					}
					spltelmname := strings.Split(elmname, "::")
					if spltl := len(spltelmname); spltl >= 2 {
						for spn, spnme := range spltelmname {
							if spn > 0 && spn < spltl-1 {
								if tstk := spnme; tstk != "" {
									if crntnextelm != nil {
										if coresttngs := crntnextelm.coresttngs; len(coresttngs) > 0 {
											if crv, crok := coresttngs[tstk]; crok {
												ts, _ := crv.(string)
												spltelmname[spn] = ts
												continue
											}
										}
									}
									if crv, crok := pgsttngs[tstk]; crok {
										ts, _ := crv.(string)
										spltelmname[spn] = ts
										continue
									}
								}
								return false
							}
						}
						elmname = strings.Join(spltelmname, "")
					}
					for _, cr := range elmname {
						if ('a' <= cr && cr <= 'z') || ('A' <= cr && cr <= 'Z') {
							return true
						}
					}
				}
				return false
			}
			elmlvl := ctntElemUnknown
			prvr := rune(0)
			var argnmerns []rune
			var ctntargsrdr io.RuneReader
			if fnderr = iorw.EOFReadRunes(untilrdr, func(r rune, size int) (rderr error) {
				chkbfrns = append(chkbfrns, r)
				if r == '/' {
					if elmlvl == ctntElemUnknown {
						elmlvl = ctntElemEnd
						prvr = 0
						return
					}
					if elmlvl == ctntElemStart {
						elmlvl = ctntElemSingle
						prvr = 0
						return
					}
					return fmt.Errorf("failed")
				}
				if iorw.IsSpace(r) {
					if elmlvl == ctntElemUnknown {
						return fmt.Errorf("failed")
					}
					return fmt.Errorf("prepeof")
				}
				if validElemChar(prvr, r) {
					if elmlvl == ctntElemUnknown {
						elmlvl = ctntElemStart
					}
					elmname += string(r)
					prvr = r
					return
				}
				return fmt.Errorf("failed")
			}); fnderr != nil {
				if fnderr.Error() == ">" {
					chkbfrns = append(chkbfrns, []rune(fnderr.Error())...)
					prvelmnme := elmname
					if elmlvl == ctntElemUnknown || !prpelmname() {
						if prvelmnme != elmname {
							chkbfrns = []rune(strings.Replace(string(chkbfrns), prvelmnme, elmname, -1))
						}
						flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
						return nil
					}
					if prvelmnme != elmname {
						chkbfrns = []rune(strings.Replace(string(chkbfrns), prvelmnme, elmname, -1))
					}
					fnderr = nil
					goto fndeof
				}
				if fnderr.Error() == "prepeof" {
					fnderr = nil
					prvelmnme := elmname
					if prpelmname() {
						if prvelmnme != elmname {
							chkbfrns = []rune(strings.Replace(string(chkbfrns), prvelmnme, elmname, -1))
						}
						goto prepeof
					}
					if prvelmnme != elmname {
						chkbfrns = []rune(strings.Replace(string(chkbfrns), prvelmnme, elmname, -1))
					}
					flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
					return nil
				}
				if fnderr.Error() == "failed" || fnderr == io.EOF {
					flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
					return nil
				}
			}
			flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
			return nil
		prepeof:
			invalfnd = invalidelempaths[nextfullname(elmname, elmlvl)]
			if !chkbf.Empty() {
				chkbf.Clear()
			}

			prvr = 0
			ctntargsrdr = iorw.ReadRunesUntil(iorw.ReadRuneFunc(func() (ar rune, asize int, aerr error) {
				ar, asize, aerr = orgrdr.ReadRune()
				if asize > 0 && (aerr == nil || aerr == io.EOF) {
					if chkbf != nil {
						chkbf.WriteRune(ar)
						return
					}
					chkbf = iorw.NewBuffer()
					chkbf.WriteRune(ar)
				}
				return
			}), "/", ">", "=", `='`, `="`, `=[$`, `[$`, func(argsprevphrase, argsphrase string, argsuntilrdr io.RuneReader, argsorgrdr iorw.SliceRuneReader, argsorgerr error, argsflushrdr iorw.SliceRuneReader) (argserr error) {
				if argsphrase == `='` || argsphrase == `="` {
					rdngval = true
					defer func() {
						rdngval = false
					}()
					argtxtpar := func() string {
						if argsphrase == `='` {
							return `'`
						}
						return `"`
					}()
					if len(argnmerns) == 0 {
						return fmt.Errorf("%s", "failed")
					}
					if !argbf.Empty() {
						argbf.Clear()
					}
					if _, argserr = argbuffer().ReadRunesFrom(iorw.ReadRunesUntil(argsorgrdr, argtxtpar, iorw.RunesUntilFunc(func(txtprevphrasefnd, txtphrasefnd string, txtuntilrdr io.RuneReader, txtorgrd iorw.SliceRuneReader, txtorgerr error, txtflushrdr iorw.SliceRuneReader) error {
						if txtphrasefnd == argtxtpar {
							return fmt.Errorf("%s", txtphrasefnd)
						}
						return nil
					}))); argserr != nil {
						if argserr.Error() == argtxtpar {
							argserr = nil
							if invalfnd {
								argnmerns = nil
								return
							}
							if argbf.Empty() {
								setElmArgVal(string(argnmerns), "")
								argnmerns = nil
								return
							}
							setElmArgVal(string(argnmerns), formatElmArgVal(argbf.Clone(true)))
							argnmerns = nil
							return
						}
						return fmt.Errorf("failed")
					}
					return
				}
				if argsphrase == "=" {
					if len(argnmerns) == 0 {
						return fmt.Errorf("%s", "failed")
					}
					return fmt.Errorf("%s", "failed")
				}
				if argsphrase == `[$` {
					if rdngval || len(argnmerns) > 0 {
						argsflushrdr.PreAppend(strings.NewReader(`[$`))
						return
					}
					rdngval = true
					defer func() {
						rdngval = false
					}()
					if !argbf.Empty() {
						argbf.Clear()
					}
					if _, argserr = argbuffer().ReadRunesFrom(iorw.ReadRunesUntil(argsorgrdr, "$]", iorw.RunesUntilFunc(func(txtprevphrasefnd, txtphrasefnd string, txtuntilrdr io.RuneReader, txtorgrd iorw.SliceRuneReader, txtorgerr error, txtflushrdr iorw.SliceRuneReader) error {
						if txtphrasefnd == "$]" {
							return fmt.Errorf("%s", txtphrasefnd)
						}
						return nil
					}))); argserr != nil {
						if argserr.Error() == "$]" {
							argserr = nil
							if invalfnd {
								return
							}
							if !argbf.Empty() {
								setElmArgVal("pre", formatElmArgVal(argbf.Clone(true)))
							}
							return
						}
						return fmt.Errorf("failed")
					}
					return
				}
				if argsphrase == `=[$` {
					if len(argnmerns) == 0 {
						return fmt.Errorf("%s", "failed")
					}
					rdngval = true
					defer func() {
						rdngval = false
					}()
					if !argbf.Empty() {
						argbf.Clear()
					}
					if _, argserr = argbuffer().ReadRunesFrom(iorw.ReadRunesUntil(argsorgrdr, "$]", iorw.RunesUntilFunc(func(txtprevphrasefnd, txtphrasefnd string, txtuntilrdr io.RuneReader, txtorgrd iorw.SliceRuneReader, txtorgerr error, txtflushrdr iorw.SliceRuneReader) error {
						if txtphrasefnd == "$]" {
							return fmt.Errorf("%s", txtphrasefnd)
						}
						return nil
					}))); argserr != nil {
						if argserr.Error() == "$]" {
							argserr = nil
							if invalfnd {
								argnmerns = nil
							}
							if argbf.Empty() {
								setElmArgVal(string(argnmerns), "")
								argnmerns = nil
								return
							}
							setElmArgVal(string(argnmerns), formatElmArgVal(argbf.Clone(true)))
							argnmerns = nil
							return
						}
						return fmt.Errorf("failed")
					}
					return
				}
				if argsphrase == "/" {
					if len(argnmerns) > 0 {
						return fmt.Errorf("%s", "failed")
					}
					if elmlvl == ctntElemStart {
						elmlvl = ctntElemSingle
						return
					}
					return fmt.Errorf("%s", "failed")
				}
				if argsphrase == ">" {
					if len(argnmerns) > 0 {
						return fmt.Errorf("%s", "failed")
					}
					return fmt.Errorf("%s", argsphrase)
				}
				return
			})
			if fnderr = iorw.EOFReadRunes(ctntargsrdr, func(r rune, size int) (rderr error) {
				if rdngval {
					return
				}
				if iorw.IsSpace(r) {
					if len(argnmerns) != 0 {
						return fmt.Errorf("%s", "failed")
					}
					return
				}
				if validElemChar(prvr, r) {
					argnmerns = append(argnmerns, r)
					prvr = r
					return
				}
				return fmt.Errorf("%s", "failed")
			}); fnderr != nil {
				if fnderr.Error() == "failed" {
					if !chkbf.Empty() {
						flushrdr.PreAppend(chkbf.Clone(true).Reader(true))
					}
					flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
					return nil
				}
				if fnderr == io.EOF {

					if !chkbf.Empty() {
						flushrdr.PreAppend(chkbf.Clone(true).Reader(true))
					}
					flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
					return nil
				}
				if fnderr.Error() == ">" {
					fnderr = nil
				}
			}
			if invalfnd {
				if !chkbf.Empty() {
					flushrdr.PreAppend(formatElmArgVal(chkbf.Clone(true)).Reader(true))
				}
				flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
				return nil
			}
		fndeof:
			var fi fsutils.FileInfo = nil
			fullelemname := nextfullname(elmname, elmlvl)
			if invalidelempaths[fullelemname] {
				if !chkbf.Empty() {
					flushrdr.PreAppend(formatElmArgVal(chkbf.Clone(true)).Reader(true))
				}
				flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
				return nil
			}

			if elmlvl == ctntElemStart || elmlvl == ctntElemSingle {
				testpath := strings.Replace(fullelemname, ":", "/", -1)
				testext := filepath.Ext(testpath)
				if testext != "" {
					testpath = testpath[:len(testpath)-len(testext)]
				}

				if fi = func() fsutils.FileInfo {
					if fs == nil {
						return nil
					}
					if testext == "" {
						testext = pathext
					}
					if fullelemname[len(fullelemname)-1] == ':' {
						for _, nextpth := range []string{testext, ".js", ".html"} {
							if nextpth != "" && nextpth[0:1] == "." {
								if fios := fs.LS(testpath + "index" + nextpth); len(fios) == 1 {
									return fios[0]
								}
								if testpath[0] == '/' && !strings.HasPrefix(testpath, pathroot) {
									if fios := fs.LS(pathroot + testpath[1:] + "index" + nextpth); len(fios) == 1 {
										return fios[0]
									}
								}
							}
						}
					}
					if fios := fs.LS(testpath + testext); len(fios) == 1 {
						return fios[0]
					}
					if testpath[0] == '/' && !strings.HasPrefix(testpath, pathroot) {
						if fios := fs.LS(pathroot + testpath[1:] + testext); len(fios) == 1 {
							return fios[0]
						}
					}
					return nil
				}(); fi == nil {
					invalidelempaths[fullelemname] = true
					if !chkbf.Empty() {
						flushrdr.PreAppend(formatElmArgVal(chkbf.Clone(true)).Reader(true))
					}
					flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
					return nil
				}
				crntnextelm = addelemlevel(fi, fullelemname, fi.PathExt())

				for argk, argv := range elmargs {
					if crntnextelm.attrs == nil {
						crntnextelm.attrs = map[string]interface{}{}
					}
					crntnextelm.attrs[argk] = argv
					delete(elmargs, argk)
				}
				if elmlvl == ctntElemSingle {
					crntnextelm.eofevent = func(crntelm *contentelem, elmerr error) {
						if elmerr == nil {
							if rawBuf := crntelm.rawBuf; !rawBuf.Empty() {
								orgrdr.PreAppend(rawBuf.Clone(true).Reader(true))
							}
							crntelm.Close()
							crntnextelm = nil
							if elemlvlL := len(elemlevels); elemlvlL > 0 {
								elemlevels = elemlevels[1:]
								if elemlvlL > 1 {
									crntnextelm = elemlevels[0]
									return
								}
							}
							return
						}
						fnderr = elmerr
					}
					orgrdr.PreAppend(crntnextelm)
				}
				return
			}
			if elmlvl == ctntElemEnd {
				if crntnextelm != nil && crntnextelm.elemname == fullelemname {
					crntnextelm.eofevent = func(crntelm *contentelem, elmerr error) {
						if elmerr == nil {
							if rawBuf := crntelm.rawBuf; !rawBuf.Empty() {
								orgrdr.PreAppend(rawBuf.Clone(true).Reader(true))
							}
							crntelm.Close()
							crntnextelm = nil
							if elemlvlL := len(elemlevels); elemlvlL > 0 {
								elemlevels = elemlevels[1:]
								if elemlvlL > 1 {
									crntnextelm = elemlevels[0]
									return
								}
							}
							return
						}
						fnderr = elmerr
					}
					orgrdr.PreAppend(crntnextelm)
					return
				}
				if !chkbf.Empty() {
					flushrdr.PreAppend(formatElmArgVal(chkbf.Clone(true)).Reader(true))
				}
				flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
				return nil
			}
			return
		}
		if chkng {
			return fmt.Errorf("%s", phrase)
		}
		flushrdr.PreAppend(strings.NewReader(phrase))
		return nil
	})

	var cdebuf *iorw.Buffer = nil
	var codebuffer = func() *iorw.Buffer {
		if cdebuf != nil {
			return cdebuf
		}
		cdebuf = iorw.NewBuffer()
		return cdebuf
	}
	defer cdebuf.Close()
	var ctntbuf *iorw.Buffer
	var ctntbuffer = func() *iorw.Buffer {
		if ctntbuf != nil {
			return ctntbuf
		}
		ctntbuf = iorw.NewBuffer()
		return ctntbuf
	}
	defer ctntbuf.Close()
	var chdctntbuf *iorw.Buffer
	var chdctntbuffer = func() *iorw.Buffer {
		if chdctntbuf != nil {
			return chdctntbuf
		}
		chdctntbuf = iorw.NewBuffer()
		return chdctntbuf
	}
	cdelstr := rune(0)
	fndcode := false
	cderdmode := func() codeReadMode {
		if invertActive {
			return codeReadingCode
		}
		return codeReadingContent
	}()

	cdetxtr := rune(0)
	cdeprvr := rune(0)

	var precodernsrdr io.RuneReader = iorw.ReadRuneFunc(func() (r rune, size int, err error) {
	reread:
		r, size, err = ctntprsrdr.ReadRune()
		if size > 0 && (err == nil || err == io.EOF) {
			if crntnextelm != nil {
				crntnextelm.writeRune(r)
				goto reread
			}
		}
		return
	})

	var cderns []rune = make([]rune, 8192)
	defer func() {
		cderns = nil
	}()
	var cdernsi = 0

	var chdrns []rune = make([]rune, 8192)
	defer func() {
		chdrns = nil
	}()
	var chdrnsi = 0
	var ctntflush func(flscde ...bool) (flsherr error)
	var ctntrplc = map[string]string{"${": "\\${", "`": "\\`", `"\`: `"\\`}
	ctntflush = func(flscde ...bool) (flsherr error) {
		if len(flscde) == 1 && flscde[0] {
			if fndcode && cdernsi > 0 {
				cderi := cdernsi
				cdernsi = 0
				if !chdctntbuf.Empty() {
					chdctntbuf.WriteTo(ctntbuffer())
					chdctntbuf.Clear()
					ctntflush()
				}
				codebuffer().WriteRunes(cderns[:cderi]...)
			}
		}
		if chdrnsi > 0 && !fndcode {
			chdctntbuffer().WriteRunes(chdrns[:chdrnsi]...)
			chdrnsi = 0
		}
		if chdrnsi > 0 && fndcode {
			ctntbuffer().WriteRunes(chdrns[:chdrnsi]...)
			chdrnsi = 0
		}
		if !ctntbuf.Empty() {
			if fndcode && cdernsi > 0 {
				codebuffer().WriteRunes(cderns[:cdernsi]...)
				cdernsi = 0
			}
			defer ctntbuf.Clear()
			hstmpltfx := ctntbuf.HasPrefix("`") && ctntbuf.HasSuffix("`")

			var cntntrdr io.RuneReader = nil
			if hstmpltfx {
				cntntrdr = ctntbuf.Clone(true).Reader(true)
			} else if contains, found := ctntbuf.ContainsAny("${", "`", `"\`); contains {
				cntntrdr = iorw.ReadRunesUntil(ctntbuf.Clone(true).Reader(true), found,
					func(prevphrase, phrase string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) error {
						flushrdr.PreAppendArgs(ctntrplc[phrase])
						return nil
					})
			} else {
				cntntrdr = ctntbuf.Clone(true).Reader(true)
			}
			if cdelstr > 0 {
				cdelstr = 0
				if hstmpltfx {
					if flsherr = codebuffer().Print(cntntrdr); flsherr != nil {
						if flsherr != io.EOF {
							return
						}
						flsherr = nil
					}
					return
				}
				if flsherr = codebuffer().Print("`", cntntrdr, "`"); flsherr != nil {
					if flsherr != io.EOF {
						return
					}
					flsherr = nil
				}
				return
			}
			if hstmpltfx {
				if flsherr = codebuffer().Print("print(", cntntrdr, ");"); flsherr != nil {
					if flsherr != io.EOF {
						return
					}
					flsherr = nil
				}
				return
			}
			if flsherr = codebuffer().Print("print(`", func() (s string) {
				s, _ = iorw.ReaderToString(iorw.NewBuffer(cntntrdr).Reader(true))
				return
			}(), "`);"); flsherr != nil {
				if flsherr != io.EOF {
					return
				}
				flsherr = nil
			}
			return
			/*hstmpltfx := ctntbuf.HasPrefix("`") && ctntbuf.HasSuffix("`") && cdepsvs >= 2
			cntsinlinebraseortmpl := !hstmpltfx && ctntbuf.Contains("${") || ctntbuf.Contains("`")
			var psvrdr io.RuneReader = func() io.RuneReader {
				if hstmpltfx {
					return ctntbuf.Clone(true).Reader(true)
				}
				if cntsinlinebraseortmpl {
					return iorw.NewReplaceRuneReader(ctntbuf.Clone(true).Reader(true), "`", "\\`", "${", "\\${")
				}
				return iorw.NewReplaceRuneReader(ctntbuf.Clone(true).Reader(true), `"\`, `"\\`)
			}()

			if cdelstr > 0 {
				cdelstr = 0
				if hstmpltfx {
					codebuffer().Print(psvrdr)
					return
				}
				if cntsinlinebraseortmpl {
					codebuffer().Print("`", psvrdr, "`")
					return
				}
				codebuffer().Print("`", psvrdr, "`")
				return
			}
			if hstmpltfx {
				codebuffer().Print("print(", psvrdr, ");")
				return
			}
			if cntsinlinebraseortmpl {
				codebuffer().Print("print(`", psvrdr, "`);")
				return
			}
			codebuffer().Print("print(`", psvrdr, "`);")*/
		}
		return
	}

	readCode := func(untilrdr io.RuneReader) (rderr error) {
		rderr = iorw.EOFReadRunes(untilrdr, func(r rune, size int) error {
			if size > 0 {
				ctntflush()
				if !fndcode {
					if !chdctntbuf.Empty() {
						chdctntbuf.WriteTo(ctntbuffer())
						chdctntbuf.Clear()
						ctntflush()
					}
					fndcode = true
				}
				cderns[cdernsi] = r
				cdernsi++
				if cdernsi == 8192 {
					codebuffer().WriteRunes(cderns[:cdernsi]...)
					cdernsi = 0
				}
				if cdetxtr == 0 {
					if cdeprvr != '\\' && iorw.IsTxtPar(r) {
						cdetxtr = r
						cdelstr = 0
						return nil
					}
					if !iorw.IsSpace(r) {
						if validLastCdeRune(r) {
							cdelstr = r
							return nil
						}
						cdelstr = 0
						return nil
					}
					cdeprvr = r
					return nil
				}
				if cdeprvr != '\\' && cdetxtr == r {
					cdetxtr = 0
					cdelstr = 0
					return nil
				}
				return nil
			}
			return nil
		})
		return
	}

	coderunsrdr := iorw.ReadRunesUntil(precodernsrdr, "<@", "@>", func(prevphrase, phrase string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) error {
		if phrase == "<@" {
			if cderdmode == codeReadingContent {
				cderdmode = codeReadingCode
			}
			if rderr := readCode(untilrdr); rderr != nil {
				if rderr.Error() == "@>" {
					cderdmode = codeReadingContent
					return nil
				}
				return rderr
			}
			return nil
		}
		if phrase == "@>" {
			if cderdmode == codeReadingCode {
				return fmt.Errorf("%s", phrase)
			}
		}
		return nil
	}, func() {

	}, func(prvr, r rune) bool {
		if cderdmode == codeReadingCode {
			return cdetxtr == 0
		}
		return true
	})

	prsngerr = iorw.EOFReadRunes(coderunsrdr, func(cr rune, csize int) (cerr error) {
		if csize > 0 {
			chdrns[chdrnsi] = cr
			chdrnsi++
			if chdrnsi == 8192 {
				chdrnsi = 0
				if fndcode {
					ctntbuffer().WriteRunes(chdrns[:8192]...)
					return
				}
				chdctntbuffer().WriteRunes(chdrns[:8192]...)
			}
			return
		}
		return
	})
	//}
	if prsngerr != nil {
		if prsngerr != io.EOF {
			return
		}
		prsngerr = nil
	}

	ctntflush(true)
	var chdpgrm interface{} = nil
	if !chdctntbuf.Empty() && cdebuf.Empty() {
		//DefaultMinifyPsv(pathext, chdctntbuf, nil)
		if capturecache != nil {
			prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
		}
	}
	if !chdctntbuf.Empty() {
		if out != nil {
			if _, prsngerr = chdctntbuf.WriteTo(out); prsngerr != nil {
				return
			}
		}
	}

	if !cdebuf.Empty() {
		if DefaultMinifyCde != nil {
			//prsngerr = DefaultMinifyCde(".js", cdebuf, nil)
		}
		if evalcode != nil && prsngerr == nil {
			var evalresult interface{} = nil
			evalresult, prsngerr = evalcode(cdebuf.Reader(), func(prgm interface{}, prsccdeerr error, cmpleerr error) {
				if cmpleerr == nil && prsccdeerr == nil {
					chdpgrm = prgm
				}
				if prsccdeerr != nil {
					prsngerr = prsccdeerr
				}
				if cmpleerr != nil {
					prsngerr = cmpleerr
				}
				if prsngerr == nil {
					if capturecache != nil {
						go func() {
							//prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
							prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
						}()
					}
				}
			})
			if prsngerr == nil {
				if pathext == ".json" {
					if out != nil {
						if evalresult != nil {
							json.NewEncoder(out).Encode(&evalresult)
						}
					}
					return
				}
				if out != nil {
					if evalresult != nil {
						iorw.Fbprint(out, evalresult)
					}
				}
			}
		}
		if prsngerr != nil {
			println(fullpath, ":=> ", prsngerr.Error())
			println()
			if cderr, _ := prsngerr.(CodeError); cderr != nil {
				for ln, cdeln := range strings.Split(cderr.Code(), "\n") {
					println((ln + 1), strings.TrimFunc(cdeln, iorw.IsSpace))
				}
				println()
			} else {
				for ln, cdeln := range strings.Split(cdebuf.String(), "\n") {
					println((ln + 1), strings.TrimFunc(cdeln, iorw.IsSpace))
				}
				println()
			}
		}
	}
	return
}

type CodeError interface {
	error
	Code() string
}

func validElmchar(cr rune) bool {
	return ('a' <= cr && cr <= 'z') || ('A' <= cr && cr <= 'Z') || cr == ':' || cr == '.' || cr == '-' || cr == '_' || ('0' <= cr && cr <= '9')
}

func validElemChar(prvr, r rune) (valid bool) {
	if prvr > 0 {
		valid = validElmchar(prvr) && validElmchar(r)
		return
	}
	valid = validElmchar(r)
	return
}
