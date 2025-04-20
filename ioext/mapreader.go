package ioext

import (
	"bufio"
	"io"
	"strings"
	"unicode/utf8"
)

type MapReader interface {
	Map() map[string]interface{}
	ReadRune() (rune, int, error)
	String() string
	Runes() []rune
}

type IString interface {
	String() string
}

type StringFunc func() string

func (strngfnc StringFunc) String() string {
	return strngfnc()
}

type IRunes interface {
	Runes() []rune
}

type RunesFunc func() []rune

func (rnsfnc RunesFunc) Runes() []rune {
	return rnsfnc()
}

type IMAP interface {
	Map() map[string]interface{}
}

type MAPFunc func() map[string]interface{}

func (mpfnc MAPFunc) Map() map[string]interface{} {
	return mpfnc()
}

var (
	emptyMapReader = &struct {
		IMAP
		io.RuneReader
		IString
		IRunes
	}{IMAP: MAPFunc(func() map[string]interface{} {
		return nil
	}), RuneReader: ReadRuneFunc(func() (rune, int, error) {
		return 0, 0, io.EOF
	}), IString: StringFunc(func() string {
		return ""
	}), IRunes: RunesFunc(func() []rune {
		return nil
	})}
)

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
			if s == "" {
				return emptyMapReader
			}
			readrune = strings.NewReader(s).ReadRune
		} else if int32arr, intk := in.([]int32); intk {
			if len(int32arr) == 0 {
				return emptyMapReader
			}
			rns := make([]rune, len(int32arr))
			copy(rns, int32arr)
			readrune = ReadRuneFunc(func() (r rune, size int, err error) {
				if len(rns) > 0 {
					r = rns[0]
					rns = rns[1:]
					size = utf8.RuneLen(r)
					return
				}
				return 0, 0, io.EOF
			})
		} else if bf, bfk := in.(*Buffer); bfk {
			if bf.Empty() {
				return emptyMapReader
			} else {
				readrune = bf.Reader().ReadRune
			}
		}
	}
	if readrune == nil {
		return emptyMapReader
	}
	mxtstl := 0
	mintsl := 0
	var bfrdrne []rune
	var bfrnrdr io.RuneReader
	var orgrns []rune
	var orgbfrdrunes = make([]rune, 8192)
	var orgbdwi = 0
	var orgbdri = 0
	var orgrnrd func() (rune, int, error) = readrune

	readOrgRune := func() (or rune, osize int, oerr error) {
		if len(orgrns) > 0 {
			or = orgrns[0]
			orgrns = orgrns[1:]
			osize = utf8.RuneLen(or)
			return or, osize, nil
		}
	rdorgbdr:
		if orgbdri < orgbdwi {
			orgbdri++
			or = orgbfrdrunes[orgbdri-1]
			if orgbdri == orgbdwi {
				orgbdri = 0
				orgbdwi = 0
			}
			return or, utf8.RuneLen(or), nil
		}
		if orgrnrd != nil {
			for orgbdwi < len(orgbfrdrunes) {
				if or, osize, oerr = orgrnrd(); osize > 0 {
					if or == 65279 {
						//handling ZERO WIDTH N)-BREAKSPACE
						continue
					}
					orgbfrdrunes[orgbdwi] = or
					orgbdwi++
					if orgbdwi == len(orgbfrdrunes) {
						goto rdorgbdr
					}
					continue
				}
				if oerr == nil {
					oerr = io.EOF
				}
				if oerr != nil {
					orgrnrd = nil
				}
				if orgbdwi > 0 {
					goto rdorgbdr
				}
				return
			}
			if oerr == nil {
				oerr = io.EOF
			}
			if oerr != nil {
				orgrnrd = nil
			}
			return
		}
		return 0, 0, io.EOF
	}

	mpkeys := [][]rune{}
	mpidx := []interface{}{}

	for nme, nmv := range mp {
		mpkeys = append(mpkeys, []rune(nme))
		nml := len(mpkeys[len(mpkeys)-1])
		mpidx = append(mpidx, nmv)
		if mxtstl < nml {
			mxtstl = nml
		}
		if mintsl == 0 || mintsl > nml {
			mintsl = nml
		}
	}

	var mprdr *struct {
		IMAP
		io.RuneReader
		IString
		IRunes
	}
	var finalReader func() (rune, int, error)
	var lstkey []rune
	var lstval interface{}
	var cptureval = func() bool {
		bfl := len(bfrdrne)
		if fncv, fnck := lstval.(func() interface{}); fnck {
			lstval = fncv()
		}
		cptrv := lstval
		lstkey = nil
		lstval = nil
		if cptrs, cpctsk := cptrv.(string); cpctsk {
			if cptrs != "" {
				bfrdrne = append([]rune(cptrs), bfrdrne...)
				return true
			}
			if bfl > 0 {
				return true
			}
			return false
		}
		if cptrint32arr, cptrk := cptrv.([]int32); cptrk {
			if cptrl := len(cptrint32arr); cptrl > 0 {
				cptrns := make([]rune, cptrl)
				copy(cptrns, cptrint32arr)
				bfrdrne = append(cptrns, bfrdrne...)
				return true
			}
			if bfl > 0 {
				return true
			}
			return false
		}
		if cptrbf, cprbfk := cptrv.(*Buffer); cprbfk {
			if cptrbf.Empty() {
				return bfl > 0
			}
			bfrnrdr = cptrbf.Reader()
			return true
		}
		if cptr, cprk := cptrv.(io.Reader); cprk {
			if rnrdr, rnrdrk := cptr.(io.RuneReader); rnrdrk {
				bfrnrdr = rnrdr
				return true
			}
			bfrnrdr = bufio.NewReader(cptr)
			return true
		}
		return bfl > 0
	}

	prpstl := len(prepost)
	ki := 0
	keyi := -1
	if prpstl == 0 || (prpstl == 1 && prepost[0] == "") || (prpstl >= 2 && prepost[0] == "" && prepost[1] == "") || prpstl > 2 {

		finalReader = ReadRuneFunc(func() (fr rune, fsize int, ferr error) {
		rdbf:
			if bfrnrdr != nil {
				fr, fsize, ferr = bfrnrdr.ReadRune()
				if fsize > 0 {
					return fr, fsize, nil
				}
				bfrnrdr = nil
				if ferr != nil {
					if ferr != io.EOF {
						return 0, 0, ferr
					}
					ferr = nil
				}
			}
			if len(bfrdrne) > 0 {
				fr = bfrdrne[0]
				bfrdrne = bfrdrne[1:]
				return fr, utf8.RuneLen(fr), nil
			}
			if readOrgRune == nil {
				return 0, 0, io.EOF
			}
			for ferr == nil {
			rdnxt:
				if fr, fsize, ferr = readOrgRune(); fsize > 0 {
					for kn, k := range mpkeys {
						if kl := len(k); kl > ki {
							if k[ki] == fr {
								ki++
								if kl == ki {
									lstkey = k
									lstval = mpidx[kn]
									if kl == mxtstl {
										ki = 0
										if cptureval() {
											goto rdbf
										}
									}
								}
								goto rdnxt
							}
						}
					}
					if len(lstkey) > 0 {
						bfrdrne = append(bfrdrne, fr)
						ki = 0
						if cptureval() {
							goto rdbf
						}
						goto rdnxt
					}
					return fr, fsize, nil
				}
				if ferr == nil {
					ferr = io.EOF
				}
				if ferr == io.EOF {
					readOrgRune = nil
					ki = 0
					if len(lstkey) > 0 {
						if cptureval() {
							ferr = nil
							goto rdbf
						}
					}
				}
				return 0, 0, ferr
			}
			return
		})
	}

	if prpstl >= 2 && prepost[0] != "" && prepost[1] != "" {
		pre := []rune(prepost[0])
		prel := len(pre)
		prei := 0
		pvr := rune(0)
		post := []rune(prepost[1])
		postl := len(post)
		posti := 0
		unmtchd := false

		finalReader = func() (fr rune, fsize int, ferr error) {
		rdbf:
			if bfrnrdr != nil {
				fr, fsize, ferr = bfrnrdr.ReadRune()
				if fsize > 0 {
					return fr, fsize, nil
				}
				bfrnrdr = nil
				if ferr != nil {
					if ferr != io.EOF {
						return 0, 0, ferr
					}
					if len(bfrdrne) > 0 {
						fr = bfrdrne[0]
						bfrdrne = bfrdrne[1:]
						return fr, utf8.RuneLen(fr), nil
					}
				}
			} else if len(bfrdrne) > 0 {
				fr = bfrdrne[0]
				bfrdrne = bfrdrne[1:]
				return fr, utf8.RuneLen(fr), nil
			}

			for ferr == nil && readOrgRune != nil {
			rdnxt:
				fr, fsize, ferr = readOrgRune()
				if fsize > 0 {
					if posti == 0 && prei < prel {
						if prei > 0 && pre[prei-1] == pvr && pre[prei] != fr {
							bfrdrne = append(bfrdrne, pre[:prei]...)
							prei = 0
							pvr = 0
							orgrns = append(orgrns, fr)
							goto rdbf
						}
						if pre[prei] == fr {
							prei++
							if prei == prel {
								continue
							}
							pvr = fr
							continue
						} else if prei > 0 {
							bfrdrne = append(bfrdrne, pre[:prei]...)
							bfrdrne = append(bfrdrne, fr)
							prei = 0
							pvr = 0
							goto rdbf
						}
						pvr = fr
						return fr, fsize, nil
					} else if prei == prel && posti < postl {
						if post[posti] == fr {
							posti++
							if posti == postl {
								posti = 0
								prei = 0
								pvr = 0
								if unmtchd {
									ki = 0
									bfrdrne = nil
									continue
								}
								if len(lstkey) > 0 {
									ki = 0
									keyi = -1
									if cptureval() {
										goto rdbf
									}
									continue
								}
								continue
							}
							continue
						} else if posti > 0 {
							ki = 0
							bfrdrne = append(bfrdrne, post[:posti]...)
							posti = 0
							orgrns = append(orgrns, fr)

							if len(lstkey) > 0 {
								if unmtchd {
									unmtchd = false
								}
								bfrdrne = append(bfrdrne, lstkey...)
								lstkey = nil
								lstval = nil
								goto rdbf
							}
							if unmtchd {
								unmtchd = false
							}
							goto rdbf
						}
						if unmtchd {
							bfrdrne = append(bfrdrne, fr)
							if ki++; ki >= mxtstl+len(post) {
								bfrdrne = append(pre[:], bfrdrne...)
								prei = 0
								posti = 0
								ki = 0
								unmtchd = false
								goto rdbf
							}
							continue
						}
						for kn, k := range mpkeys {
							if kl := len(k); kl > ki {
								if k[ki] == fr {
									keyi = kn
									ki++
									if kl == ki {
										lstkey = k
										lstval = mpidx[kn]
										if kl == mxtstl {
											goto rdnxt
										}
									}
									goto rdnxt
								}
							}
						}
						if len(lstkey) == 0 {
							unmtchd = true
							if keyi > -1 {
								bfrdrne = append(bfrdrne, mpkeys[keyi][:ki]...)
								keyi = -1
							}
							bfrdrne = append(bfrdrne, fr)
							ki++
							continue
						}
						goto rdbf
					}
					continue
				}
				if ferr == nil {
					ferr = io.EOF
				}
				readOrgRune = nil
				if ferr != io.EOF {
					return
				}
				if prei > 0 {
					bfrdrne = append(bfrdrne, pre[:prei]...)
					prei = 0
				}
				if keyi > -1 {
					bfrdrne = append(bfrdrne, mpkeys[keyi][:ki]...)
				}
				ki = 0
				if posti > 0 {
					bfrdrne = append(bfrdrne, post[:posti]...)
					posti = 0
				}
				if len(bfrdrne) > 0 {
					goto rdbf
				}
				return 0, 0, io.EOF
			}
			return 0, 0, io.EOF
		}
	}

	if finalReader != nil {
		mprdr = &struct {
			IMAP
			io.RuneReader
			IString
			IRunes
		}{IMAP: MAPFunc(func() map[string]interface{} {
			return mp
		}),
			RuneReader: ReadRuneFunc(finalReader),
			IString: StringFunc(func() (s string) {
				if finalReader == nil {
					return
				}
				for {
					r, si, rerr := finalReader()
					if si > 0 {
						s += string(r)
						continue
					}
					if rerr != nil || si == 0 {
						break
					}
				}
				return
			}),
			IRunes: RunesFunc(func() (rns []rune) {
				if finalReader == nil {
					return
				}
				for {
					r, si, rerr := finalReader()
					if si > 0 {
						rns = append(rns, r)
						continue
					}
					if rerr != nil || si == 0 {
						break
					}
				}
				return
			})}
		return mprdr
	}
	return emptyMapReader
}

