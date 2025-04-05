package ioext

import (
	"bufio"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lnksnk/lnksnk/iorw"
)

type MapReader interface {
	Map() map[string]interface{}
	ReadRune() (rune, int, error)
	String() string
}

type mapreader struct {
	eofrns   []rune
	eofi     int
	tstrns   []rune
	bfrdrne  func() (rune, int, error)
	mxtstl   int
	lstval   interface{}
	lstkey   []rune
	orgrdrne func() (rune, int, error)
	mp       map[string]interface{}
	pre      []rune
	prel     int
	post     []rune
	postl    int
}

func MapReplaceReader(in interface{}, mp map[string]interface{}, prepost ...string) MapReader {
	if mp == nil || in == nil {
		return nil
	}
	var readrune, _ = in.(func() (rune, int, error))
	if readrune == nil {
		if r, rk := in.(io.Reader); rk {
			rdr, _ := r.(io.RuneReader)
			if rdr == nil {
				rdr = bufio.NewReaderSize(r, 1)
			}
			readrune = rdr.ReadRune
		} else if s, sk := in.(string); sk {
			readrune = strings.NewReader(s).ReadRune
		} else if int32arr, intk := in.([]int32); intk {
			rns := make([]rune, len(int32arr))
			copy(rns, int32arr)
			readrune = iorw.ReadRuneFunc(func() (r rune, size int, err error) {
				if len(rns) > 0 {
					r = rns[0]
					rns = rns[1:]
					size = utf8.RuneLen(r)
					return
				}
				return 0, 0, io.EOF
			})
		}
	}
	mxtstl := 0
	for nme := range mp {
		if nml := len(nme); mxtstl < nml {
			mxtstl = nml
		}
	}
	mpr := &mapreader{mp: mp, orgrdrne: readrune, mxtstl: mxtstl}
	if prpstl := len(prepost); prpstl > 0 {
		if prpstl == 1 {
			mpr.pre = []rune(prepost[0])
			mpr.prel = len(mpr.pre)
		}
		if prpstl == 2 {
			mpr.pre = []rune(prepost[0])
			mpr.prel = len(mpr.pre)
			mpr.post = []rune(prepost[1])
			mpr.postl = len(mpr.post)
		}
	}
	return mpr
}

func (mpr *mapreader) String() (s string) {
	if mpr == nil {
		return
	}
	for r, si, rerr := mpr.ReadRune(); ; r, si, rerr = mpr.ReadRune() {
		if si > 0 {
			s += string(r)
		}
		if rerr != nil || si == 0 {
			break
		}
	}
	return
}

func (mpr *mapreader) Map() map[string]interface{} {
	if mpr == nil {
		return nil
	}
	if mp := mpr.mp; mp != nil {
		return mp
	}
	return nil
}

func (mpr *mapreader) ReadRune() (r rune, size int, err error) {
	if mpr == nil {
		return 0, 0, io.EOF
	}
	if eofl := len(mpr.eofrns); eofl > 0 {
		if mpr.eofi < eofl {
			r = mpr.eofrns[mpr.eofi]
			mpr.eofi++
			size = utf8.RuneLen(r)
			return
		}
		return 0, 0, io.EOF
	}
rdbuf:
	if mpr.bfrdrne != nil {
		r, size, err = mpr.bfrdrne()
		if size > 0 {
			return
		}
		mpr.bfrdrne = nil
	}
	if mpr.orgrdrne != nil {
		if mpr.prel > 0 {
			for {
			retrypre:
				r, size, err = mpr.orgrdrne()
				if size > 0 {
					if r == mpr.pre[len(mpr.tstrns)] {
						mpr.tstrns = append(mpr.tstrns, r)
						if len(mpr.tstrns) == mpr.prel {
							goto chkval
						}
						goto retrypre
					}
					if len(mpr.tstrns) == 0 {
						return
					}
					mpr.bfrdrne = iorw.NewRunesReader(append(mpr.tstrns, r)...).ReadRune
					mpr.tstrns = nil
					goto rdbuf
				}
				mpr.orgrdrne = nil
				if len(mpr.tstrns) == 0 {
					return
				}
				mpr.bfrdrne = iorw.NewRunesReader(mpr.tstrns...).ReadRune
				mpr.tstrns = nil
				goto rdbuf
			}
		}
	chkval:
		for {
			r, size, err = mpr.orgrdrne()
			if size == 0 {
				mpr.orgrdrne = nil
				mpr.bfrdrne = iorw.NewRunesReader(mpr.tstrns...).ReadRune
				mpr.tstrns = nil
				goto rdbuf
			}
			mpr.tstrns = append(mpr.tstrns, r)
			if mpr.postl > 0 {
				if tstl := len(mpr.tstrns[mpr.prel:]); tstl >= mpr.postl && string(mpr.tstrns[mpr.prel:][tstl-mpr.postl:]) == string(mpr.post) {
					if len(mpr.lstkey) > 0 {
						goto cptrval
					}
					//mpr.bfrdrne = iorw.NewRunesReader(mpr.tstrns...).ReadRune
					mpr.tstrns = nil
					goto rdbuf
				}
			}
			if len(mpr.tstrns[mpr.prel:]) <= mpr.mxtstl+mpr.postl {
				if val, cmthd := mpr.mp[string(mpr.tstrns[mpr.prel:])]; cmthd {
					mpr.lstkey = append([]rune{}, mpr.tstrns[mpr.prel:]...)
					mpr.lstval = val
					if len(mpr.lstkey) == mpr.mxtstl {
						if mpr.postl == 0 {
							goto cptrval
						}
						continue
					}
					continue
				}
				continue
			}
			mpr.bfrdrne = iorw.NewRunesReader(mpr.tstrns...).ReadRune
			mpr.tstrns = nil
			goto rdbuf
		}
	cptrval:
		if keyl, val := len(mpr.lstkey), mpr.lstval; keyl > 0 {
			mpr.lstkey = nil
			mpr.lstval = nil
			mpr.tstrns = nil
			if bfv, bfok := val.(*iorw.Buffer); bfok {
				if !bfv.Empty() {
					mpr.bfrdrne = bfv.Reader().ReadRune
					goto rdbuf
				}
			}
			if sv, sok := val.(string); sok {
				mpr.bfrdrne = strings.NewReader(sv).ReadRune
				goto rdbuf
			}
			if int32arr, arrok := val.([]int32); arrok {
				if arrl := len(int32arr); arrl > 0 {
					rns := make([]rune, arrl)
					copy(rns, int32arr)
					mpr.bfrdrne = iorw.NewRunesReader(rns...).ReadRune
				}
				goto rdbuf
			}
			if arrr, arrok := val.([]interface{}); arrok {
				mpr.bfrdrne = iorw.NewBuffer(arrr...).Reader(true).ReadRune
				goto rdbuf
			}
			mpr.bfrdrne = iorw.NewBuffer(val).Reader(true).ReadRune
			goto rdbuf
		}
		goto rdbuf
	}
	return
}
