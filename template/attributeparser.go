package template

import "github.com/lnksnk/lnksnk/ioext"

type proplevel int

const (
	AttribUknown proplevel = iota
	AttribName
	AttribAssign
	AttribValue
	AttribInline
	AttribContinue
)

type attributeparser struct {
	*Parsing
	txttst          *textparsing
	raw             []rune
	rawl            int
	invdl           bool
	tstlvl          proplevel
	tstname         []rune
	tstvalue        []rune
	tstprvr         rune
	eventFoundValue func(name []rune, value interface{}, empty bool)
	eventDispose    func()
}

func (attrbprsr *attributeparser) passiveRunes(rns ...rune) {
	if attrbprsr == nil {
		return
	}
	fndspace := false
	for _, r := range rns {
		tstlvl := attrbprsr.tstlvl
		if tstlvl == AttribUknown || tstlvl == AttribContinue {
			fndspace = false
			if validElemChar(attrbprsr.tstprvr, r, func(r rune) {
				fndspace = true
			}) {
				attrbprsr.tstname = append(attrbprsr.tstname, r)
				attrbprsr.tstlvl = AttribName
				continue
			}
			if tstlvl == AttribContinue && fndspace {
				continue
			}
			attrbprsr.invdl = true
			return
		}
		if tstlvl == AttribName {
			fndspace = false
			if validElemChar(attrbprsr.tstprvr, r, func(r rune) {
				fndspace = true
			}) {
				attrbprsr.tstname = append(attrbprsr.tstname, r)
				continue
			}
			if r == '=' {
				attrbprsr.tstlvl = AttribAssign
				continue
			}
			if fndspace {
				continue
			}
			attrbprsr.invdl = true
			return
		}
		if tstlvl == AttribAssign {
			txttst := attrbprsr.txttst
			if !txttst.isText() {
				if ioext.IsSpace(r) {
					continue
				}
				if txttst.Parse(r) {
					continue
				}
				attrbprsr.invdl = true
				break
			}
			if txttst.isText() {
				if txttst.Parse(r) && !txttst.isText() {
					name, value := attrbprsr.tstname, attrbprsr.tstvalue
					attrbprsr.tstname = nil
					attrbprsr.tstvalue = nil
					attrbprsr.tstlvl = AttribContinue
					attrbprsr.tstprvr = 0
					attrbprsr.txtprs = nil
					attrbprsr.ParseValue(name, value, len(value) == 0)
					continue
				}
				attrbprsr.tstvalue = append(attrbprsr.tstvalue, r)
				continue
			}
			attrbprsr.invdl = true
			return
		}
	}
}

func (attrbprsr *attributeparser) ParseValue(name []rune, value interface{}, empty bool) {
	if attrbprsr == nil {
		return
	}
	if evtfndval := attrbprsr.eventFoundValue; evtfndval != nil {
		evtfndval(name, value, empty)
	}
}

func (attrbprsr *attributeparser) passiveDone() {
	if attrbprsr == nil {
		return
	}

}

func (attrbprsr *attributeparser) Close() (err error) {
	if attrbprsr == nil {
		return
	}
	evtdspse := attrbprsr.eventDispose
	attrbprsr.eventDispose = nil
	attrbprsr.txttst = nil
	parsing := attrbprsr.Parsing
	attrbprsr.raw = nil
	attrbprsr.Parsing = nil
	if parsing != nil {
		parsing.Close()
	}
	if evtdspse != nil {
		evtdspse()
	}
	return
}

func (attrbprsr *attributeparser) Parse(rns ...rune) {
	if attrbprsr == nil {
		return
	}
	if rnsl, parsing := len(rns), attrbprsr.Parsing; rnsl > 0 && parsing != nil {
		attrbprsr.raw = append(attrbprsr.raw, rns...)
		attrbprsr.rawl += rnsl
		parsing.Parse(rns...)
	}
}

func (attrbprsr *attributeparser) activeRunes(canreset bool, rns ...rune) (reset bool) {
	if attrbprsr == nil {
		return
	}
	return
}

func (attrbprsr *attributeparser) activeDone() (reset bool) {
	if attrbprsr == nil {
		return
	}
	return
}

func nextattrbprsr(prelbl, postlbl string, readRune func() (rune, int, error), eofrns ...rune) (attrbprsr *attributeparser) {
	attrbprsr = &attributeparser{Parsing: nextparsing(prelbl, postlbl, &textparsing{}, readRune)}
	attrbprsr.txttst = &textparsing{}
	parsing := attrbprsr.Parsing
	parsing.EventPostRunes = attrbprsr.activeRunes
	parsing.EventMatchedPost = attrbprsr.activeDone
	//parsing.eventCanPostParse = cntntprsr.canPostParse
	parsing.EventPreRunes = attrbprsr.passiveRunes
	parsing.EventMatchedPre = attrbprsr.passiveDone
	//parsing.eventCanPreParse = cntntprsr.canPreParse
	return
}
