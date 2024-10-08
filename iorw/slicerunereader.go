package iorw

import (
	"bufio"
	"io"
	"strings"
)

type SliceRuneReader interface {
	Length() (ln int)
	Empty() bool
	ReadRune() (r rune, size int, err error)
	PostAppendArgs(...interface{})
	PreAppendArgs(...interface{})
	PreAppend(...io.RuneReader)
	PostAppend(rdrs ...io.RuneReader)
	Close() (err error)
}

type runeReaderSlice struct {
	rnrdrs   []io.RuneReader
	crntrdr  io.RuneReader
	EventEof func(io.RuneReader, error)
}

func NewSliceRuneReader(rnrdrs ...io.RuneReader) (rnrdrsslce SliceRuneReader) {
	rnrdrsslce = &runeReaderSlice{crntrdr: nil, rnrdrs: append([]io.RuneReader{}, rnrdrs...)}
	return
}

func (rnrdrsslce *runeReaderSlice) Length() (ln int) {
	if rnrdrsslce != nil {
		ln = len(rnrdrsslce.rnrdrs)
	}
	return
}

func (rnrdrsslce *runeReaderSlice) Empty() bool {
	if rnrdrsslce == nil {
		return true
	}
	if crntrdr, rnrdrs := rnrdrsslce.crntrdr, rnrdrsslce.rnrdrs; crntrdr == nil && len(rnrdrs) == 0 {
		return true
	}
	return false
}

func (rnrdrsslce *runeReaderSlice) PostAppendArgs(argrdrs ...interface{}) {
	if rnrdrsslce != nil {
		var rdrs []io.RuneReader
		for _, arg := range argrdrs {
			if s, sok := arg.(string); sok {
				if s != "" {
					rdrs = append(rdrs, strings.NewReader(s))
				}
				continue
			}
			if int32s, int32ok := arg.([]int32); int32ok {
				if len(int32s) > 0 {
					rns := make([]rune, len(int32s))
					copy(rns, int32s)
					rdrs = append(rdrs, NewRunesReader(rns...))
				}
				continue
			}
			if int32s, int32ok := arg.(int32); int32ok {
				rns := make([]rune, 1)
				copy(rns, []int32{int32s})
				rdrs = append(rdrs, NewRunesReader(rns...))
				continue
			}
			if rnsrdr, _ := arg.(io.RuneReader); rnsrdr != nil {
				rdrs = append(rdrs, rnsrdr)
				continue
			}
			if rdr, _ := arg.(io.Reader); rdr != nil {
				rdrs = append(rdrs, bufio.NewReaderSize(rdr, 1))
				continue
			}
		}
		if len(rdrs) > 0 {
			rnrdrsslce.PostAppend(rdrs...)
		}
	}
}

func (rnrdrsslce *runeReaderSlice) PreAppendArgs(argrdrs ...interface{}) {
	if rnrdrsslce != nil {
		var rdrs []io.RuneReader
		for _, arg := range argrdrs {
			if s, sok := arg.(string); sok {
				if s != "" {
					rdrs = append(rdrs, strings.NewReader(s))
				}
				continue
			}
			if int32s, int32ok := arg.([]int32); int32ok {
				if len(int32s) > 0 {
					rns := make([]rune, len(int32s))
					copy(rns, int32s)
					rdrs = append(rdrs, NewRunesReader(rns...))
				}
				continue
			}
			if rnsrdr, _ := arg.(io.RuneReader); rnsrdr != nil {
				rdrs = append(rdrs, rnsrdr)
				continue
			}
			if rdr, _ := arg.(io.Reader); rdr != nil {
				rdrs = append(rdrs, bufio.NewReaderSize(rdr, 1))
				continue
			}
		}
		if len(rdrs) > 0 {
			rnrdrsslce.PreAppend(rdrs...)
		}
	}
}

func (rnrdrsslce *runeReaderSlice) PreAppend(rdrs ...io.RuneReader) {
	if rnrdrsslce != nil {
		if len(rdrs) > 0 {
			if rnrdrsslce.crntrdr != nil {
				rdrs = append(rdrs, rnrdrsslce.crntrdr)
				rnrdrsslce.crntrdr = nil
			}
			rnrdrsslce.rnrdrs = append(rdrs, rnrdrsslce.rnrdrs...)
		}
	}
}

func (rnrdrsslce *runeReaderSlice) PostAppend(rdrs ...io.RuneReader) {
	if rnrdrsslce != nil {
		if len(rdrs) > 0 {
			rnrdrsslce.rnrdrs = append(rnrdrsslce.rnrdrs, rdrs...)
		}
	}
}

func readSliceRune(rnrdrsslce *runeReaderSlice, eventeof func(io.RuneReader, error), crntrdr io.RuneReader) (r rune, size int, err error) {
	if rnrdrsslce == nil {
		err = io.EOF
		return
	}

NXTR:
	rdrsl := len(rnrdrsslce.rnrdrs)
	if crntrdr != nil {
		r, size, err = crntrdr.ReadRune()
		if size > 0 {
			return
		}
		rnrdrsslce.crntrdr = nil
		if err == nil || err == io.EOF {
			if rdrsl == 0 {
				if err == nil {
					err = io.EOF
				}
				if eventeof != nil {
					eventeof(crntrdr, err)
				}
				return
			}
			if err == io.EOF {
				err = nil
			}
		}
		if eventeof != nil {
			eventeof(crntrdr, err)
		}
	}
	if rdrsl > 0 {
		crntrdr = rnrdrsslce.rnrdrs[0]
		rnrdrsslce.crntrdr = crntrdr
		rnrdrsslce.rnrdrs = rnrdrsslce.rnrdrs[1:]
		goto NXTR
	}
	err = io.EOF
	return
}

func (rnrdrsslce *runeReaderSlice) ReadRune() (r rune, size int, err error) {
	return readSliceRune(rnrdrsslce, rnrdrsslce.EventEof, rnrdrsslce.crntrdr)
}

func (rnrdrsslce *runeReaderSlice) Close() (err error) {
	if rnrdrsslce != nil {
		if rnrdrsl := len(rnrdrsslce.rnrdrs); rnrdrsl > 0 {
			for rnrdrsl > 0 {
				rnrdrsslce.rnrdrs[0] = nil
				rnrdrsslce.rnrdrs = rnrdrsslce.rnrdrs[1:]
				rnrdrsl--
			}
			rnrdrsslce.rnrdrs = nil
		}
		if rnrdrsslce.crntrdr != nil {
			rnrdrsslce.crntrdr = nil
		}
	}
	return
}
