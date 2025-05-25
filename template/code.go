package template

import (
	"io"

	"github.com/lnksnk/lnksnk/ioext"
)

const (
	CdePre  string = "<%"
	CdePost string = "%>"
	PsvPre  string = "{@"
	PsvPost string = "@}"
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

func nextCodeParsing(c *contentparsing, m *markuptemplate, parsepretxt, parseposttxt bool, cdeprelbl, cdepostlbl string) (cde *codeparsing) {
	cde = &codeparsing{m: m, c: c, parsing: nextparsing(cdeprelbl, cdepostlbl, func() *textparsing {
		if parsepretxt {
			return &textparsing{}
		}
		return nil
	}(), func() *textparsing {
		if parseposttxt {
			return &textparsing{}
		}
		return nil
	}(), nil)}
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

func processInline(pre string, post string, inrdr io.RuneReader, cdebf *ioext.Buffer) {
	if pre != "" {
		cdebf.Print(pre)
	}
	New(inrdr, PsvPre, PsvPost, false, false, nil, func(rns ...rune) {
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
	if post != "" {
		cdebf.Print(post)
	}
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

		contnsinle := psvbf.Contains(PsvPre) && psvbf.Contains(PsvPost)

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
		texttocode(cde, contnsinle, psvbf.Clone(true).Reader(true))
	}
}

func texttocode(cde *codeparsing, isinline bool, txtrdr io.RuneReader) {
	if cde == nil {
		return
	}

	cdebf := cde.cdebf
	if cdebf == nil {
		cdebf = ioext.NewBuffer()
		cde.cdebf = cdebf
	}
	if !cde.fndcde {
		cde.fndcde = true
	}
	lstr, isspace := rune(cdebf.LastByte(true)), ioext.IsSpace(rune(cdebf.LastByte()))
	istxt := ioext.IsTxtPar(lstr)

	if istxt || (validLastCdeRune(lstr) && (lstr != '/' || (lstr == '/' && !isspace))) {
		txtrdr = ioext.MapReplaceReader(txtrdr, map[string]interface{}{"`": "\\`", "${": "\\$\\{"})
		if istxt {
			if lstr != '`' {
				if isinline {
					r, s, _ := txtrdr.ReadRune()
					if s > 0 {
						cde.cdebf = cdebf.SubBuffer(0, cdebf.Size()-1)
						cdebf = cde.cdebf
						processInline("`", "", ioext.ReadRuneFunc(func() (rune, int, error) {
							if r > 0 {
								tr := r
								r = 0
								return tr, s, nil
							}
							return txtrdr.ReadRune()
						}), cdebf)
						if posttxtprs := cde.parsing.posttxtprs; posttxtprs != nil {
							posttxtprs.alttxtr = '`'
						}
						return
					}
					return
				}
				cdebf.Print(txtrdr)
				return
			}
			if isinline {
				processInline("", "", txtrdr, cdebf)
				return
			}
			cdebf.Print(txtrdr)
			return
		}
		if isinline {
			if lstr == '`' {
				processInline("", "", txtrdr, cdebf)
			} else {
				processInline("`", "`", txtrdr, cdebf)
			}
			return
		}
		if lstr == '`' {
			cdebf.Print(txtrdr)
		} else {
			cdebf.Print("`", txtrdr, "`")
		}
		return
	}
	if isinline {
		txtrdr = ioext.MapReplaceReader(txtrdr, map[string]interface{}{"`": "\\`", "${": "\\$\\{"})
		processInline("print(`", "`);", txtrdr, cdebf)
		return
	}
	txtrdr = ioext.MapReplaceReader(txtrdr, map[string]interface{}{"`": "\\`", "${": "\\$\\{"})
	cdebf.Print("print(`", txtrdr, "`);")
}

func validLastCdeRune(cr rune) bool {
	return cr == '=' || cr == '(' || cr == '[' || cr == ',' || cr == '+' || cr == '/' || cr == ':'
}
