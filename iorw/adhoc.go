package iorw

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Printer - interface
type Printer interface {
	Print(a ...interface{}) error
	Println(a ...interface{}) error
	Write(p []byte) (int, error)
}

// Reader - interface
type Reader interface {
	Seek(int64, int) (int64, error)
	SetMaxRead(int64) (err error)
	Read([]byte) (int, error)
	ReadRune() (rune, int, error)
	Readln() (string, error)
	ReadLines() ([]string, error)
	ReadAll() (string, error)
}

// PrinterReader - interface
type PrinterReader interface {
	Printer
	Reader
}

// Fprint - refer to fmt.Fprint
func Fprint(w io.Writer, a ...interface{}) (err error) {
	if len(a) > 0 && w != nil {
		for dn := range a {
			if s, sok := a[dn].(string); sok {
				if _, err = w.Write(RunesToUTF8([]rune(s)...)); err != nil {
					if err == io.EOF {
						err = nil
						continue
					}
					break
				}
				continue
			}
			if ir, irok := a[dn].(io.RuneReader); irok {
				if wbf, _ := w.(*Buffer); wbf != nil {
					if _, err = wbf.ReadRunesFrom(ir); err != nil {
						if err != io.EOF {
							break
						}
						err = nil
					}
					continue
				}
				for err == nil {
					pr, prs, prserr := ir.ReadRune()
					if prs > 0 && (prserr == nil || prserr == io.EOF) {
						_, prserr = w.Write(RunesToUTF8(pr))
					}
					if prserr != nil {
						if prserr != io.EOF {
							err = prserr
						}
						break
					}
				}
				if err != nil {
					break
				}
				continue
			}
			if ir, irok := a[dn].(io.Reader); irok {
				if irnr, irnrok := ir.(io.RuneReader); irnrok {
					for err == nil {
						pr, prs, prserr := irnr.ReadRune()
						if prs > 0 && (prserr == nil || prserr == io.EOF) {
							_, err = w.Write(RunesToUTF8(pr))
						}
						if prserr != nil && err == nil {
							if prserr != io.EOF {
								err = prserr
							}
							break
						}
					}
					if err != nil {
						break
					}
					continue
				}
				if wfrom, _ := w.(io.ReaderFrom); wfrom != nil {
					if _, err = wfrom.ReadFrom(ir); err != nil {
						if err == io.EOF {
							err = nil
							continue
						}
						break
					}
					continue
				}
				if wto, _ := ir.(io.WriterTo); wto != nil {
					if _, err = wto.WriteTo(w); err != nil {
						if err == io.EOF {
							err = nil
							continue
						}
						break
					}
					continue
				}
				if _, err = WriteToFunc(ir, func(b []byte) (int, error) {
					return w.Write(b)
				}); err != nil {
					if err == io.EOF {
						err = nil
						continue
					}
					break
				}
				continue
			}
			if bf, irok := a[dn].(*Buffer); irok {
				_, err = bf.WriteTo(w)
				continue
			}
			if aa, aaok := a[dn].([]interface{}); aaok {
				if len(aa) > 0 {
					if err = Fprint(w, aa...); err != nil {
						break
					}
				}
				continue
			}
			if arn, arnok := a[dn].([]int32); arnok {
				if len(arn) > 0 {
					if err = Fprint(w, string(arn)); err != nil {
						break
					}
				}
				continue
			}
			if sa, saok := a[dn].([]string); saok {
				if len(sa) > 0 {
					if _, err = w.Write(RunesToUTF8([]rune(strings.Join(sa, ""))...)); err != nil {
						break
					}
				}
				continue
			}
			if a[dn] != nil {
				if _, err = fmt.Fprint(w, a[dn]); err != nil {
					break
				}
				continue
			}
		}
	}
	return
}

