package iorw

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type UntilRunesReader interface {
	RemainingRunes() []rune
	ReadRune() (rune, int, error)
	ReadLine() (string, error)
	ReadLines() ([]string, error)
	ReadAll() (string, error)
	Reset(eof ...interface{})
	Read([]rune) (int, error)
	FoundEOF() bool
}

type runesreaderuntil struct {
	eofrunes []rune
	eofdone  bool
	eofl     int
	eofi     int
	prveofr  rune
	p        []rune
	pl       int
	orgrdr   io.RuneReader
	intrmbuf []rune
	tmpbuf   []rune
	intrml   int
	tmpintrl int
	intrmi   int
	rmngrns  []rune
}

type ReadRuneFunc func() (rune, int, error)

func (rdrnefunc ReadRuneFunc) ReadRune() (rune, int, error) {
	return rdrnefunc()
}

type ReadFunc func(p []byte) (n int, err error)

func (rdfunc ReadFunc) Read(p []byte) (n int, err error) {
	return rdfunc(p)
}

func RunesReaderUntil(r interface{}, eof ...interface{}) (rdr UntilRunesReader) {
	var rd io.RuneReader = nil
	if rd, _ = r.(io.RuneReader); rd == nil {
		if rr, _ := r.(io.Reader); rr != nil {
			rd = bufio.NewReader(rr)
		}
	}
	rdr = &runesreaderuntil{orgrdr: rd}
	rdr.Reset(eof...)
	return
}

func (rdrrnsuntil *runesreaderuntil) FoundEOF() bool {
	if rdrrnsuntil != nil {
		return rdrrnsuntil.eofdone
	}
	return false
}

func (rdrrnsuntil *runesreaderuntil) RemainingRunes() []rune {
	if rdrrnsuntil != nil {
		return rdrrnsuntil.rmngrns
	}
	return []rune{}
}

