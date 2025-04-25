package template

import (
	"github.com/lnksnk/lnksnk/ioext"
)

type codeparsing struct {
	parsing *Parsing
	m       *markuptemplate
	fndcde  bool
	hsecde  bool
	psvbf   *ioext.Buffer
	cdebf   *ioext.Buffer
	c       *contentparsing
}

func (cde *codeparsing) parse(r rune) {
	if cde != nil {
		if prsng := cde.parsing; prsng != nil {
			prsng.parse(r)
		}
	}
}

func (cde *codeparsing) noncode() bool {
	return cde != nil && cde.c.noncode()
}

func (cde *codeparsing) Parse(r ...rune) {
	cde.parsing.Parse(r...)
}

func (cde *codeparsing) Busy() bool {
	return cde != nil && cde.parsing.Busy()
}

func nextCodeParsing(c *contentparsing, m *markuptemplate, cdeprelbl, cdepostlbl string) (cde *codeparsing) {
	cde = &codeparsing{m: m, c: c, parsing: nextparsing(cdeprelbl, cdepostlbl, &textparsing{}, nil)}
	if prsng := cde.parsing; prsng != nil {
		prsng.EventMatchedPre = cde.startCaptureCode
		prsng.EventMatchedPost = cde.doneCaptureCode
		prsng.EventPreRunes = cde.passiveRunes
		prsng.EventPostRunes = cde.codeRunes
	}
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
	cde.psvbf = ioext.NewBuffer(rns)
}

func (cde *codeparsing) startCaptureCode() {
	if cde.noncode() {
		cde.flushPsv()
		cde.c.content().WriteRunes(cde.parsing.prelbl...)
	}
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
	cde.cdebf = ioext.NewBuffer(rns)
	return
}

func (cde *codeparsing) doneCaptureCode() bool {
	if cde == nil {
		return false
	}
	if cde.noncode() {
		if cdebf := cde.cdebf; !cdebf.Empty() {
			func() {
				defer cdebf.Clear()
				cdebf.WriteTo(cde.c.content())
			}()
		}
		cde.c.content().WriteRunes(cde.parsing.postlbl...)
	}
	return true
}

func (cde *codeparsing) reset() {
	if cde == nil {
		return
	}
	if psvbf := cde.psvbf; psvbf != nil {
		psvbf.Clear()
	}
	if cdebf := cde.cdebf; cdebf != nil {
		cdebf.Clear()
	}
	if prnsg := cde.parsing; prnsg != nil {
		prnsg.Reset()
	}
	cde.fndcde = false
	cde.hsecde = false
}

func (cde *codeparsing) flushPsv() {
	if cde == nil {
		return
	}
	if psvbf := cde.psvbf; !psvbf.Empty() {
		defer psvbf.Clear()
		if cde.noncode() {
			psvbf.WriteTo(cde.c.content())
			return
		}

		contnsinle := psvbf.Contains("{@") && psvbf.Contains("@}")

		if !cde.fndcde && contnsinle {
			cde.fndcde = true
		}
		if !cde.fndcde {
			psvbf.WriteTo(cde.c.content())
			return
		}
		cdebf := cde.cdebf
		if cdebf == nil {
			cdebf = ioext.NewBuffer()
			cde.cdebf = cdebf
		}
		if lstr, isspace := rune(cdebf.LastByte(true)), ioext.IsSpace(rune(cdebf.LastByte())); validLastCdeRune(rune(cdebf.LastByte(true))) && (lstr != '/' || (lstr == '/' && !isspace)) {

			cdebf.Print("`")
			if contnsinle {
				if contnsinle {
					New(psvbf.Clone(true).Reader(true), "{@", "@}", false, nil, func(rns ...rune) {
						cdebf.WriteRunes(rns...)
					}, func() {
						cdebf.WriteRunes([]rune("${")...)
					}, nil, func(canreset bool, rns ...rune) (reset bool) {
						if reset = canreset; reset {
							cdebf.WriteRunes(rns...)
						}
						cdebf.WriteRunes(rns...)
						return
					}, func() bool {
						cdebf.WriteRunes([]rune("}")...)
						return false
					}).Process()
				}
			} else {
				cdebf.Print(psvbf)
			}
			cdebf.Print("`")

			return
		}
		cdebf.Print("print(`")
		if contnsinle {
			if contnsinle {
				New(psvbf.Clone(true).Reader(true), "{@", "@}", false, nil, func(rns ...rune) {
					cdebf.WriteRunes(rns...)
				}, func() {
					cdebf.WriteRunes([]rune("${")...)
				}, nil, func(canreset bool, rns ...rune) (reset bool) {
					if reset = canreset; reset {
						cdebf.WriteRunes(rns...)
					}
					cdebf.WriteRunes(rns...)
					return
				}, func() bool {
					cdebf.WriteRunes([]rune("}")...)
					return false
				}).Process()
			}
		} else {
			cdebf.Print(psvbf)
		}
		cdebf.Print("`);")
	}
}

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/' || cr == ':'
}