// Fbprint - refer to fmt.Fprint
func Fbprint(w io.Writer, a ...interface{}) (err error) {
	if len(a) > 0 && w != nil {
		for dn := range a {
			if s, sok := a[dn].(string); sok {
				if _, err = w.Write(RunesToUTF8([]rune(s)...)); err != nil {
					if err == io.EOF {
						err = nil
						continue
					}
					break
				}
				continue
			}
			if irdr, irok := a[dn].(io.RuneReader); irok {
				if ir, irok := a[dn].(io.Reader); irok {
					a[dn] = ir
				} else {
					prns := make([]rune, 4096)
					for err == nil {
						pn, prerr := ReadRunes(prns, irdr)
						if pn > 0 {
							if _, err = w.Write(RunesToUTF8(prns[:pn]...)); err != nil {
								continue
							}
						}
						if prerr != nil {
							err = prerr
							break
						}
					}
					if err == io.EOF {
						err = nil
					}
				}
			}
			if ir, irok := a[dn].(io.Reader); irok {
				if wfrom, _ := w.(io.ReaderFrom); wfrom != nil {
					if _, err = wfrom.ReadFrom(ir); err != nil {
						if err == io.EOF {
							err = nil
							continue
						}
						break
					}
					continue
				}
				if wto, _ := ir.(io.WriterTo); wto != nil {
					if _, err = wto.WriteTo(w); err != nil {
						if err == io.EOF {
							err = nil
							continue
						}
						break
					}
					continue
				}
				if _, err = WriteToFunc(ir, func(b []byte) (int, error) {
					return w.Write(b)
				}); err != nil {
					if err == io.EOF {
						err = nil
						continue
					}
					break
				}
				continue
			}
			if bf, irok := a[dn].(*Buffer); irok {
				_, err = bf.WriteTo(w)
				continue
			}
			if aa, aaok := a[dn].([]interface{}); aaok {
				if len(aa) > 0 {
					if err = Fprint(w, aa...); err != nil {
						break
					}
				}
				continue
			}
			if arn, arnok := a[dn].([]int32); arnok {
				if len(arn) > 0 {
					if err = Fprint(w, string(arn)); err != nil {
						break
					}
				}
				continue
			}
			if sa, saok := a[dn].([]string); saok {
				if len(sa) > 0 {
					if _, err = w.Write(RunesToUTF8([]rune(strings.Join(sa, ""))...)); err != nil {
						break
					}
				}
				continue
			}
			if a[dn] != nil {
				if _, err = fmt.Fprint(w, a[dn]); err != nil {
					break
				}
				continue
			}
		}
	}
	return
}

// IsSpace reports if the rune is a space
// douse an ascci test first then an unicode code if not
func IsSpace(r rune) bool {
	return (asciiSpace[r] == 1) || (r > 128 && unicode.IsSpace(r))
}

var asciiSpace = map[rune]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// IsTxtPar reports is either a ' or " or `
func IsTxtPar(r rune) bool {
	return (txtpars[r] == 1)
}

var txtpars = map[rune]uint8{'\'': 1, '"': 1, '`': 1}

func copyBytes(dest []byte, desti int, src []byte, srci int) (lencopied int, destn int, srcn int) {
	destl, srcl := len(dest), len(src)
	if (destl > 0 && desti < destl) && (srcl > 0 && srci < srcl) {
		if (srcl - srci) <= (destl - desti) {
			cpyl := copy(dest[desti:desti+(srcl-srci)], src[srci:srci+(srcl-srci)])
			srcn = srci + cpyl
			destn = desti + cpyl
			lencopied = cpyl
			return
		}
		if (destl - desti) < (srcl - srci) {
			cpyl := copy(dest[desti:desti+(destl-desti)], src[srci:srci+(destl-desti)])
			srcn = srci + cpyl
			destn = desti + cpyl
			lencopied = cpyl
		}
	}
	return
}

// EOFReadRunes helper function takes a io.RuneReader and read each rune until io.EOF or an error occure
// readrune func(r rune, size int) error) gets called every time a rune is read
func EOFReadRunes(rdr io.RuneReader, readrune func(r rune, size int) error) (err error) {
	if rdr == nil {
		return io.EOF
	}
	r, size, rderr := rune(0), 0, error(nil)
	for err == nil {
		r, size, err = rdr.ReadRune()
		if size > 0 && (err == io.EOF || err == nil) {
			if rderr = readrune(r, size); rderr != nil {
				return rderr
			}
			if err == io.EOF {
				return
			}
			continue
		}
		break
	}
	if size == 0 {
		if err == nil {
			err = io.EOF
		}
	}
	return
}

// Fprintln - refer to fmt.Fprintln
func Fprintln(w io.Writer, a ...interface{}) (err error) {
	if len(a) > 0 && w != nil {
		err = Fprint(w, a...)
	}
	if err == nil {
		err = Fprint(w, "\r\n")
	}
	return
}

