package template

import (
	"path/filepath"
	"strings"

	"github.com/lnksnk/lnksnk/fs"
	"github.com/lnksnk/lnksnk/ioext"
)

type contentparsing struct {
	parsing   *Parsing
	m         *markuptemplate
	elmlvl    ElemLevel
	elmname   string
	elmroot   string
	elmbase   string
	root      string
	base      string
	path      string
	tstname   []rune
	tstr      rune
	tstlvl    ElemLevel
	tstatrbs  *attributeparser
	lstattrbs map[string]interface{}
	attrbs    map[string]interface{}
	cbf       *ioext.Buffer
	cde       *codeparsing
	fsys      fs.MultiFileSystem
	fi        fs.FileInfo
	prvc      *contentparsing
}

func nextContentParsing(prvc *contentparsing, m *markuptemplate, fsys fs.MultiFileSystem, fi fs.FileInfo, elemlvl ElemLevel) (c *contentparsing) {
	c = &contentparsing{m: m, prvc: prvc, fsys: fsys, fi: fi, parsing: nextparsing("<", ">", &textparsing{}, nil)}
	c.path = fi.Path()
	c.root = fi.Root()
	c.base = fi.Base()
	c.elmroot = strings.Replace(c.root, "/", ":", -1)
	c.elmname = strings.Replace(c.path, "/", ":", -1)
	c.elmbase = strings.Replace(c.base, "/", ":", -1)
	c.elmlvl = elemlvl
	if len(m.cntntprsngs) == 0 {
		c.attrbs = map[string]interface{}{
			"p-e-base": c.elmbase,
			"p-e-root": c.elmroot,
			"p-base":   c.base,
			"p-root":   c.root}
	}
	ext := fi.Ext()
	if ext != "" {
		if c.path == fi.Root()+"index"+ext {
			c.elmname = c.elmroot
		} else {
			c.elmname = strings.Replace(c.path[:len(c.path)-len(ext)], "/", ":", -1)
		}
	}
	c.parsing.EventPreRunes = c.preRunes
	c.parsing.EventMatchedPre = c.matchPre
	c.parsing.EventPostRunes = c.postRunes
	c.parsing.EventMatchedPost = c.matchPost

	c.cde = nextCodeParsing(c, m, func() string {
		if prvc != nil {
			if cde := prvc.cde; cde != nil {
				if cdeprsg := cde.parsing; cdeprsg != nil {
					return string(cdeprsg.prelbl)
				}
			}
		}
		return "<@"
	}(), func() string {
		if prvc != nil {
			if cde := prvc.cde; cde != nil {
				if cdeprsg := cde.parsing; cdeprsg != nil {
					return string(cdeprsg.postlbl)
				}
			}
		}
		return "@>"
	}())
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
	cde := c.cde
	if cde != nil {
		cde.flushPsv()
	}
	cbf := c.cbf
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
				defer cbf.Close()
				m.Parse(cbf.Reader(true))
			}
		}
		if cde != nil {
			if cdebf := cde.cdebf; !cdebf.Empty() {
				if prvc == c {
					m.cdebf = cdebf
				} else if prvcde := prvc.cde; prvcde != nil {
					defer cdebf.Close()
					if !prvc.noncode() {
						prvc.cde.flushPsv()
					}
					if !prvcde.Busy() {
						prvcde.Parse(prvcde.parsing.prelbl...)
					}
					m.Parse(cdebf.Reader(true))
					prvcde.Parse(prvcde.parsing.postlbl...)
				}
			}
		}
		attrs := c.attrbs
		c.attrbs = nil
		if attrs != nil {
			clear(attrs)
			attrs = nil
		}
	}

	return
}

func (c *contentparsing) preRunes(rns ...rune) {
	if c == nil {
		return
	}
	if cde := c.cde; cde != nil {
		for _, r := range rns {
			cde.parse(r)
		}
	}
}

func (c *contentparsing) content() *ioext.Buffer {
	if c == nil {
		return nil
	}
	cbf := c.cbf
	if cbf == nil {
		c.cbf = ioext.NewBuffer()
		return c.cbf
	}
	return cbf
}

