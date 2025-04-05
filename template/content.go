package template

import (
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
	"github.com/lnksnk/lnksnk/iorw"
)

type contentparsing struct {
	*parsing
	m         *markuptemplate
	elmlvl    ElemLevel
	elmname   string
	elmroot   string
	elmbase   string
	tstname   []rune
	tstr      rune
	tstlvl    ElemLevel
	tstatrbs  *attributeparser
	lstattrbs map[string]interface{}
	attrbs    map[string]interface{}
	cbf       *iorw.Buffer
	prscde    *parsing
	cde       *codeparsing
	fsys      fs.MultiFileSystem
	fi        fs.FileInfo
	prvc      *contentparsing
}

func nextContentParsing(prvc *contentparsing, m *markuptemplate, fsys fs.MultiFileSystem, fi fs.FileInfo) (c *contentparsing) {
	c = &contentparsing{m: m, prvc: prvc, fsys: fsys, fi: fi, parsing: nextparsing("<", ">", &textparsing{}, nil), prscde: nextparsing("<@", "@>", nil, nil)}
	path := fi.Path()
	c.elmroot = strings.Replace(fi.Root(), "/", ":", -1)
	c.elmname = strings.Replace(path, "/", ":", -1)
	c.elmbase = strings.Replace(fi.Base(), "/", ":", -1)
	c.elmlvl = ElemSingle
	ext := fi.Ext()
	if ext != "" {
		if path == fi.Root()+"index"+ext {
			c.elmname = c.elmroot
		} else {
			c.elmname = strings.Replace(path[:len(path)-len(ext)], "/", ":", -1)
		}
	}
	c.parsing.EventPreRunes = func(r ...rune) {
		c.prscde.parse(r...)
	}
	c.parsing.EventMatchedPre = c.matchPre
	c.parsing.EventPostRunes = c.postRunes
	c.parsing.EventMatchedPost = c.matchPost
	c.resetCdeParsing()

	return
}

func (c *contentparsing) noncode() bool {
	if c == nil {
		return false
	}
	if c.elmlvl == ElemStart {
		return true
	}

	if prvc := c.prvc; prvc != nil {
		return prvc.noncode()
	}
	return false
}

func (c *contentparsing) Close() (err error) {
	if c == nil {
		return
	}
	m := c.m
	cbf := c.cbf
	cde := c.cde
	if cde != nil {
		cde.flushPsv()
	}
	c.prscde = nil
	c.cbf = nil
	c.m = nil
	c.fi = nil
	c.fsys = nil
	if m != nil {
		var prvc *contentparsing = c
		if m.prsix > 0 {
			if m.cntntprsngs[m.prsix] == c {
				delete(m.cntntprsngs, m.prsix)
				m.prsix--
				prvc = m.cntntprsngs[m.prsix]
			}
		}
		if cbf != nil && !cbf.Empty() {
			if prvc == c {
				m.cntntbf = cbf
			} else {
				m.Parse(cbf.Reader(true))
			}
		}
		if cde != nil {
			if cdebf := cde.cdebf; !cdebf.Empty() {
				if prvc == c {
					m.cdebf = cdebf
				} else if prvc.cde != nil {
					if !prvc.noncode() {
						prvc.cde.flushPsv()
					}
					if !prvc.cde.Busy() {
						prvc.cde.Parse(prvc.prscde.prelbl...)
					}
					m.Parse(cdebf.Reader(true))
					prvc.cde.Parse(prvc.prscde.postlbl...)
				}
			}
		}
	}

	return
}

func (c *contentparsing) preRunes(rns ...rune) {
	c.prscde.Parse(rns...)
}

func (c *contentparsing) matchPre() {

}