// ReadLines from r io.Reader as lines []string
func ReadLines(r interface{}) (lines []string, err error) {
	if r != nil {
		var rnrd io.RuneReader = nil
		if rnr, rnrok := r.(io.RuneReader); rnrok {
			rnrd = rnr
		} else {
			if rd, _ := r.(io.Reader); rd != nil {
				rnrd = bufio.NewReader(rd)
			}
		}
		if rnrd == nil {
			return
		}
		rns := make([]rune, 1024)
		rnsi := 0
		s := ""
		for {
			rn, size, rnerr := rnrd.ReadRune()
			if size > 0 {
				if rn == '\n' {
					if rnsi > 0 {
						s += string(rns[:rnsi])
						rnsi = 0
					}
					if s != "" {
						s = strings.TrimSpace(s)
						if lines == nil {
							lines = []string{}
						}
						lines = append(lines, s)
						s = ""
					}
					continue
				}
				rns[rnsi] = rn
				rnsi++
				if rnsi == len(rns) {
					s += string(rns[:rnsi])
					rnsi = 0
				}
			}
			if rnerr != nil {
				err = rnerr
				if rnsi > 0 {
					s += string(rns[:rnsi])
					rnsi = 0
				}
				if s != "" {
					s = strings.TrimSpace(s)
					if lines == nil {
						lines = []string{}
					}
					lines = append(lines, s)
					s = ""
				}
				if err == io.EOF {
					err = nil
				}
				break
			}
		}
	}
	return
}

// ReadWriteEof takes readfunc and writefunc and keaps on reading and then writing until io.EOF or error occurs
func ReadWriteEof(readfunc func([]byte) (int, error), writefunc func([]byte) (int, error), nexteof func() []byte, foundeeof func() error) (n int64, err error) {
	for err == nil {
		oefbytes := nexteof()
		if eofl := len(oefbytes); eofl > 0 {
			eofl := len(oefbytes)
			eofi := 0
			wn := 0
			prveofb := byte(0)
			rdrbytes := make([]byte, 8192)
			rdrn, rdrerr := readfunc(rdrbytes)
			if rdrbts := rdrbytes[:rdrn]; len(rdrbts) > 0 {
				for bn, bb := range rdrbts {
					if eofi > 0 && oefbytes[eofi-1] == prveofb && oefbytes[eofi] != bb {
						ei := eofi
						eofi = 0
						prveofb = 0
						if wn, err = writefunc(oefbytes[:ei]); err != nil {
							return
						}
						n += int64(wn)
					}
					if oefbytes[eofi] == bb {
						eofi++
						if eofi == eofl {
							eofi = 0
							eofl = 0
							prveofb = 0
							if err = foundeeof(); err != nil {
								return
							}
							oefbytes = nexteof()
							if eofl = len(oefbytes); eofl == 0 {
								return
							}
						} else {
							prveofb = bb
						}
					} else {
						if eofi > 0 {
							ei := eofi
							eofi = 0
							prveofb = 0
							if wn, err = writefunc(oefbytes[:ei]); err != nil {
								return
							}
							n += int64(wn)
						}
						prveofb = bb
						if wn, err = writefunc(rdrbts[bn : bn+1]); err != nil {
							return
						}
						n += int64(wn)
					}
				}
			}
			if rdrerr == io.EOF {

				rdrerr = nil
			}
		} else {
			break
		}
	}
	return
}

// ReadLine from r io.Reader as s string
func ReadLine(rs ...interface{}) (s string, err error) {
	if rsl := len(rs); rsl >= 1 {
		var r interface{} = rs[0]
		var cantrim = false
		if rsl > 1 {
			cantrim, _ = rs[1].(bool)
		}
		if r != nil {
			var rnrd io.RuneReader = nil
			if rnr, rnrok := r.(io.RuneReader); rnrok {
				rnrd = rnr
			} else if rr, rrok := r.(io.Reader); rrok {
				rnrd = bufio.NewReader(rr)
			}
			rns := make([]rune, 1024)
			rnsi := 0
			for {
				rn, size, rnerr := rnrd.ReadRune()
				if size > 0 {
					if rn == '\n' {
						if rnsi > 0 {
							s += strings.TrimFunc(string(rns[:rnsi]), IsSpace)
							rnsi = 0
						}
						break
					}
					rns[rnsi] = rn
					rnsi++
					if rnsi == len(rns) {
						s += string(rns[:rnsi])
						rnsi = 0
					}
				}
				if rnerr != nil {
					err = rnerr
					if rnsi > 0 {
						if err == io.EOF {
							err = nil
						}
						s += string(rns[:rnsi])
						rnsi = 0
					}
					break
				}
			}
		}
		if cantrim {
			s = strings.TrimFunc(s, IsSpace)
		}
		//
	}
	return
}