func (rdrrnsuntil *runesreaderuntil) ReadRune() (r rune, size int, err error) {
	if rdrrnsuntil != nil {
		if rdrrnsuntil.pl == 0 {
			if len(rdrrnsuntil.p) == rdrrnsuntil.pl {
				rdrrnsuntil.pl = 8192
				rdrrnsuntil.p = make([]rune, rdrrnsuntil.pl)
			}
			if rdrrnsuntil.pl, err = rdrrnsuntil.Read(rdrrnsuntil.p[:rdrrnsuntil.pl]); err != nil {
				if rdrrnsuntil.pl == 0 && err == io.EOF {
					return
				}
				if rdrrnsuntil.pl > 0 && err == io.EOF {
					err = nil
				}
				if err != io.EOF {
					return
				}
			}
			rdrrnsuntil.p = rdrrnsuntil.p[:rdrrnsuntil.pl]
		}
		if rdrrnsuntil.pl > 0 {
			rdrrnsuntil.pl--
			r = rdrrnsuntil.p[0]
			rdrrnsuntil.p = rdrrnsuntil.p[1:]
			size = 1
			if r >= utf8.RuneSelf {
				size = utf8.RuneLen(r)
			}
		}
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) ReadLine() (ln string, err error) {
	if rdrrnsuntil != nil {
		ln, err = ReadLine(rdrrnsuntil)
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) ReadLines() (ln []string, err error) {
	if rdrrnsuntil != nil {
		ln, err = ReadLines(rdrrnsuntil)
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) ReadAll() (all string, err error) {
	if rdrrnsuntil != nil {
		all, err = ReaderToString(rdrrnsuntil)
	}
	return
}

func (rdrrnsuntil *runesreaderuntil) Reset(eof ...interface{}) {
	if rdrrnsuntil != nil {

		var eofrunes []rune = nil
		if len(eof) == 1 {
			if s, sok := eof[0].(string); sok && s != "" {
				eofrunes = []rune(s)
			} else {
				eofrunes, _ = eof[0].([]rune)
			}
		}
		eofl := len(eofrunes)
		if eofl == 0 {
			if eofl = len(rdrrnsuntil.eofrunes); eofl > 0 {
				eofrunes = append([]rune{}, rdrrnsuntil.eofrunes...)
			}
		}
		if rdrrnsuntil.eofdone = !(eofl > 0); !rdrrnsuntil.eofdone {
			if rdrrnsuntil.eofrunes != nil {
				rdrrnsuntil.eofrunes = nil
			}
			rdrrnsuntil.eofrunes = append([]rune{}, eofrunes...)
			if eofl > 0 {
				if rdrrnsuntil.intrmbuf == nil {
					rdrrnsuntil.intrmbuf = make([]rune, 8192)
				}
			}
			rdrrnsuntil.intrml = len(rdrrnsuntil.intrmbuf)
			rdrrnsuntil.intrmi = 0
			rdrrnsuntil.tmpintrl = 0
			rdrrnsuntil.eofl = eofl
			rdrrnsuntil.eofi = 0
			rdrrnsuntil.prveofr = 0
			rdrrnsuntil.tmpbuf = []rune{}
		}
	}
}

func (rdrrnsuntil *runesreaderuntil) Read(p []rune) (n int, err error) {
	if rdrrnsuntil != nil && !rdrrnsuntil.eofdone {
		if pl := len(p); pl > 0 && rdrrnsuntil.intrml > 0 {
			for tn, tb := range rdrrnsuntil.tmpbuf {
				p[n] = tb
				n++
				if n == pl {
					rdrrnsuntil.tmpbuf = rdrrnsuntil.tmpbuf[tn+1:]
					return
				}
			}
			if rdrrnsuntil.tmpintrl == 0 {
				rdrrnsuntil.intrmi = 0
				if rdrrnsuntil.tmpintrl, err = ReadRunes(rdrrnsuntil.intrmbuf, rdrrnsuntil.orgrdr); err == nil {
					return rdrrnsuntil.Read(p)
				}
			} else {
				tmpintrmbuf := rdrrnsuntil.intrmbuf[rdrrnsuntil.intrmi : rdrrnsuntil.intrmi+(rdrrnsuntil.tmpintrl-rdrrnsuntil.intrmi)]
				tmpintrmbufl := len(tmpintrmbuf)
				for bn, bb := range tmpintrmbuf {
					if rdrrnsuntil.eofi > 0 && rdrrnsuntil.eofrunes[rdrrnsuntil.eofi-1] == rdrrnsuntil.prveofr && rdrrnsuntil.eofrunes[rdrrnsuntil.eofi] != bb {
						tmpbuf := rdrrnsuntil.eofrunes[:rdrrnsuntil.eofi]
						rdrrnsuntil.eofi = 0
						for tn, tb := range tmpbuf {
							p[n] = tb
							n++
							if n == pl {
								if tn < len(tmpbuf)-1 {
									rdrrnsuntil.tmpbuf = append(rdrrnsuntil.tmpbuf, tmpbuf[tn+1:]...)
								}
								rdrrnsuntil.intrmi += bn + 1
								if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
									rdrrnsuntil.tmpintrl = 0
								}
								return
							}
						}
					}
					if rdrrnsuntil.eofrunes[rdrrnsuntil.eofi] == bb {
						rdrrnsuntil.eofi++
						if rdrrnsuntil.eofi == rdrrnsuntil.eofl {
							rdrrnsuntil.eofdone = true
							rdrrnsuntil.rmngrns = append([]rune{}, tmpintrmbuf[bn+1:]...)
							return
						} else {
							rdrrnsuntil.prveofr = bb
						}
					} else {
						if rdrrnsuntil.eofi > 0 {
							tmpbuf := rdrrnsuntil.eofrunes[:rdrrnsuntil.eofi]
							rdrrnsuntil.eofi = 0
							for tn, tb := range tmpbuf {
								p[n] = tb
								n++
								if n == pl {
									if tn < len(tmpbuf)-1 {
										rdrrnsuntil.tmpbuf = append(rdrrnsuntil.tmpbuf, tmpbuf[tn+1:]...)
									}
									rdrrnsuntil.intrmi += bn + 1
									if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
										rdrrnsuntil.tmpintrl = 0
									}
									return
								}
							}
						}
						rdrrnsuntil.prveofr = bb
						p[n] = bb
						n++
						if n == pl {
							rdrrnsuntil.intrmi += bn + 1
							if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
								rdrrnsuntil.tmpintrl = 0
							}
							return
						}
					}
					if bn == tmpintrmbufl-1 {
						rdrrnsuntil.intrmi += tmpintrmbufl
						if rdrrnsuntil.intrmi == rdrrnsuntil.tmpintrl {
							rdrrnsuntil.tmpintrl = 0
						}
					}
				}
			}
		}
	}

	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}

