package template

import "github.com/lnksnk/lnksnk/ioext"

type textparsing struct {
	txtr rune
	prvr rune
}

func (txtprs *textparsing) Parse(r rune) bool {
	if txtprs != nil {
		if txtprs.txtr == 0 {
			if ioext.IsTxtPar(r) {
				txtprs.txtr = r
			}
			return txtprs.isText()
		}
		if txtprs.txtr == r {
			if txtprs.prvr != '\\' {
				txtprs.txtr = 0
				txtprs.prvr = 0
				return true
			}
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
