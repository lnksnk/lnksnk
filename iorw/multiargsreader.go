package iorw

import (
	"fmt"
	"io"
	"strings"
)

type MultiArgsReader struct {
	args    []interface{}
	crntr   io.Reader
	crntrnr io.RuneReader
	buf     []byte
	bufi    int
	bufl    int
}

func NewMultiArgsReader(a ...interface{}) (mltiargsr *MultiArgsReader) {
	mltiargsr = &MultiArgsReader{}
	mltiargsr.InsertArgs(a...)
	return
}

func (mltiargsr *MultiArgsReader) ArgsSize() (s int) {
	if mltiargsr != nil {
		s = len(mltiargsr.args)
	}
	return
}

func (mltiargsr *MultiArgsReader) CanRead() (canread bool) {
	if mltiargsr != nil {
		canread = len(mltiargsr.args) > 0 || mltiargsr.crntr != nil || mltiargsr.crntrnr != nil
	}
	return
}

func (mltiargsr *MultiArgsReader) nextrdr() (nxtrdr io.Reader, nxtrnrdr io.RuneReader) {
	if mltiargsr != nil {
		for nxtrdr == nil && len(mltiargsr.args) > 0 {
			d := mltiargsr.args[0]
			mltiargsr.args = mltiargsr.args[1:]
			if d != nil {
				if s, _ := d.(string); s != "" {
					nxtrdr = strings.NewReader(s)
				} else if rdr, _ := d.(io.Reader); rdr != nil {
					nxtrdr = rdr
				} else {
					nxtrdr = strings.NewReader(fmt.Sprint(d))
				}
			} else {
				continue
			}
		}
		if nxtrdr != nil {
			if nxtrnrdr, _ = nxtrdr.(io.RuneReader); nxtrnrdr == nil {
				nxtrnrdr = NewEOFCloseSeekReader(nxtrdr, false)
			}
		}
	}
	return
}

func multiArgsRead(mltiargsr *MultiArgsReader, p []byte) (n int, err error) {
	if pl := len(p); pl > 0 {
		for n < pl && err == nil {
			if mltiargsr != nil {
				if mltiargsr.crntr != nil {
					var crntr = mltiargsr.crntr
					if crntrrnr, _ := crntr.(io.RuneReader); crntrrnr != nil && crntrrnr == mltiargsr.crntrnr {
						crntr, _ = crntrrnr.(io.Reader)
					}
					crntn, cnrterr := crntr.Read(p[n : n+(pl-n)])
					n += crntn
					if cnrterr != nil {
						if cnrterr == io.EOF {
							if mltiargsr.crntr, mltiargsr.crntrnr = mltiargsr.nextrdr(); mltiargsr.crntr == nil {
								break
							}
						} else {
							mltiargsr.crntr = nil
							err = cnrterr
						}
					}
				} else if mltiargsr.crntr == nil {
					if mltiargsr.crntr, mltiargsr.crntrnr = mltiargsr.nextrdr(); mltiargsr.crntr == nil {
						break
					}
				}
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
		}
	}
	return
}

func (mltiargsr *MultiArgsReader) Read(p []byte) (n int, err error) {
	if pl := len(p); pl > 0 {
		for n < pl && err == nil {
			if mltiargsr != nil {
				if mltiargsr.bufl == 0 || mltiargsr.bufl > 0 && mltiargsr.bufi == mltiargsr.bufl {
					if len(mltiargsr.buf) != 4096 {
						mltiargsr.buf = nil
						mltiargsr.buf = make([]byte, 4096)
					}
					pn, perr := multiArgsRead(mltiargsr, mltiargsr.buf)
					if pn > 0 {
						mltiargsr.buf = mltiargsr.buf[:pn]
						mltiargsr.bufi = 0
						mltiargsr.bufl = pn
					}
					if perr != nil {
						if perr != io.EOF {
							err = perr
							break
						}
					}
					if pn == 0 {
						break
					}
				}
				_, n, mltiargsr.bufi = CopyBytes(p, n, mltiargsr.buf[:mltiargsr.bufl], mltiargsr.bufi)
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
		}
	}
	return
}

func (mltiargsr *MultiArgsReader) ReadLine() (ln string, err error) {
	ln, err = ReadLine(mltiargsr)
	return
}

func (mltiargsr *MultiArgsReader) ReadLines() (lines []string, err error) {
	lines, err = ReadLines(mltiargsr)
	return
}

func (mltiargsr *MultiArgsReader) ReadRune() (r rune, size int, err error) {
	r, size, err = mutiArgsReadRune(mltiargsr)
	return
}

func (mltiargsr *MultiArgsReader) ReadAll() (all string, err error) {
	all, err = ReaderToString(mltiargsr)
	return
}

func mutiArgsReadRune(mltiargsr *MultiArgsReader) (r rune, size int, err error) {
	if mltiargsr != nil {
		if mltiargsr.crntrnr == nil {
			if mltiargsr.crntr == nil {
				mltiargsr.crntr, mltiargsr.crntrnr = mltiargsr.nextrdr()
			}
			if mltiargsr.crntrnr != nil {
				r, size, err = mltiargsr.crntrnr.ReadRune()
				if err != nil {
					mltiargsr.crntrnr = nil
					mltiargsr.crntr = nil
					if err == io.EOF {
						r, size, err = mutiArgsReadRune(mltiargsr)
					}
				}
			} else {
				err = io.EOF
			}
		} else {
			r, size, err = mltiargsr.crntrnr.ReadRune()
			if err != nil {
				mltiargsr.crntrnr = nil
				mltiargsr.crntr = nil
				if err == io.EOF {
					r, size, err = mutiArgsReadRune(mltiargsr)
				}
			}
		}
	} else {
		err = io.EOF
	}
	return
}

func (mltiargsr *MultiArgsReader) InsertArgs(a ...interface{}) {
	if mltiargsr != nil && len(a) > 0 {
		if mltiargsr.crntr != nil {
			mltiargsr.args = append([]interface{}{mltiargsr.crntr}, mltiargsr.args...)
			mltiargsr.args = append(a, mltiargsr.args...)
		} else {
			mltiargsr.args = append(a, mltiargsr.args...)
		}
		mltiargsr.crntr, mltiargsr.crntrnr = mltiargsr.nextrdr()
	}
}

func (mltiargsr *MultiArgsReader) Close() (err error) {
	if mltiargsr != nil {
		if mltiargsr.crntr != nil {
			mltiargsr.crntr = nil
		}
		if mltiargsr.args != nil {
			if len(mltiargsr.args) > 0 {
				for n, d := range mltiargsr.args {
					mltiargsr.args[n] = nil
					if d != nil {
						d = nil
					}
				}
				mltiargsr.args = nil
			}
		}
		if mltiargsr.crntr != nil {
			mltiargsr.crntr = nil
		}
		if mltiargsr.crntrnr != nil {
			mltiargsr.crntrnr = nil
		}
		if mltiargsr.buf != nil {
			mltiargsr.buf = nil
		}
		mltiargsr = nil
	}
	return
}