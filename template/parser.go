package template

import (
	"io"
	"strings"

	"github.com/lnksnk/lnksnk/iorw"
)

type ParseAPI interface {
	Parse(...rune)
	Close() error
	CanPostParse() bool
	CanPreParse() bool
	Busy() bool
	PreLabel() []rune
	PostLabel() []rune
	Process()
}

type Parsing struct {
	readRune          func() (rune, int, error)
	bufdrns           []rune
	prelbl            []rune
	preL              int
	prei              int
	EventCanPreParse  func() bool
	EventPreRunes     func(...rune)
	EventMatchedPre   func()
	prvr              rune
	postlbl           []rune
	postL             int
	posti             int
	EventPostRunes    func(bool, ...rune) bool
	EventMatchedPost  func() bool
	EventCanPostParse func() bool
	pretxtprs         *textparsing
	posttxtprs        *textparsing
}

func (prsng *Parsing) PreLabel() []rune {
	if prsng == nil {
		return nil
	}

	return prsng.prelbl
}

func (prsng *Parsing) PostLabel() []rune {
	if prsng == nil {
		return nil
	}

	return prsng.postlbl
}

func (prsng *Parsing) CanPreParse() bool {
	if prsng == nil {
		return false
	}
	if txtprs := prsng.pretxtprs; txtprs == nil || !txtprs.isText() {
		return prsng.EventCanPreParse == nil || prsng.EventCanPreParse()
	}
	return prsng.EventCanPreParse == nil || prsng.EventCanPreParse()
}

func (prsng *Parsing) CanPostParse() bool {
	if prsng == nil {
		return true
	}
	if txtprs := prsng.posttxtprs; txtprs == nil || !txtprs.isText() {
		return prsng.EventCanPostParse == nil || prsng.EventCanPostParse()
	}
	return prsng.EventCanPostParse == nil || prsng.EventCanPostParse()
}

func (prsng *Parsing) Process() {
	if prsng == nil {
		return
	}
	parseReadRune(prsng.readRune, prsng)
}

func (prsng *Parsing) Close() (err error) {
	if prsng == nil {
		return
	}
	prsng.bufdrns = nil
	prsng.postlbl = nil
	prsng.prelbl = nil
	prsng.readRune = nil
	prsng.pretxtprs = nil
	prsng.posttxtprs = nil
	return
}

func New(in interface{}, prelbl, postlbl string, chkpretext bool, chkposttext bool, canPreParse func() bool, preRunes func(...rune), matchedPre func(), canPostParse func() bool, postRunes func(bool, ...rune) bool, matchedPost func() bool) (prsng *Parsing) {
	var readRune, _ = in.(func() (rune, int, error))
	if readRune == nil {
		if r, _ := in.(io.Reader); r != nil {
			rdnr, _ := r.(io.RuneReader)
			if rdnr == nil {
				rdnr = iorw.NewEOFCloseSeekReader(r)
			}
			readRune = rdnr.ReadRune
			goto prpparse
		}
		if rdnr, _ := in.(io.RuneReader); rdnr != nil {
			readRune = rdnr.ReadRune
		}
	}
	if readRune == nil {
		if s, sk := in.(string); sk {
			readRune = strings.NewReader(s).ReadRune
			goto prpparse
		}
		if int32arr, ink := in.([]int32); ink {
			readRune = strings.NewReader(string(int32arr)).ReadRune
			goto prpparse
		}
		if arrgs, argsk := in.([]interface{}); argsk {
			readRune = iorw.NewBuffer(arrgs...).Reader(true).ReadRune
		}
	}
prpparse:
	if prsng = nextparsing(prelbl, postlbl, func() *textparsing {
		if chkpretext {
			return &textparsing{}
		}
		return nil
	}(), func() *textparsing {
		if chkposttext {
			return &textparsing{}
		}
		return nil
	}(), readRune); prsng != nil {
		if canPreParse != nil {
			prsng.EventCanPreParse = canPreParse
		}
		if preRunes != nil {
			prsng.EventPreRunes = preRunes
		}
		if matchedPre != nil {
			prsng.EventMatchedPre = matchedPre
		}
		if canPostParse != nil {
			prsng.EventCanPostParse = canPostParse
		}
		if postRunes != nil {
			prsng.EventPostRunes = postRunes
		}
		if matchedPost != nil {
			prsng.EventMatchedPost = matchedPost
		}
	}
	return
}