func (c *contentparsing) writeRunes(rns ...rune) {
	if c == nil {
		return
	}
	cbf := c.cbf
	if cbf == nil {
		c.cbf = ioext.NewBuffer(rns)
		return
	}
	cbf.WriteRunes(rns...)
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
			prsrns = append(prsrns, []rune(c.parsing.prelbl)...)

			prsrns = append(prsrns, tstname...)
			if tstatrbs != nil {
				prsrns = append(prsrns, tstatrbs.raw...)
			}
			prsrns = append(prsrns, '/')
			prsrns = append(prsrns, rns...)
			return
		}
		prsrns = append(prsrns, []rune(c.parsing.prelbl)...)
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
	if cde := c.cde; cde != nil && cde.Busy() {
		cde.parse(r)
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
			c.parsing.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemStart {
			if tstatrbs := c.tstatrbs; tstatrbs != nil {
				if tstatrbs.Busy() {
					tstatrbs.Parse(r)
					continue
				}
				if (tstatrbs.tstlvl == AttribUknown || tstatrbs.tstlvl == AttribContinue) && r == '/' {
					c.tstlvl = ElemSingle
					continue
				}
				tstatrbs.Parse(r)
				if tstatrbs.invdl {
					c.resetTest(true, r)
					reset = true
					c.parsing.bufdrns = rns[rn+1:]
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
					tstatrbs := nextattrbprsr("[$", "$]", c.parsing.readRune)
					tstatrbs.eventDispose = c.clearAttibutes
					tstatrbs.raw = append(tstatrbs.raw, r)
					c.tstatrbs = tstatrbs
					tstatrbs.eventFoundValue = c.setArributeValue
					continue
				}
			}
			c.resetTest(true, r)

			reset = true
			c.parsing.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemSingle {
			c.resetTest(true, '/', r)
			reset = true
			c.parsing.bufdrns = rns[rn+1:]
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
			c.parsing.bufdrns = rns[rn+1:]
			return
		}
		if c.tstlvl == ElemSingle {
			c.resetTest(true, r)

			reset = true
			c.parsing.bufdrns = rns[rn+1:]
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
		lstattrbs = map[string]interface{}{string(name): value}
		c.lstattrbs = lstattrbs
	}
	if len(name) > 0 {
		lstattrbs[string(name)] = value
		return
	}
	if len(name) == 0 && !empty {
		tstnme := "pre"
		if c.tstlvl == ElemEnd {
			tstnme = "post"
		}
		if vlrns, _ := value.([]int32); len(vlrns) > 0 {
			crntvl, _ := lstattrbs[tstnme].([]int32)
			if len(crntvl) == 0 {
				lstattrbs[tstnme] = append([]rune{}, vlrns...)
				return
			}
			crntvl = append(crntvl, vlrns...)
			lstattrbs[tstnme] = append([]rune{}, crntvl...)
		}
		return
	}

}

func validElmchar(cr rune) bool {
	return ('a' <= cr && cr <= 'z') || ('A' <= cr && cr <= 'Z') || cr == ':' || cr == '.' || cr == '-' || cr == '_' || ('0' <= cr && cr <= '9')
}

func validNameChar(prvr, r rune) (valid bool) {
	if ioext.IsSpace(r) {
		return false
	}
	if prvr > 0 {
		valid = validElmchar(prvr) && validElmchar(r)
		return
	}
	valid = validElmchar(r)
	return
}

func validElemChar(prvr, r rune, funcspace ...func(rune)) (valid bool) {
	if ioext.IsSpace(r) {
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
		tstname = []rune(ioext.MapReplaceReader(tstname, attrbs, validNameChar, "::", "::").Runes())
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
			c.resetTest(true, c.parsing.postlbl...)
			return
		}
		fullname = strings.Replace(fullname, "..:", "", -1)
		if validElem := c.m.validElem; validElem != nil {
			validElem[fullname] = fi
		} else {
			c.m.validElem = map[string]fs.FileInfo{fullname: fi}
		}
		c.parsing.Reset()
		c.resetTest(false)
		if cde := c.cde; cde != nil {
			cde.flushPsv()
		}
		nxtc := appendCntntParsing(c, c.m, c.fsys, fi, tstlvl)
		nxtc.attrbs = lstattrbs
		lstattrbs = nil
		if tstlvl == ElemSingle {
			attrbs := nxtc.attrbs
			pgec := c.m.cntntprsngs[0]
			if attrbs == nil {
				attrbs = map[string]interface{}{
					"cntnt":    "",
					"e-root":   nxtc.elmroot,
					"e-base":   nxtc.elmbase,
					"root":     nxtc.root,
					"base":     nxtc.base,
					"p-e-root": pgec.elmroot,
					"p-e-base": pgec.elmbase,
					"p-root":   pgec.root,
					"p-base":   pgec.base}
				nxtc.attrbs = attrbs
			} else {
				attrbs["e-root"] = nxtc.elmroot
				attrbs["e-base"] = nxtc.elmbase
				attrbs["root"] = nxtc.root
				attrbs["base"] = nxtc.base
				attrbs["cntnt"] = ""
				attrbs["p-e-root"] = pgec.elmroot
				attrbs["p-e-base"] = pgec.elmbase
				attrbs["p-root"] = pgec.root
				attrbs["p-base"] = pgec.base
			}
			mps := ioext.MapReplaceReader(fi.Reader(), attrbs, validNameChar, "[#", "#]")
			c.m.Parse(mps)
			nxtc.Close()
		}
		return
	}
	if tstlvl == ElemEnd {
		c.parsing.Reset()
		c.resetTest(false)
		fullname = strings.Replace(fullname, "..:", "", -1)
		if c.m.prsix > 0 && c == c.m.cntntprsngs[c.m.prsix] && (func() bool {
			if fullname[0] == ':' && c.elmname == c.elmbase+fullname[1:] {
				return true
			}
			return fullname == c.elmname
		}() && c.elmlvl == ElemStart) {
			if cde := c.cde; cde != nil {
				cde.flushPsv()
			}
			c.elmlvl = ElemEnd
			attrbs := c.attrbs
			cbf := c.cbf
			c.cbf = nil
			pgec := c.m.cntntprsngs[0]
			if attrbs == nil {
				attrbs = map[string]interface{}{"cntnt": func() interface{} {
					if cbf.Empty() {
						return ""
					}
					return cbf
				}(),
					"e-root":   c.elmroot,
					"e-base":   c.elmbase,
					"root":     c.root,
					"base":     c.base,
					"p-e-root": pgec.elmroot,
					"p-e-base": pgec.elmbase,
					"p-root":   pgec.root,
					"p-base":   pgec.base}
				c.attrbs = attrbs
			} else {
				attrbs["e-root"] = c.elmroot
				attrbs["e-base"] = c.elmbase
				attrbs["root"] = c.root
				attrbs["base"] = c.base
				attrbs["cntnt"] = func() interface{} {
					if cbf.Empty() {
						return ""
					}
					return cbf
				}()
				attrbs["p-e-root"] = pgec.elmroot
				attrbs["p-e-base"] = pgec.elmbase
				attrbs["p-root"] = pgec.root
				attrbs["p-base"] = pgec.base
			}
			if lstattrbs != nil {

				lstattrbs = nil
			}
			c.resetCdeParsing()
			c.elmlvl = ElemSingle
			c.m.Parse(ioext.MapReplaceReader(c.fi.Reader(), attrbs, validNameChar, "[#", "#]"))
			c.Close()
			return
		}
		c.tstname = tstname
		c.tstlvl = tstlvl
		c.tstatrbs = tstatrbs
		c.resetTest(true, c.parsing.postlbl...)
		return
	}
	defer tstatrbs.Close()
	return
}

func (c *contentparsing) resetCdeParsing() {
	if c == nil {
		return
	}
	if cde := c.cde; cde != nil {
		cde.reset()
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