// ReaderToString read reader and return content as string
func ReaderToString(r interface{}) (s string, err error) {
	runes := make([]rune, 1024)
	runesi := 0
	if err = ReadRunesEOFFunc(r, func(rn rune) error {
		runes[runesi] = rn
		runesi++
		if runesi == len(runes) {
			s += string(runes[:runesi])
			runesi = 0
		}
		return nil
	}); err == nil || err == io.EOF {
		if runesi > 0 {
			s += string(runes[:runesi])
			runesi = 0
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// ReadRunesEOFFunc read runes from r io.Reader, r io.RuneReader or r func() (rune,int,error) and call fncrne func(rune) error
func ReadRunesEOFFunc(r interface{}, fncrne func(rune) error) (err error) {
	if r != nil && fncrne != nil {
		var rdrne func() (rune, int, error)
		if rnr, rnrok := r.(io.RuneReader); rnrok {
			//rnrd = rnr
			rdrne = rnr.ReadRune
		} else if rdr, rdrok := r.(io.Reader); rdrok {
			rdrne = bufio.NewReader(rdr).ReadRune
		}
		if rdrne != nil {
			for {
				rn, size, rnerr := rdrne()
				if size > 0 {
					if err = fncrne(rn); err != nil {
						break
					}
				}
				if err == nil && rnerr != nil {
					if rnerr != io.EOF {
						err = rnerr
					}
					break
				}
			}
		}
	}
	return
}

// RunesToUTF8 convert rs []rune to []byte of raw utf8
func RunesToUTF8(rs ...rune) []byte {
	size := 0
	for rn := range rs {
		size += utf8.RuneLen(rs[rn])
	}
	bs := make([]byte, size)
	count := 0
	for rn := range rs {
		count += utf8.EncodeRune(bs[count:], rs[rn])
	}

	return bs
}

type funcrdrwtr struct {
	funcw func([]byte) (int, error)
	funcr func([]byte) (int, error)
}

func (fncrw *funcrdrwtr) Close() (err error) {
	if fncrw != nil {
		if fncrw.funcr != nil {
			fncrw.funcr = nil
		}
		if fncrw.funcw != nil {
			fncrw.funcw = nil
		}
		fncrw = nil
	}
	return
}

func (fncrw *funcrdrwtr) Write(p []byte) (n int, err error) {
	if fncrw != nil && fncrw.funcw != nil {
		n, err = fncrw.funcw(p)
	}
	return
}

func (fncrw *funcrdrwtr) Read(p []byte) (n int, err error) {
	if fncrw != nil && fncrw.funcr != nil {
		n, err = fncrw.funcr(p)
	}
	return
}

// WriteToFunc takes a io.Reader and func(p[]byte) (n int,err error) arguments write p []byte to func argument until an error or io.EOF
func WriteToFunc(r io.Reader, funcw func([]byte) (int, error), bufsize ...int) (n int64, err error) {
	if r != nil && funcw != nil {
		func() {
			n, err = ReadWriteToFunc(funcw, func(b []byte) (int, error) {
				return r.Read(b)
			}, bufsize...)
		}()
	}
	return
}

// ReadToFunc takes a io.Writer and func(p[]byte) (n int,err error) as arguments and keep on reading p[]byte from func argument and write it to io.Write argument until error or io.EOF
func ReadToFunc(w io.Writer, funcr func([]byte) (int, error)) (n int64, err error) {
	if w != nil && funcr != nil {
		func() {
			n, err = ReadWriteToFunc(func(b []byte) (int, error) {
				return w.Write(b)
			}, funcr)
		}()
	}
	return
}

// ReadWriteToFunc has funcw func([]byte) (int, error), funcr func([]byte) (int, error) arguments
// continuesly read p[]byte from funcr and write it to funcw untile io.EOF or an error
func ReadWriteToFunc(funcw func([]byte) (int, error), funcr func([]byte) (int, error), bufsize ...int) (n int64, err error) {
	if funcw != nil && funcr != nil {
		fncrw := &funcrdrwtr{funcr: funcr, funcw: funcw}
		func() {
			defer func() {
				if rv := recover(); rv != nil {
					switch x := rv.(type) {
					case string:
						err = errors.New(x)
					case error:
						err = x
					default:
						err = errors.New("unknown panic")
					}
				}
				fncrw.Close()
			}()
			if len(bufsize) > 0 {
				if bufsize[0] < 8912 {
					n, err = io.Copy(fncrw, fncrw)
				} else {
					n, err = io.CopyBuffer(fncrw, fncrw, make([]byte, bufsize[0]))
				}
			} else {
				n, err = io.Copy(fncrw, fncrw)
			}
		}()
	}
	return
}

// RunesToBytes takes rns[]rune argument and return []byte and bytes length
func RunesToBytes(r ...rune) (bts []byte, rl int) {
	return RunesToUTF8(r...), len(r)
}

func ToData(format string, a ...interface{}) (data interface{}, err error) {
	mltiargrdr := NewMultiArgsReader(a...)
	defer mltiargrdr.Close()
	if format = strings.TrimFunc(format, IsSpace); format == "" || strings.EqualFold(format, "json") {
		data, err = Marshal(mltiargrdr)
	} else if strings.EqualFold(format, "raw") {
		data, err = mltiargrdr.ReadAll()
	}
	return
}

// RunesHasPrefix report if runes []rune argument has subrunes []rune as prefix rune(s)
func RunesHasPrefix(runes []rune, subrunes ...rune) bool {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		for rni := range lnsubrns {
			if runes[rni] != subrunes[rni] {
				return false
			}
		}
		return true
	}
	return false
}

// RunesHasSuffix report if runes []rune argument has subrunes []rune as suffix rune(s)
func RunesHasSuffix(runes []rune, subrunes ...rune) bool {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		adjsi := lnrns - lnsubrns
		for rni := range lnsubrns {
			if runes[rni+adjsi] != subrunes[rni] {
				return false
			}
		}
		return true
	}
	return false
}

// IndexOfRunes return -1 or the first index where subrunes []rune argument occur in runes []rune argument
func IndexOfRunes(runes []rune, subrunes ...rune) int {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		srn := 0
		for rn, r := range runes {
			if subrunes[srn] == r {
				srn++
				if srn == lnsubrns {
					return rn - (srn - 1)
				}
				continue
			}
			if srn > 0 && subrunes[srn-1] == r {
				continue
			}
			srn = 0
			continue
		}
	}
	return -1
}