func (c *contentparsing) resetTest(parse bool, rns ...rune) {
	tstlvl := c.tstlvl
	tstatrbs := c.tstatrbs
	defer tstatrbs.Close()
	tstname := c.tstname
	c.tstlvl = ElemUnkown
	c.tstatrbs = nil
	c.tstname = nil
	c.tstr = 0
	var prsrns []rune
	defer func() {
		if len(prsrns) > 0 {
			c.preRunes(prsrns...)
		}
	}()
	if parse {
		if tstlvl == ElemSingle {
			prsrns = append(prsrns, []rune(c.prelbl)...)

			prsrns = append(prsrns, tstname...)
			if tstatrbs != nil {
				prsrns = append(prsrns, tstatrbs.raw...)
			}
			prsrns = append(prsrns, '/')
			prsrns = append(prsrns, rns...)
			return
		}
		prsrns = append(prsrns, []rune(c.prelbl)...)
		if tstlvl == ElemEnd {
			prsrns = append(prsrns, '/')
		}
		prsrns = append(prsrns, tstname...)
		if tstatrbs != nil {
			prsrns = append(prsrns, tstatrbs.raw...)
		}
		prsrns = append(prsrns, rns...)
		return
	}
}

func (c *contentparsing) parse(r rune) {
	if c.prscde.Busy() {
		c.prscde.parse(r)
		return
	}
	c.parsing.parse(r)
}

