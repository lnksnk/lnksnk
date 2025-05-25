package template

import "github.com/lnksnk/lnksnk/ioext"

type textparsing struct {
	alttxtr rune
	txtr    rune
	prvr    rune
	lstr    rune
}

func (txtprs *textparsing) Parse(r rune) bool {
	if txtprs != nil {
		if txtprs.txtr == 0 {
			if ioext.IsTxtPar(r) {
				txtprs.lstr = r
				txtprs.txtr = r
			}
			return txtprs.isText()
		}
		if txtprs.txtr == r {
			txtprs.lstr = r
			if txtprs.prvr != '\\' {
				if txtprs.alttxtr > 0 {
					txtprs.lstr = txtprs.alttxtr
					txtprs.alttxtr = 0
				}
				txtprs.txtr = 0
				txtprs.prvr = 0
				return true
			}
			txtprs.lstr = r
			txtprs.prvr = r
		}
	}
	return txtprs.isText()
}

func (txtprs *textparsing) isText() bool {
	if txtprs == nil {
		return false
	}
	return txtprs.txtr > 0
}