type ReadRunesUntilHandler interface {
	StartNextSearch()
	FoundPhrase(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error
	CanCheckRune(prvr, r rune) bool
}

type RunesUntilFunc func(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error) error

func (rdrnsuntlfunc RunesUntilFunc) Foundmatch(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error) error {
	return rdrnsuntlfunc(phrasefnd, untilrdr, orgrd, orgerr)
}

type RunesUntilFlushFunc func(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error, flushrdr SliceRuneReader) error

func (rdrnsuntlflshfunc RunesUntilFlushFunc) Foundmatch(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error, flushrdr SliceRuneReader) error {
	return rdrnsuntlflshfunc(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
}

type RunesUntilSliceFunc func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error) error

func (rdrnsuntlslcfunc RunesUntilSliceFunc) Foundmatch(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error) error {
	return rdrnsuntlslcfunc(phrasefnd, untilrdr, orgrd, orgerr)
}

type RunesUntilSliceFlushFunc func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error

func (rdrnsuntlslcflshfunc RunesUntilSliceFlushFunc) Foundmatch(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
	return rdrnsuntlslcflshfunc(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
}

func ReadRunesUntil(rdr interface{}, eof ...interface{}) io.RuneReader {
	if rdr == nil {
		return nil
	}
	orgrdr, _ := rdr.(io.RuneReader)
	if orgrdr == nil {
		if rdrd, _ := rdr.(io.Reader); rdrd != nil {
			orgrdr = NewEOFCloseSeekReader(rdrd)
		}
	}
	var startnxtsrch func()
	var cancheckrune func(prvr, r rune) bool
	var foundmatch func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error = nil
	var eofrunes [][]rune = nil
	oefmtchd := map[string]bool{}
	for _, eofd := range eof {
		if s, _ := eofd.(string); s != "" {
			if !oefmtchd[s] {
				eofrunes = append(eofrunes, []rune(s))
				oefmtchd[s] = true
			}
			continue
		}
		if arrs, _ := eofd.([]string); len(arrs) > 0 {
			for _, s := range arrs {
				if s != "" && !oefmtchd[s] {
					eofrunes = append(eofrunes, []rune(s))
					oefmtchd[s] = true
				}
			}
			continue
		}
		if int32s, _ := eofd.([]int32); len(int32s) > 0 {
			eofrns := make([]rune, len(int32s))
			copy(eofrns, int32s)
			if !oefmtchd[string(eofrns)] {
				eofrunes = append(eofrunes, eofrns)
				oefmtchd[string(eofrns)] = true
			}
			continue
		}
		if foundmatchd, _ := eofd.(func(phrase string, unitlrdr, orgrdr io.RuneReader, orgerr error) error); foundmatchd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
		}
		if foundmatchflshd, _ := eofd.(func(phrasefnd string, untilrdr, orgrdr io.RuneReader, orgerr error, flushrdr SliceRuneReader) error); foundmatchflshd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchflshd(phrasefnd, untilrdr, orgrdr, orgerr, flushrdr)
				}
			}
			continue
		}
		if foundmatchslcd, _ := eofd.(func(phrase string, unitlrdr io.RuneReader, orgrdr SliceRuneReader, orgerr error) error); foundmatchslcd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchslcd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
		}
		if foundmatchslcflshd, _ := eofd.(func(phrasefnd string, untilrdr io.RuneReader, orgrdr SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error); foundmatchslcflshd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchslcflshd(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
				}
			}
			continue
		}
		if foundmatchfncd, _ := eofd.(RunesUntilFunc); foundmatchfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchfncd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
			continue
		}
		if foundmatchflshfncd, _ := eofd.(RunesUntilFlushFunc); foundmatchflshfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchflshfncd(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
				}
			}
			continue
		}
		if foundmatchslcfncd, _ := eofd.(RunesUntilSliceFunc); foundmatchslcfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchslcfncd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
			continue
		}
		if foundmatchslcflshfncd, _ := eofd.(RunesUntilSliceFlushFunc); foundmatchslcflshfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return foundmatchslcflshfncd(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
				}
			}
			continue
		}
		if startnxtsrchd, _ := eofd.(func()); startnxtsrchd != nil {
			if startnxtsrch == nil {
				startnxtsrch = startnxtsrchd
			}
			continue
		}
		if cancheckruned, _ := eofd.(func(rune, rune) bool); cancheckruned != nil {
			if cancheckrune == nil {
				cancheckrune = cancheckruned
			}
			continue
		}
		if rdrnseofhndlrd, _ := eofd.(ReadRunesUntilHandler); rdrnseofhndlrd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd SliceRuneReader, orgerr error, flushrdr SliceRuneReader) error {
					return rdrnseofhndlrd.FoundPhrase(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
				}
			}
			if startnxtsrch == nil {
				startnxtsrch = rdrnseofhndlrd.StartNextSearch
			}
			if cancheckrune == nil {
				cancheckrune = rdrnseofhndlrd.CanCheckRune
			}
		}
	}
	if eofl := len(eofrunes); eofl > 0 && foundmatch != nil && orgrdr != nil {
		orgslcrdr, _ := orgrdr.(SliceRuneReader)
		if orgslcrdr == nil {
			orgslcrdr = NewSliceRuneReader(orgrdr)
		}
		if cancheckrune == nil {
			cancheckrune = func(prvr, r rune) bool { return true }
		}
		lsteofphrse := ""
		var rnsrdr io.RuneReader = nil

		bfrdrns := []rune{}
		orgbfrns := []rune{}
		mthchdrns := []rune{}
		preorgbfrns := []rune{}
		prvr := rune(0)

		var mtcheofs = map[string]int{}
		strsrch := false

		var flshrdr SliceRuneReader

		var nextorgr = func() (r rune, size int, err error) {
			if len(orgbfrns) > 0 {
				r = orgbfrns[0]
				size = utf8.RuneLen(r)
				orgbfrns = orgbfrns[1:]
				return
			}
			r, size, err = orgrdr.ReadRune()
			return
		}

		rnsrdr = ReadRuneFunc(func() (r rune, size int, err error) {
			if flshrdr != nil && !flshrdr.Empty() {
				r, size, err = flshrdr.ReadRune()
				if size > 0 {
					return
				}
				if err != nil && err != io.EOF {
					return
				}
				err = nil
			}
			if len(bfrdrns) > 0 {
				r = bfrdrns[0]
				bfrdrns = bfrdrns[1:]
				size = utf8.RuneLen(r)
				return
			}
			if !strsrch && startnxtsrch != nil {
				strsrch = true
				startnxtsrch()
			}
			r, size, err = nextorgr()
			if size > 0 {
				if !cancheckrune(prvr, r) {
					return r, size, err
				}
				msteofi := -1
				for eofi := range len(eofrunes) {
					if eofl := len(eofrunes[eofi]); eofrunes[eofi][0] == r {
						if eofl == 1 {
							msteofi = eofi
							//didmtch = true
						}
						mtcheofs[string(eofrunes[eofi])] = eofi
					}
				}
				prvr = r
				if len(mtcheofs) == 0 {
					return r, size, err
				}
				if len(mtcheofs) == 1 && msteofi > -1 {
					clear(mtcheofs)
					strsrch = false
					prvr = 0
					if flshrdr == nil {
						flshrdr = NewSliceRuneReader()
					}
					if fnderr := foundmatch(string(eofrunes[msteofi]), rnsrdr, orgslcrdr, err, flshrdr); fnderr != nil {
						return 0, 0, fnderr
					}
					return rnsrdr.ReadRune()

				}

				mthchdrns = nil
				preorgbfrns = nil
				lsteofphrse = ""
				for err == nil {
					mthchdrns = append(mthchdrns, r)
					r, size, err = nextorgr()
					if size > 0 {
						if err == io.EOF {
							err = nil
						}

						mxeofkl := 0
						chdk := false
						for eofk, eofi := range mtcheofs {
							eofkl := len(eofk)
							if mtchl := len(mthchdrns) + 1; eofkl >= mtchl {
								if []rune(eofk)[mtchl-1] == r {
									orgbfrns = nil
									chdk = true
									if eofkl == mtchl {
										for ek, eki := range mtcheofs {
											if ekl := len(ek); eki != eofi {
												if ekl < eofkl || (ek[:eofkl] != eofk[:eofkl]) {
													if ekl > mxeofkl {
														mxeofkl = ekl
													}
													delete(mtcheofs, ek)
													continue
												}
											}
										}
										lsteofphrse = eofk
										delete(mtcheofs, eofk)
									}
									continue
								}
							}
							if eofkl > mxeofkl {
								mxeofkl = eofkl
							}
							delete(mtcheofs, eofk)
						}
						if !chdk {
							preorgbfrns = append(preorgbfrns, r)
						}
						if len(mtcheofs) == 0 {
							prvr = 0
							orgbfrns = append(orgbfrns, preorgbfrns...)
							if lsteofphrse == "" {
								bfrdrns = append(bfrdrns, mthchdrns...)
								r, size, err = rnsrdr.ReadRune()
								return
							}
							strsrch = false
							if flshrdr == nil {
								flshrdr = NewSliceRuneReader()
							}
							if fnderr := foundmatch(lsteofphrse, rnsrdr, orgslcrdr, err, flshrdr); fnderr != nil {
								return 0, 0, fnderr
							}
							return rnsrdr.ReadRune()
						}
						continue
					}
					bfrdrns = append(bfrdrns, mthchdrns...)
					if err != nil {
						if err != io.EOF {
							err = nil
						}
						return
					}
					return rnsrdr.ReadRune()
				}
			}

			if size == 0 && err == nil {
				err = io.EOF
			}
			return
		})
		return rnsrdr
	}
	return orgrdr
}