// LastIndexOfRunes return -1 or the last index where subrunes []rune argument occur in runes []rune argument
func LastIndexOfRunes(runes []rune, subrunes ...rune) int {
	if lnrns, lnsubrns := len(runes), len(subrunes); lnrns >= lnsubrns {
		for rn := range lnrns {
			if rn < lnsubrns-1 {
				break
			}
			for srn := range subrunes {
				if sr, r := subrunes[lnsubrns-(srn+1)], runes[lnrns-(rn+1)]; sr != r {
					break
				}
				if srn == lnsubrns-1 {
					return rn
				}
			}
		}
	}
	return -1
}

// ReadRunes reads p[]rune from any argument that implements io.RuneReader
func ReadRunes(p []rune, rds ...interface{}) (n int, err error) {
	if pl := len(p); pl > 0 {
		var rd io.RuneReader = nil
		if len(rds) == 1 {
			if rd, _ = rds[0].(io.RuneReader); rd == nil {
				if r, _ := rds[0].(io.Reader); r != nil {
					rd = bufio.NewReader(r)
				}
			}
			if rd != nil {
				pi := 0
				for pi < pl {
					pr, ps, perr := rd.ReadRune()
					if ps > 0 {
						p[pi] = pr
						pi++
					}
					if perr != nil || ps == 0 {
						if perr == nil {
							perr = io.EOF
						}
						err = perr
						break
					}
				}
				if n = pi; n > 0 && err == io.EOF {
					err = nil
				}
			}
		}
	}
	return
}
