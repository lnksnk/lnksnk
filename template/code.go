package template

import "github.com/lnksnk/lnksnk/iorw"

type codeparsing struct {
	*parsing
	m      *markuptemplate
	fndcde bool
	hsecde bool
	psvbf  *iorw.Buffer
	cdebf  *iorw.Buffer
	c      *contentparsing
}

func nextCodeParsing(m *markuptemplate, prsg *parsing) (cde *codeparsing) {
	cde = &codeparsing{m: m, parsing: prsg, c: m.cntntprsngs[m.prsix]}
	prsg.EventMatchedPre = nil
	prsg.EventPreRunes = cde.passiveRunes
	prsg.EventPostRunes = cde.codeRunes
	return
}

func (cde *codeparsing) passiveRunes(rns ...rune) {
	if cde == nil {
		return
	}
	if cde.hsecde {
		cde.hsecde = false
	}
	if psvbf := cde.psvbf; psvbf != nil {
		psvbf.WriteRunes(rns...)
		return
	}
	cde.psvbf = iorw.NewBuffer(rns)
}

func (cde *codeparsing) codeRunes(canreset bool, rns ...rune) (reset bool) {
	if cde == nil {
		return
	}
	if !cde.hsecde {
		cde.hsecde = true
		cde.flushPsv()
		if !cde.fndcde {
			cde.fndcde = true
		}
	}
	if cdebf := cde.cdebf; cdebf != nil {
		cdebf.WriteRunes(rns...)
		return
	}
	cde.cdebf = iorw.NewBuffer(rns)
	return
}

func (cde *codeparsing) flushPsv() {
	if cde == nil {
		return
	}
	if psvbf := cde.psvbf; !psvbf.Empty() {
		defer psvbf.Clear()
		if !cde.fndcde {
			if cbf := cde.c.cbf; cbf != nil {
				psvbf.WriteTo(cbf)
				return
			}
			cde.c.cbf = psvbf.Clone(true)
			return
		}
		cdebf := cde.cdebf
		if lstr, isspace := rune(cdebf.LastByte(true)), iorw.IsSpace(rune(cdebf.LastByte())); validLastCdeRune(rune(cdebf.LastByte(true))) && (lstr != '/' || (lstr == '/' && !isspace)) {
			if psvbf.HasPrefix("`") && psvbf.HasSuffix("`") {
				cdebf.Print(psvbf.Reader())
			} else {
				cdebf.Print("`", psvbf.Reader(), "`")
			}
			return
		}
		if cdebf != nil {
			if psvbf.HasPrefix("`") && psvbf.HasSuffix("`") {
				cdebf.Print("print(", psvbf.Reader(), ");")
			} else {
				cdebf.Print("print(`", psvbf.Reader(), "`);")
			}
		}
		return
	}
}

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/' || cr == ':'
}
