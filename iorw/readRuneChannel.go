package iorw

import (
	"io"
	"unicode/utf8"
)

type readerrunechan struct {
	chnlrns chan rune
}

func ChannelRuneReader(chnlrns chan rune) (rdrrnschn *readerrunechan) {
	rdrrnschn = &readerrunechan{chnlrns: chnlrns}
	return
}

func (rdrrnschn *readerrunechan) Get() (r rune) {
	if rdrrnschn != nil {
		select {
		case r = <-rdrrnschn.chnlrns:
		default:
		}
	}
	return
}

func (rdrrnschn *readerrunechan) ReadRune() (r rune, size int, err error) {
	if rdrrnschn != nil {
		select {
		case r = <-rdrrnschn.chnlrns:
			size = utf8.RuneLen(r)
		default:
		}
	}
	if size == 0 && err == io.EOF {
		return r, size, io.EOF
	}
	return
}
