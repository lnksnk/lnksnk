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
		attrs := ctntelm.attrs

		var prpbf *iorw.Buffer
		prpbuffer := func() *iorw.Buffer {
			if prpbf != nil {
				return prpbf
			}
			prpbf = iorw.NewBuffer()
			return prpbf
		}
		preprdr := iorw.ReadRunesUntil(rdr, func(prasefnd string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, prpflushrdr iorw.SliceRuneReader) (prperr error) {
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

		path := ctntelm.fi.Path()
		pathroot := path
		pthexti, pathpthi := strings.LastIndex(pathroot, "."), strings.LastIndex(pathroot, "/")
		if pathpthi > -1 {
			if pthexti > pathpthi {
				pathroot = pathroot[:pathpthi+1]
			}
			pathroot = pathroot[:pathpthi+1]
		} else {
			pathroot = "/"
		}
		path = path[len(pathroot):]
		root := pathroot
		if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
			root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
		}
		if strings.HasSuffix(ctntelm.elemname, ":") {
			path = ""
		}
		coresttngs := map[string]interface{}{}
		coresttngs["pathroot"] = pathroot
		coresttngs["root"] = root
		coresttngs["elemroot"] = func() (elmroot string) {
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
		coresttngs["elembase"] = func() (elembase string) {
			elmbases := strings.Split(coresttngs["elemroot"].(string), ":")
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
		corefnd := false
		preprdr = iorw.ReadRunesUntil(preprdr, func(prpphrase string, prpuntilrdr io.RuneReader, prporgrdr iorw.SliceRuneReader, prporgerr error, prpflush iorw.SliceRuneReader) (prperr error) {
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
			if err = cntntbuf.Print(iorw.ReadRunesUntil(cntntbuf.Clone(true).Reader(true), "[:", func(cntphrase string, cntuntilrdr io.RuneReader, cntorgrdr iorw.SliceRuneReader, cntorgerr error, cntflushrdr iorw.SliceRuneReader) (cnterr error) {
				if cntphrase == "[:" {
					cntbf, cntbferr := iorw.NewBufferError(iorw.ReadRunesUntil(cntuntilrdr), "::", func(tplphrase string, tpluntilrdr io.RuneReader, tplorgrdr iorw.SliceRuneReader, tplorgerr error, tplflushrdr iorw.SliceRuneReader) (tplerr error) {

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
			}
		}
		ctntstngs["cntnt"] = cntntbuf
		fndctnt := false

		preprdr = iorw.ReadRunesUntil(preprdr, "<:", ":/>", func(prpphrase string, prpuntilrdr io.RuneReader, prporgrdr iorw.SliceRuneReader, prporgerr error, prpflushrdr iorw.SliceRuneReader) (prperr error) {
			if prpphrase == "<:" {
				fndctnt = true
				defer func() {
					fndctnt = false
				}()
				tmpbf := iorw.NewBuffer()
				if prperr = tmpbf.Print(prpuntilrdr); prperr != nil {
					if prperr.Error() == ":/>" {
						prperr = nil
						for ctntk, ctntv := range ctntstngs {
							if eql, _ := tmpbf.Equals(ctntk); eql {
								prpflushrdr.PreAppend(valToRuneReader(ctntv, false))
								return
							}
						}
						return
					}
					if prperr == io.EOF {
						prperr = nil
						return
					}
				}
				return
			}
			if prpphrase == ":/>" {
				if fndctnt {
					return fmt.Errorf("%s", prpphrase)
				}
				prpflushrdr.PreAppend(strings.NewReader(prpphrase))
			}
			return
		})
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

	root := pathroot
	if root[0:1] == "/" && root[len(root)-1:] == "/" && root != "/" {
		root = root[:strings.LastIndex(root[:len(root)-1], "/")+1]
	}
	tmpmatchthis := map[string]interface{}{}
	tmpmatchthis["pathroot"] = pathroot
	tmpmatchthis["root"] = root

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
	tmpmatchthis["elemroot"] = elempath
	tmpmatchthis["elembase"] = func() (elembase string) {
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

	var rnsrdrslcrdr = iorw.NewSliceRuneReader(rnrdrs...)
	var phrsbf *iorw.Buffer = nil
	var chninit = false
	var ctntinitrplcrdr = iorw.ReadRunesUntil(rnsrdrslcrdr, "<:_:", ":/>", iorw.RunesUntilSliceFlushFunc(func(phrasefnd string, untilrdr io.RuneReader, orgrd iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) error {
		if phrasefnd == "<:_:" {
			if !chninit {
				chninit = true
			}
			defer func() {
				chninit = false
			}()
			if phrsbf == nil {
				phrsbf = iorw.NewBuffer()
			}
			phrsbf.Clear()
			if _, phrserr := phrsbf.ReadRunesFrom(untilrdr); phrserr != nil {
				if phrserr.Error() == ":/>" {
					for thisk, thisv := range tmpmatchthis {
						if qkl, _ := phrsbf.Equals(thisk); qkl {
							flushrdr.PreAppend(valToRuneReader(thisv, true))
							return nil
						}
					}
					return nil
				}
				if phrserr == orgerr {

					return orgerr
				}
			}
			return nil
		}
		if phrasefnd == ":/>" {
			if chninit {
				return fmt.Errorf("%s", phrasefnd)
			}
			flushrdr.PreAppendArgs(phrasefnd)
		}
		return nil
	}), func() {

	})

	var crntnextelm *contentelem = nil
	var elemlevels = []*contentelem{}

	var addelemlevel = func(fi fsutils.FileInfo, elemname string, elemext string) (elmnext *contentelem) {
		elmnext = &contentelem{
			modified: fi.ModTime(),
			fi:       fi,
			elemname: elemname,
			elemroot: elemname[:strings.LastIndex(elemname, ":")+1],
			elemext:  elemext,
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
				if al := len(elemlevels); al > 0 && elemlevels[0] == crntnextelm && strings.HasSuffix(crntnextelm.elemname, elemname) {
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

	ctntprsrdr := iorw.ReadRunesUntil(iorw.NewSliceRuneReader(ctntinitrplcrdr), "<", ">", func(phrase string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) (fnderr error) {
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
					eofargs = append(eofargs, func(rplcphrase string, rplcuntilrdr, rplcorgrdr io.RuneReader, rplcorgerr error, rplcflushrdr iorw.SliceRuneReader) (rplcfnderr error) {
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
					if elmlvl == ctntElemUnknown {
						flushrdr.PreAppend(iorw.NewRunesReader(chkbfrns...))
						return nil
					}
					fnderr = nil
					goto fndeof
				}
				if fnderr.Error() == "prepeof" {
					fnderr = nil
					goto prepeof
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
			}), "/", ">", "=", `='`, `="`, `=[$`, `[$`, func(argsphrase string, argsuntilrdr, argsorgrdr io.RuneReader, argsorgerr error, argsflushrdr iorw.SliceRuneReader) (argserr error) {
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
					if _, argserr = argbuffer().ReadRunesFrom(iorw.ReadRunesUntil(argsorgrdr, argtxtpar, iorw.RunesUntilSliceFlushFunc(func(txtphrasefnd string, txtuntilrdr io.RuneReader, txtorgrd iorw.SliceRuneReader, txtorgerr error, txtflushrdr iorw.SliceRuneReader) error {
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
					if _, argserr = argbuffer().ReadRunesFrom(iorw.ReadRunesUntil(argsorgrdr, "$]", iorw.RunesUntilSliceFlushFunc(func(txtphrasefnd string, txtuntilrdr io.RuneReader, txtorgrd iorw.SliceRuneReader, txtorgerr error, txtflushrdr iorw.SliceRuneReader) error {
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
					if _, argserr = argbuffer().ReadRunesFrom(iorw.ReadRunesUntil(argsorgrdr, "$]", iorw.RunesUntilSliceFlushFunc(func(txtphrasefnd string, txtuntilrdr io.RuneReader, txtorgrd iorw.SliceRuneReader, txtorgerr error, txtflushrdr iorw.SliceRuneReader) error {
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
	}, func() {})

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
	fncode := false
	cderdmode := func() codeReadMode {
		if invertActive {
			return codeReadingCode
		}
		return codeReadingContent
	}()

	cdetxtr := rune(0)

	coderunsrdr := iorw.ReadRunesUntil(iorw.ReadRuneFunc(func() (r rune, size int, err error) {
	reread:
		r, size, err = ctntprsrdr.ReadRune()
		if size > 0 && (err == nil || err == io.EOF) {
			if crntnextelm != nil {
				crntnextelm.writeRune(r)
				goto reread
			}
		}
		return
	}), "<@", "@>", func(phrase string, untilrdr io.RuneReader, orgrdr iorw.SliceRuneReader, orgerr error, flushrdr iorw.SliceRuneReader) error {
		if phrase == "<@" {
			if cderdmode == codeReadingContent {
				cderdmode = codeReadingCode
			}
			return nil
		}
		if phrase == "@>" {
			if cderdmode == codeReadingCode {
				cderdmode = codeReadingContent
			}
		}
		return nil
	}, func() {

	}, func(prvr, r rune) bool {
		if cderdmode == codeReadingCode {
			if cdetxtr == 0 {
				if prvr != '\\' && iorw.IsTxtPar(r) {
					cdetxtr = r
					cdelstr = 0
					return false
				}
				if !iorw.IsSpace(r) {
					cdelstr = func() rune {
						if validLastCdeRune(r) {
							return r
						}
						return 0
					}()
				}
				return true
			}
			if prvr != '\\' && cdetxtr == r {
				cdetxtr = 0
				return false
			}
			return false
		}
		return true
	})

	writecderns := func(rns ...rune) {
		for _, r := range rns {
			if cderdmode == codeReadingCode {
				codebuffer().WriteRune(r)
				return
			}
			if cderdmode == codeReadingContent {
				if fncode {
					ctntbuffer().WriteRune(r)
					return
				}
				chdctntbuffer().WriteRune(r)
				return
			}
		}
	}

	ctntflush := func() (flsherr error) {
		if cdepsvs := ctntbuf.Size(); cdepsvs > 0 {
			defer ctntbuf.Clear()
			hstmpltfx := ctntbuf.HasPrefix("`") && ctntbuf.HasSuffix("`") && cdepsvs >= 2
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
			codebuffer().Print("print(`", psvrdr, "`);")
		}
		return
	}

	if prsngerr = iorw.EOFReadRunes(coderunsrdr, func(cr rune, csize int) (cerr error) {
		if csize > 0 {
			if cderdmode == codeReadingCode {
				ctntflush()
				if !fncode {
					fncode = true
				}
			}
			writecderns(cr)
		}
		return
	}); prsngerr != nil {
		if prsngerr != io.EOF {
			return
		}
		prsngerr = nil
	}

	ctntflush()
	var chdpgrm interface{} = nil
	if !chdctntbuf.Empty() && cdebuf.Empty() {
		DefaultMinifyPsv(pathext, chdctntbuf, nil)
		if capturecache != nil {
			prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
		}
	}
	if !chdctntbuf.Empty() {
		//fmt.Println(chdctntbuf)
		if out != nil {
			if _, prsngerr = chdctntbuf.WriteTo(out); prsngerr != nil {
				return
			}
		}
	}

	if !cdebuf.Empty() {
		//fmt.Println(cdebuf)
		if DefaultMinifyCde != nil {
			prsngerr = DefaultMinifyCde(".js", cdebuf, nil)
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
						prsngerr = capturecache(fullpath, pathModified, validelempaths, chdctntbuf, cdebuf, chdpgrm)
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
			println(prsngerr.Error())
			println()
			if cderr, _ := prsngerr.(CodeError); cderr != nil {
				println(cderr.Code())

			} else {
				println(cdebuf.String())
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
