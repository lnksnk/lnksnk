package iorw

import (
	"bufio"
	"fmt"
	"io"
	"strings"
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
	FoundPhrase(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error
	CanCheckRune(prvr, r rune) bool
}

type RunesUntilFunc func(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error) error

func (rdrnsuntlfunc RunesUntilFunc) Foundmatch(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error) error {
	return rdrnsuntlfunc(phrasefnd, untilrdr, orgrd, orgerr)
}

type RunesUntilFlushFunc func(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error, flushrdr *RuneReaderSlice) error

func (rdrnsuntlflshfunc RunesUntilFlushFunc) Foundmatch(phrasefnd string, untilrdr, orgrd io.RuneReader, orgerr error, flushrdr *RuneReaderSlice) error {
	return rdrnsuntlflshfunc(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
}

type RunesUntilSliceFunc func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error) error

func (rdrnsuntlslcfunc RunesUntilSliceFunc) Foundmatch(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error) error {
	return rdrnsuntlslcfunc(phrasefnd, untilrdr, orgrd, orgerr)
}

type RunesUntilSliceFlushFunc func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error

func (rdrnsuntlslcflshfunc RunesUntilSliceFlushFunc) Foundmatch(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
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
	var foundmatch func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error = nil
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
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
		}
		if foundmatchflshd, _ := eofd.(func(phrasefnd string, untilrdr, orgrdr io.RuneReader, orgerr error, flushrdr *RuneReaderSlice) error); foundmatchflshd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchflshd(phrasefnd, untilrdr, orgrdr, orgerr, flushrdr)
				}
			}
			continue
		}
		if foundmatchslcd, _ := eofd.(func(phrase string, unitlrdr io.RuneReader, orgrdr *RuneReaderSlice, orgerr error) error); foundmatchslcd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchslcd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
		}
		if foundmatchslcflshd, _ := eofd.(func(phrasefnd string, untilrdr io.RuneReader, orgrdr *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error); foundmatchslcflshd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchslcflshd(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
				}
			}
			continue
		}
		if foundmatchfncd, _ := eofd.(RunesUntilFunc); foundmatchfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchfncd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
			continue
		}
		if foundmatchflshfncd, _ := eofd.(RunesUntilFlushFunc); foundmatchflshfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchflshfncd(phrasefnd, untilrdr, orgrd, orgerr, flushrdr)
				}
			}
			continue
		}
		if foundmatchslcfncd, _ := eofd.(RunesUntilSliceFunc); foundmatchslcfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
					return foundmatchslcfncd(phrasefnd, untilrdr, orgrd, orgerr)
				}
			}
			continue
		}
		if foundmatchslcflshfncd, _ := eofd.(RunesUntilSliceFlushFunc); foundmatchslcflshfncd != nil {
			if foundmatch == nil {
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
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
				foundmatch = func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) error {
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
		orgslcrdr, _ := orgrdr.(*RuneReaderSlice)
		if orgslcrdr == nil {
			orgslcrdr = NewRuneReaderSlice(orgrdr)
		}
		if cancheckrune == nil {
			cancheckrune = func(prvr, r rune) bool { return true }
		}
		lsteofphrse := ""
		tstphrse := ""
		var rnsrdr io.RuneReader = nil
		//ri := 0
		bfrdrns := []rune{}
		prvr := rune(0)
		noorg := false
		var mtcheofs = map[string]int{}
		strsrch := false

		var flshrdr *RuneReaderSlice

		rnsrdr = ReadRuneFunc(func() (r rune, size int, err error) {
			if !flshrdr.Empty() {
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
			for !noorg {
			rdfndeofs:
				if !strsrch && startnxtsrch != nil {
					strsrch = true
					startnxtsrch()
				}
				r, size, err = orgrdr.ReadRune()
				if size > 0 {
					if err == nil || err == io.EOF {
						if len(mtcheofs) > 0 {
							tstphrse += string(r)
							tstphrsl := len(tstphrse)
							for eofk, eofi := range mtcheofs {
								if eofkl := len(eofk); tstphrsl <= eofkl {
									if eofk[:tstphrsl] == tstphrse {
										if tstphrsl == eofkl {
											for ek, eki := range mtcheofs {
												if ekl := len(ek); eki != eofi {
													if ekl < eofkl || (ek[:eofkl] != eofk[:eofkl]) {
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
								delete(mtcheofs, eofk)
							}
							if len(mtcheofs) == 0 {
								prvr = 0
								if lsteofphrse == "" {
									bfrdrns = []rune(tstphrse)
									tstphrse = ""
									return rnsrdr.ReadRune()
								}
								strsrch = false
								if fnderr := foundmatch(lsteofphrse, rnsrdr, orgslcrdr, err, func() *RuneReaderSlice {
									if flshrdr != nil {
										return flshrdr
									}
									flshrdr = NewRuneReaderSlice()
									return flshrdr
								}()); fnderr != nil {
									return 0, 0, fnderr
								}
								return rnsrdr.ReadRune()
							}
							goto rdfndeofs
						}
						if !cancheckrune(prvr, r) {
							return r, size, err
						}
						for eofi := range len(eofrunes) {
							if eofrunes[eofi][0] == r {
								mtcheofs[string(eofrunes[eofi])] = eofi
							}
						}
						prvr = r
						if len(mtcheofs) == 0 {
							return r, size, err
						}
						if len(mtcheofs) == 1 {
							for eofk, eofi := range mtcheofs {
								if len(eofk) == 1 {
									delete(mtcheofs, eofk)
									prvr = 0
									strsrch = false
									lsteofphrse = string(eofrunes[eofi])
									if fnderr := foundmatch(lsteofphrse, rnsrdr, orgslcrdr, err, func() *RuneReaderSlice {
										if flshrdr != nil {
											return flshrdr
										}
										flshrdr = NewRuneReaderSlice()
										return flshrdr
									}()); fnderr != nil {
										return 0, 0, fnderr
									}
									return rnsrdr.ReadRune()
								}
								break
							}
						}
						tstphrse = string(r)
						lsteofphrse = ""
						for eofk := range mtcheofs {
							if eofk[:1] != tstphrse {
								delete(mtcheofs, eofk)
							}
							if len(mtcheofs) == 0 {
								return r, size, err
							}
						}
						goto rdfndeofs
					}
					if err != io.EOF {
						return
					}
					continue
				}
				if err != nil {
					if err.Error() == lsteofphrse {
						if len(bfrdrns) > 0 {

						}
						return
					}
				}
				if noorg = err != nil; noorg {
					if err == io.EOF {
						return rnsrdr.ReadRune()
					}
					return
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

func valToRuneReader(val interface{}, clear bool) io.RuneReader {
	if s, _ := val.(string); s != "" {
		return strings.NewReader(s)
	}
	if int32s, _ := val.([]int32); len(int32s) > 0 {
		rns := make([]rune, len(int32s))
		copy(rns, int32s)
		return NewRunesReader(rns...)
	}
	if bf, _ := val.(*Buffer); bf != nil {
		return bf.Clone(clear).Reader(true)
	}
	return nil
}

func init() {
	tstslcrnsrdr := NewRuneReaderSlice()
	tstrdr := ReadRunesUntil(strings.NewReader(
		`<:_:rootpath:/>jhjkhj hkj <:_:rootpath:/> hjh jh jh jk hjk h j h`), RunesUntilSliceFlushFunc(func(phrasefnd string, untilrdr io.RuneReader, orgrd *RuneReaderSlice, orgerr error, flushrdr *RuneReaderSlice) (fnderr error) {
		if phrasefnd == "<:_:" {
			bf := NewBuffer()
			if _, fnderr = bf.ReadRunesFrom(untilrdr); fnderr != nil {
				if fnderr.Error() == ":/>" {
					fnderr = nil
					if eql, _ := bf.Equals("rootpath"); eql {
						tstslcrnsrdr.PreAppend(strings.NewReader("TEST/PATH"))
						return io.EOF
					}
				}
				if fnderr == io.EOF {
					if !bf.Empty() {
						flushrdr.PreAppend(bf.Reader(true))
					}
				}
			}
			return
		}

		return fmt.Errorf("%s", phrasefnd)
	}), "<:_:", ":/>", "<", ">")

	if s, _ := ReaderToString(ReadRuneFunc(func() (r rune, size int, err error) {
	hsrns:
		if !tstslcrnsrdr.Empty() {
			r, size, err = tstslcrnsrdr.ReadRune()
			if size > 0 {
				return
			}
			if err != nil && err != io.EOF {
				return
			}
		}
		r, size, err = tstrdr.ReadRune()
		if size > 0 {
			if err == io.EOF {
				if tstslcrnsrdr.Length() > 0 {
					err = nil
				}
			}
			return r, size, err
		}
		if size == 0 && err == io.EOF && tstslcrnsrdr.Length() > 0 {
			goto hsrns
		}
		if size == 0 && err == nil {
			err = io.EOF
		}
		return
	})); s != "" {
		if s == `TEST/PATHjhjkhj hkj TEST/PATH hjh jh jh jk hjk h j h` {
			//fmt.Println(s)
		}
	}
}