/*
	func (mpr *mapreader) ReadRune() (r rune, size int, err error) {
		if mpr == nil {
			return 0, 0, io.EOF
		}

rdbuf:

		if bfrdr := mpr.bfrnrdr; bfrdr != nil {
			r, size, err = bfrdr.ReadRune()
			if size > 0 {
				return
			}
			if err != nil {
				if err != io.EOF {
					return
				}
				err = nil
			}
		} else if bfl := len(mpr.bfrdrne); bfl > 0 {
			r = mpr.bfrdrne[0]
			mpr.bfrdrne = mpr.bfrdrne[1:]
			size = utf8.RuneLen(r)
			if size > 0 {
				return r, size, nil
			}
		}

		if mp, mpidx, mpkeys, orgrdrne := mpr.mp, mpr.mpidx, mpr.mpkeys, mpr.orgrdrne; orgrdrne != nil {
		rdorg:
			r, size, err = orgrdrne()
			if size == 0 || err != nil {
				if err == nil {
					err = io.EOF
				}
				if err != nil {
					if err == io.EOF {
						if mpr.prei == 0 && mpr.posti == 0 {
							if len(mpr.lstkey) > 0 {
								mpr.orgrdrne = nil
								mpr.tki = 0
								mpr.bfrdrne = nil
								goto cptrval
							}
						}
						mpr.orgrdrne = nil
						if mpr.prei > 0 {
							mpr.bfrdrne = append(mpr.bfrdrne, mpr.pre[:mpr.prei]...)
							mpr.prei = 0
						}
						if mpr.tstl > 0 {
							mpr.bfrdrne = append(mpr.bfrdrne, mpr.tstrns[:mpr.tstl]...)
							mpr.tstl = 0
							mpr.tstrns = nil
						}
						if mpr.posti > 0 {
							mpr.bfrdrne = append(mpr.bfrdrne, mpr.post[:mpr.posti]...)
							mpr.posti = 0
						}
						goto rdbuf
					}
				}
				return
			}
			if mpr.prel == 0 && mpr.postl == 0 {
				for kn := range mpkeys {
					krns := mpkeys[kn]
					if kl := len(krns); kl > mpr.tki {
						if krns[mpr.tki] == r {
							mpr.tki++
							if kl == mpr.tki {
								mpr.lstkey = krns
								mpr.lstval = mpidx[kn]
								if kl == mpr.mxtstl {
									mpr.tki = 0
									goto cptrval
								}
							}
							goto rdorg
						}
					}
				}
				if len(mpr.lstkey) > 0 {
					mpr.bfrdrne = append(mpr.bfrdrne, r)
					mpr.tki = 0
					goto cptrval
				}
				return r, size, nil
			}
			if mpr.posti == 0 && mpr.prei < mpr.prel {
				if mpr.prei > 0 && mpr.pre[mpr.prei-1] == mpr.pvr && mpr.pre[mpr.prei] != r {
					mpr.bfrdrne = append(mpr.bfrdrne, mpr.pre[:mpr.prei]...)
					mpr.prei = 0
					mpr.orgrne = append(mpr.orgrne, r)
					mpr.pvr = 0
					goto rdbuf
				}
				if mpr.pre[mpr.prei] == r {
					mpr.prei++
					if mpr.prei == mpr.prel {

						goto rdorg
					}
					mpr.pvr = r
					goto rdorg
				}
				if mpr.prei > 0 {
					mpr.bfrdrne = append(mpr.bfrdrne, mpr.pre[:mpr.prei]...)
					mpr.bfrdrne = append(mpr.bfrdrne, r)
					mpr.prei = 0
					goto rdbuf
				}
				return r, size, nil
			}
			if mpr.prei == mpr.prel && mpr.posti < mpr.postl {
				if mpr.post[mpr.posti] == r {
					mpr.posti++
					if mpr.posti == mpr.postl {
						mpr.posti = 0
						mpr.prei = 0
						mpr.pvr = 0
						if len(mpr.lstkey) > 0 {
							mpr.tstl = 0
							mpr.tstrns = nil
							goto cptrval
						}

						if mpr.tstl == 0 {
							goto rdbuf
						}
						mpr.tstrns = nil
						mpr.tstl = 0
						goto rdbuf
					}
					goto rdorg
				}
				if mpr.posti > 0 {
					if len(mpr.lstkey) > 0 {
						mpr.lstkey = nil
						mpr.lstval = nil
					}
					mpr.bfrdrne = append(mpr.bfrdrne, mpr.pre...)
					mpr.prei = 0
					if mpr.tstl > 0 {
						mpr.bfrdrne = append(mpr.bfrdrne, mpr.tstrns...)
						mpr.tstrns = nil
						mpr.tstl = 0
					}
					mpr.bfrdrne = append(mpr.bfrdrne, mpr.post[:mpr.posti]...)
					mpr.posti = 0
					mpr.orgrne = append(mpr.orgrne, r)
					goto rdorg
				}

				mpr.tstrns = append(mpr.tstrns, r)
				mpr.tstl++
				if mpr.tstl >= mpr.mintsl {
					for k, v := range mp {
						kl := len([]rune(k))
						if krns := []rune(k); kl >= mpr.tstl && string(krns[:mpr.tstl]) == string(mpr.tstrns[:mpr.tstl]) {
							if kl == mpr.tstl {
								mpr.lstkey = krns
								mpr.lstval = v
							}
							goto rdorg
						}
					}
				}
				if lstk := len(mpr.lstkey); mpr.tstl >= mpr.mintsl {
					if lstk == 0 {
						mpr.bfrdrne = append(mpr.bfrdrne, mpr.pre...)
						mpr.prei = 0
						mpr.bfrdrne = append(mpr.bfrdrne, mpr.tstrns...)
						mpr.tstrns = nil
						mpr.tstl = 0
						goto rdbuf
					}
					if mpr.tstl > mpr.mxtstl {
						mpr.bfrdrne = append(mpr.bfrdrne, mpr.pre...)
						mpr.prei = 0
						mpr.bfrdrne = append(mpr.bfrdrne, mpr.lstkey...)
						mpr.lstkey = nil
						mpr.lstval = nil
						mpr.tstrns = nil
						mpr.tstl = 0
						goto rdbuf
					}
				}
				goto rdorg
			}
		cptrval:
			bfl := len(mpr.bfrdrne)
			if fncv, fnck := mpr.lstval.(func() interface{}); fnck {
				mpr.lstval = fncv()
			}
			cptrv := mpr.lstval
			mpr.lstkey = nil
			mpr.lstval = nil
			if cptrs, cpctsk := cptrv.(string); cpctsk {
				if cptrs != "" {
					mpr.bfrdrne = append([]rune(cptrs), mpr.bfrdrne...)
					goto rdbuf
				}
				if bfl > 0 {
					goto rdbuf
				}
				goto rdorg
			}
			if cptrint32arr, cptrk := cptrv.([]int32); cptrk {
				if cptrl := len(cptrint32arr); cptrl > 0 {
					cptrns := make([]rune, cptrl)
					copy(cptrns, cptrint32arr)
					mpr.bfrdrne = append(cptrns, mpr.bfrdrne...)
					goto rdbuf
				}
				if bfl > 0 {
					goto rdbuf
				}
				goto rdorg
			}
			if cptrbf, cprbfk := cptrv.(*Buffer); cprbfk {
				if cptrbf.Empty() {
					if bfl > 0 {
						goto rdbuf
					}
					goto rdorg
				}
				if bfl > 0 {
					mpr.bfrnrdr = NewSliceRuneReader(cptrbf.Reader(), NewRunesReader(mpr.bfrdrne...))
					mpr.bfrdrne = nil
					goto rdbuf
				}
				mpr.bfrnrdr = cptrbf.Reader()
				goto rdbuf
			}
			if cptr, cprk := cptrv.(io.Reader); cprk {
				if rnrdr, rnrdrk := cptr.(io.RuneReader); rnrdrk {
					if bfl > 0 {
						mpr.bfrnrdr = NewSliceRuneReader(rnrdr, NewRunesReader(mpr.bfrdrne...))
						mpr.bfrdrne = nil
						goto rdbuf
					}
					mpr.bfrnrdr = rnrdr
					goto rdbuf
				}
				if bfl > 0 {
					mpr.bfrnrdr = NewSliceRuneReader(bufio.NewReader(cptr), NewRunesReader(mpr.bfrdrne...))
					mpr.bfrdrne = nil
					goto rdbuf
				}
				mpr.bfrnrdr = bufio.NewReader(cptr)
				goto rdbuf
			}
			goto rdbuf
		}
		if size == 0 && err == nil {
			err = io.EOF
		}
		return
	}
*/
func init() {

	/*for {
		tst := MapReplaceReader("{@dat@}{@dat@} {@dat@}{@dat@} {@dat@}", map[string]interface{}{
			"{@": "${", "@}": "}", "@}{@": "}${"})
		if nxts := tst.String(); nxts != "${dat}${dat} ${dat}${dat} ${dat}" {
			fmt.Println(nxts)
		}
	}*/

}