func (c *contentparsing) postRunes(canreset bool, rns ...rune) (reset bool) {

	if reset = canreset; reset {
		c.resetTest(true, rns...)
		return
	}
	var fndspace = false
	for rn, r := range rns {
		if c.tstlvl == ElemUnkown {
			if r == '/' {
				c.tstlvl = ElemEnd
				continue
			}
			if validElemChar(c.tstr, r) {
				c.tstname = append(c.tstname, r)
				c.tstlvl = ElemStart
				c.tstr = r
				continue
			}
			c.resetTest(true, r)
			reset = true
			c.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemStart {
			if tstatrbs := c.tstatrbs; tstatrbs != nil {
				if (tstatrbs.tstlvl == AttribUknown || tstatrbs.tstlvl == AttribContinue) && r == '/' {
					c.tstlvl = ElemSingle
					continue
				}
				tstatrbs.Parse(r)
				if tstatrbs.invdl {
					c.resetTest(true, r)
					reset = true
					c.bufdrns = rns[rn+1:]
					return
				}
				continue
			}
			if r == '/' {
				c.tstlvl = ElemSingle
				continue
			}
			if validElemChar(c.tstr, r, func(r rune) {
				fndspace = true
			}) {
				c.tstname = append(c.tstname, r)
				c.tstr = r
				continue
			}
			if fndspace {
				if len(c.tstname) > 0 {
					tstatrbs := nextattrbprsr("[$", "$]", c.readRune)
					tstatrbs.eventDispose = c.clearAttibutes
					tstatrbs.raw = append(tstatrbs.raw, r)
					c.tstatrbs = tstatrbs
					tstatrbs.eventFoundValue = c.setArributeValue
					continue
				}
			}
			c.resetTest(true, r)

			reset = true
			c.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemSingle {
			c.resetTest(true, '/', r)
			reset = true
			c.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemEnd {
			if validElemChar(c.tstr, r) {
				c.tstname = append(c.tstname, r)
				c.tstr = r
				continue
			}
			c.resetTest(true, r)

			reset = true
			c.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemSingle {
			c.resetTest(true, r)

			reset = true
			c.bufdrns = rns[rn+1:]
			return
		}
	}

	reset = canreset
	return
}

func (c *contentparsing) clearAttibutes() {

}

func (c *contentparsing) setArributeValue(name []rune, value interface{}, empty bool) {
	if c == nil {
		return
	}
	lstattrbs := c.lstattrbs
	if lstattrbs == nil {
		c.lstattrbs = map[string]interface{}{string(name): value}
		return
	}
	lstattrbs[string(name)] = value
}

func validElmchar(cr rune) bool {
	return ('a' <= cr && cr <= 'z') || ('A' <= cr && cr <= 'Z') || cr == ':' || cr == '.' || cr == '-' || cr == '_' || ('0' <= cr && cr <= '9')
}

func validElemChar(prvr, r rune, funcspace ...func(rune)) (valid bool) {
	if iorw.IsSpace(r) {
		if len(funcspace) > 0 && funcspace[0] != nil {
			funcspace[0](r)
		}
		return
	}
	if prvr > 0 {
		valid = validElmchar(prvr) && validElmchar(r)
		return
	}
	valid = validElmchar(r)
	return
}

func (c *contentparsing) matchPost() (reset bool) {
	tstname, tstlvl, tstatrbs := c.tstname, c.tstlvl, c.tstatrbs
	if len(tstname) == 0 {
		c.resetTest(true, c.parsing.postlbl...)
		return
	}
	if attrbs := c.attrbs; attrbs != nil {
		tstname = []rune(ioext.MapReplaceReader(tstname, attrbs, "::", "::").String())
	}
	c.tstname = nil
	c.tstlvl = ElemUnkown
	c.tstatrbs = nil
	lstattrbs := c.lstattrbs
	defer func() {
		if lstattrbs != nil {
			clear(lstattrbs)
			lstattrbs = nil
		}
	}()
	c.lstattrbs = nil
	fullname := func() string {
		if tstname[0] == ':' {
			return string(tstname)
		}
		if tstlvl == ElemEnd && c.elmlvl == ElemStart {
			if prvc := c.prvc; prvc != nil {
				if prvc.elmroot+string(tstname) == c.elmname {
					return c.elmname
				}
			}
			return strings.Replace(c.fi.Root(), "/", ":", -1) + string(tstname)
		}
		return strings.Replace(c.fi.Root(), "/", ":", -1) + string(tstname)
	}()
	if tstlvl == ElemStart || tstlvl == ElemSingle {
		var fi fs.FileInfo
		if fi = func() (fifnd fs.FileInfo) {
			fullpath := strings.Replace(fullname, ":", "/", -1)
			ext := filepath.Ext(fullpath)
			if ext == "" {
				ext = c.fi.Ext()
			}
			if fullpath[0] == '/' {
				if fullpath[len(fullpath)-1] == '/' {
					if fifnd = c.fsys.Stat(fullpath + "index" + ext); fifnd != nil {
						return
					}
					if fifnd = c.fsys.Stat(c.fi.Base() + fullpath[1:] + "index" + ext); fifnd != nil {
						return
					}
				} else {
					if fifnd = c.fsys.Stat(fullpath + ext); fifnd != nil {
						return
					}
					if fifnd = c.fsys.Stat(c.fi.Base() + fullpath[1:] + ext); fifnd != nil {
						return
					}
				}
			}
			return
		}(); fi == nil {
			if invalidElem := c.m.invalidElem; invalidElem != nil {
				invalidElem[fullname] = true
			} else {
				c.m.invalidElem = map[string]bool{fullname: true}
			}
			c.tstname = tstname
			c.tstlvl = tstlvl
			c.tstatrbs = tstatrbs
			c.resetTest(true, c.postlbl...)
			return
		}
		fullname = strings.Replace(fullname, "..:", "", -1)
		if validElem := c.m.validElem; validElem != nil {
			validElem[fullname] = fi
		} else {
			c.m.validElem = map[string]fs.FileInfo{fullname: fi}
		}
		c.Reset()
		c.resetTest(false)
		if !c.noncode() {
			if cde := c.cde; cde != nil {
				cde.flushPsv()
			}
		}
		nxtc := nextContentParsing(c, c.m, c.fsys, fi)
		nxtc.attrbs = lstattrbs
		lstattrbs = nil
		nxtc.elmlvl = tstlvl
		c.m.prsix++
		c.m.cntntprsngs[c.m.prsix] = nxtc
		if tstlvl == ElemSingle {
			attrbs := nxtc.attrbs
			if attrbs == nil {
				attrbs = map[string]interface{}{"cntnt": "", "e-root": nxtc.elmroot, "e-base": nxtc.elmbase, "root": nxtc.fi.Root(), "base": nxtc.fi.Base()}
				nxtc.attrbs = attrbs
			} else {
				attrbs["e-root"] = nxtc.elmroot
				attrbs["e-base"] = nxtc.elmbase
				attrbs["root"] = nxtc.fi.Root()
				attrbs["base"] = nxtc.fi.Base()
				attrbs["cntnt"] = ""
			}
			c.m.Parse(ioext.MapReplaceReader(fi.Reader(), attrbs, "[#", "#]"))
			nxtc.Close()
		}
		return
	}
	if tstlvl == ElemEnd {
		c.Reset()
		c.resetTest(false)
		fullname = strings.Replace(fullname, "..:", "", -1)
		if c.m.prsix > 0 && c == c.m.cntntprsngs[c.m.prsix] && fullname == c.elmname && c.elmlvl == ElemStart {
			c.elmlvl = ElemEnd
			attrbs := c.attrbs
			cbf := c.cbf
			c.cbf = nil
			if attrbs == nil {
				attrbs = map[string]interface{}{"cntnt": func() interface{} {
					if cbf.Empty() {
						return ""
					}
					return cbf
				}(), "e-root": c.elmroot, "e-base": c.elmbase, "root": c.fi.Root(), "base": c.fi.Base()}
				c.attrbs = attrbs
			} else {
				attrbs["e-root"] = c.elmroot
				attrbs["e-base"] = c.elmbase
				attrbs["root"] = c.fi.Root()
				attrbs["base"] = c.fi.Base()
				attrbs["cntnt"] = func() interface{} {
					if cbf.Empty() {
						return ""
					}
					return cbf
				}()
			}
			if lstattrbs != nil {

				lstattrbs = nil
			}
			c.resetCdeParsing()
			c.m.Parse(ioext.MapReplaceReader(c.fi.Reader(), attrbs, "[#", "#]"))
			c.Close()
			return
		}
		c.tstname = tstname
		c.tstlvl = tstlvl
		c.tstatrbs = tstatrbs
		c.resetTest(true, c.postlbl...)
		return
	}
	defer tstatrbs.Close()
	return
}

func (c *contentparsing) resetCdeParsing() {
	if c == nil {
		return
	}
	c.prscde.EventPreRunes = func(r ...rune) {
		cbf := c.cbf
		if cbf == nil {
			c.cbf = iorw.NewBuffer(r)
			return
		}
		cbf.WriteRunes(r...)
	}
	c.prscde.EventMatchedPre = func() {
		if c.noncode() {
			c.prscde.EventPreRunes(c.prscde.prelbl...)
			c.prscde.EventMatchedPre = func() {
				c.prscde.EventPreRunes(c.prscde.prelbl...)
			}
			c.prscde.EventPostRunes = func(canreset bool, rns ...rune) (reset bool) {
				c.prscde.EventPreRunes(rns...)
				return
			}
			c.prscde.EventMatchedPost = func() (reset bool) {
				c.prscde.EventPreRunes(c.prscde.postlbl...)
				return
			}

		} else {
			c.cde = nextCodeParsing(c.m, c.prscde)
			c.prscde.EventMatchedPre = nil
			c.prscde.EventMatchedPost = nil
		}
	}
}

type ElemLevel int

func (elmlvl ElemLevel) String() string {
	if elmlvl == ElemStart {
		return "start"
	}
	if elmlvl == ElemEnd {
		return "end"
	}
	if elmlvl == ElemSingle {
		return "single"
	}
	return "unknown"
}

const (
	ElemUnkown ElemLevel = iota
	ElemStart
	ElemSingle
	ElemEnd
)
