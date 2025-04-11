package template

import (
	"github.com/lnksnk/lnksnk/iorw"
)

type codeparsing struct {
	parsing *Parsing
	m       *markuptemplate
	fndcde  bool
	hsecde  bool
	psvbf   *iorw.Buffer
	cdebf   *iorw.Buffer
	c       *contentparsing
}

func (cde *codeparsing) Parse(r ...rune) {
	cde.parsing.Parse(r...)
}

func (cde *codeparsing) Busy() bool {
	return cde != nil && cde.parsing.Busy()
}

func nextCodeParsing(m *markuptemplate, prsg *Parsing) (cde *codeparsing) {
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
		prepcode := func(direct bool) {
			tmpcde := iorw.NewBuffer()
			prsng := New(psvbf.Clone(true).Reader(true), "{{$", "$}}", false, nil, func(r ...rune) {
				psvbf.WriteRunes(r...)
			}, func() {
				//matchedPre
				if !psvbf.Empty() {
					defer psvbf.Clear()
					if tmpcde.Empty() {
						if direct {
							tmpcde.Print("`", psvbf.Reader(), "`+")
						} else {
							tmpcde.Print("print(`", psvbf.Reader(), "`+")
						}
					} else {
						tmpcde.Print("+`", psvbf.Reader(), "`+")
					}
					return
				}
				if !tmpcde.Empty() {
					tmpcde.Print("+")
				}
			}, nil, func(canreset bool, rns ...rune) (reset bool) {
				tmpcde.WriteRunes(rns...)
				return
			}, func() (reset bool) {
				return
			})
			prsng.Process()
			if !psvbf.Empty() {
				defer psvbf.Clear()
				if tmpcde.Empty() {
					if direct {
						tmpcde.Print("`", psvbf.Reader(), "`")
					} else {
						tmpcde.Print("print(`", psvbf.Reader(), "`);")
					}
				} else {
					if direct {
						tmpcde.Print("+`", psvbf.Reader(), "`")
					} else {
						tmpcde.Print("+`", psvbf.Reader(), "`);")
					}
				}
			}
			if !tmpcde.Empty() {
				if s := tmpcde.String(); s != "" {
					tmpcde.WriteTo(cdebf)
					tmpcde.Clear()
				}
			}
		}
		if lstr, isspace := rune(cdebf.LastByte(true)), iorw.IsSpace(rune(cdebf.LastByte())); validLastCdeRune(rune(cdebf.LastByte(true))) && (lstr != '/' || (lstr == '/' && !isspace)) {
			if psvbf.Contains("{{$") && psvbf.Contains("$}}") {
				prepcode(true)
				return
			}
			cdebf.Print("`")
			psvbf.WriteTo(cdebf)
			cdebf.Print("`")
			return
		}
		if cdebf != nil {
			if psvbf.Contains("{{$") && psvbf.Contains("$}}") {
				prepcode(false)
				return
			}
			cdebf.Print("print(`")
			psvbf.WriteTo(cdebf)
			cdebf.Print("`);")
		}
		return
	}
}

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/' || cr == ':'
}