func nextparsing(prelbl, postlbl string, pretxtprs *textparsing, posttxtprs *textparsing, readRune func() (rune, int, error)) (prsng *Parsing) {
	prsng = &Parsing{prelbl: []rune(prelbl), postlbl: []rune(postlbl), prvr: rune(0), pretxtprs: pretxtprs, posttxtprs: posttxtprs, readRune: readRune}
	prsng.postL = len(prsng.postlbl)
	prsng.preL = len(prsng.prelbl)
	return
}

func parseReadRune(readRune func() (r rune, size int, rerr error), prs ParseAPI) (err error) {
	if readRune == nil || prs == nil {
		return
	}
	for {
		r, s, rerr := readRune()
		if s > 0 {
			prs.Parse(r)
			continue
		}
		if s == 0 {
			if rerr == nil {
				err = io.EOF
				return
			}
		}
		if rerr != nil {
			err = rerr
			return
		}
	}
}

func (prsng *Parsing) Busy() bool {
	if prsng == nil {
		return false
	}
	return prsng.preL > 0 && prsng.postL > 0 && prsng.prei == prsng.preL && prsng.posti < prsng.postL
}

func (prsng *Parsing) Reset() {
	if prsng == nil {
		return
	}
	prsng.prei = 0
	prsng.posti = 0
	prsng.prvr = 0
}

func (prsng *Parsing) parse(r rune) {
	if prsng.posti == 0 && prsng.prei < prsng.preL {
		if pretxtprs := prsng.pretxtprs; pretxtprs != nil && pretxtprs.Parse(r) {
			if evtpre := prsng.EventPreRunes; evtpre != nil {
				evtpre(pretxtprs.lstr)
				prsng.prei = 0
				prsng.prvr = 0
			}
			return
		}
		if prsng.CanPreParse() {
			if prsng.prei > 0 && prsng.prelbl[prsng.prei-1] == prsng.prvr && prsng.prelbl[prsng.prei] != r {
				if evtpre := prsng.EventPreRunes; evtpre != nil {
					evtpre(prsng.prelbl[:prsng.prei]...)
				}
				prsng.prei = 0
				prsng.prvr = 0
			}
			if prsng.prelbl[prsng.prei] == r {
				prsng.prei++
				if prsng.prei == prsng.preL {
					if evtmtchpre := prsng.EventMatchedPre; evtmtchpre != nil {
						evtmtchpre()
					}
					return
				}
				prsng.prvr = r
				return
			}
		}
		if prsng.prei > 0 {
			if evtpre := prsng.EventPreRunes; evtpre != nil {
				evtpre(append(prsng.prelbl[:prsng.prei], r)...)
			}
			prsng.prei = 0
			prsng.prvr = 0
			return
		}

		if evtpre := prsng.EventPreRunes; evtpre != nil {
			prsng.prvr = r
			evtpre(r)
			return
		}
		prsng.prvr = r
		return
	}
	if prsng.prei == prsng.preL && prsng.posti < prsng.postL {
		if posttxtprs := prsng.posttxtprs; posttxtprs != nil && posttxtprs.Parse(r) {
			if evtpost := prsng.EventPostRunes; evtpost != nil {
				if evtpost(false, posttxtprs.lstr) {
					prsng.posti = 0
					prsng.prei = 0
					prsng.prvr = 0
				}
			}
			return
		}
		if prsng.CanPostParse() && prsng.postlbl[prsng.posti] == r {
			prsng.posti++
			if prsng.posti == prsng.postL {
				if evtmtchpost := prsng.EventMatchedPost; evtmtchpost != nil {
					evtmtchpost()
				}
				prsng.posti = 0
				prsng.prei = 0
				prsng.prvr = 0
				return
			}
			return
		}
		if prsng.posti > 0 {
			if evtpost := prsng.EventPostRunes; evtpost != nil {
				if evtpost(true, prsng.postlbl[:prsng.posti]...) {
					prsng.posti = 0
					prsng.prei = 0
					prsng.prvr = 0
					prsng.parse(r)
					return
				}
				prsng.posti = 0
				if evtpost(false, r) {
					prsng.posti = 0
					prsng.prei = 0
					prsng.prvr = 0
					return
				}
				return
			}
			prsng.posti = 0
			prsng.prei = 0
			prsng.prvr = 0
			return
		}
		if evtpost := prsng.EventPostRunes; evtpost != nil {
			if evtpost(false, r) {
				prsng.posti = 0
				prsng.prei = 0
				prsng.prvr = 0
				return
			}
		}
		return
	}
}

func (prsng *Parsing) Parse(rns ...rune) {
	if prsng == nil {
		return
	}
	if bufdrns := prsng.bufdrns; len(bufdrns) > 0 {
		prsng.bufdrns = nil
		rns = append(bufdrns, rns...)
	}

	for _, r := range rns {
		prsng.parse(r)
	}
}
