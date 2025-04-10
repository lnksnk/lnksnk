package serveio

import (
	"bufio"
	"io"
	"net/http"

	"github.com/lnksnk/lnksnk/iorw"
)

type flusher interface {
	Flush()
}

type flushererror interface {
	Flush() error
}

type statusCodewriter interface {
	Write([]byte) (int, error)
	WriteHeader(int)
}

type header interface {
	map[string][]string
	Add(key string, value string)
	Clone() http.Header
	Del(key string)
	Get(key string) string
	Set(key string, value string)
	Values(key string) []string
	Write(w io.Writer) error
	WriteSubset(w io.Writer, exclude map[string]bool) error
}

type headerwriter interface {
	Header() http.Header
}

type Writer interface {
	io.WriteCloser
	WriteHeader(int)
	Flush() error
	Header() http.Header
	Print(...interface{}) error
	BPrint(...interface{}) error
	Println(...interface{}) error
	ReadFrom(r io.Reader) (n int64, err error)
	MaxWriteSize(int64) bool
}

type writer struct {
	stscdewtr statusCodewriter
	flshr     flusher
	flshrerr  flushererror
	orgwrtr   io.Writer
	hdrwtr    headerwriter
	header    http.Header
	buff      *bufio.Writer
	Status    int
	MaxSize   int64
	FlushSize int
}

func NewWriter(orgwrtr io.Writer) (rqw *writer) {
	stscdewtr, _ := orgwrtr.(statusCodewriter)
	hdrwtr, _ := orgwrtr.(headerwriter)
	flshr, _ := orgwrtr.(flusher)
	flshrerr, _ := orgwrtr.(flushererror)
	rqw = &writer{orgwrtr: orgwrtr, stscdewtr: stscdewtr, hdrwtr: hdrwtr, flshr: flshr, flshrerr: flshrerr, Status: 200, MaxSize: -1, FlushSize: 32768 * 2}
	if hdrwtr != nil {
		rqw.header = hdrwtr.Header()
	} else {
		rqw.header = http.Header{}
	}
	return
}

func (rqw *writer) MaxWriteSize(maxsize int64) bool {
	if rqw == nil {
		return false
	}
	if rqw.MaxSize == -1 {
		rqw.MaxSize = maxsize
		return true
	}
	return false
}

func (rqw *writer) ReadFrom(r io.Reader) (n int64, err error) {
	if rqw != nil {
		if orgwrtr := rqw.orgwrtr; orgwrtr != nil {
			if rqw.buff != nil {
				if err = rqw.Flush(); err != nil {
					return
				}
			}
			n, err = iorw.ReadToFunc(orgwrtr, r.Read)
		}
	}
	return
}

func (rqw *writer) Header() http.Header {
	if rqw != nil {
		if header := rqw.header; header != nil {
			return header
		}
	}
	return nil
}

func (rqw *writer) WriteHeader(status int) {
	if rqw != nil {
		if stscdewtr := rqw.stscdewtr; stscdewtr != nil {
			if status == 0 {
				status = rqw.Status
			}
			stscdewtr.WriteHeader(status)
		}
	}
}

func (rqw *writer) Flush() (err error) {
	if rqw != nil {
		if buff, flshr, flshrerr := rqw.buff, rqw.flshr, rqw.flshrerr; buff != nil {
			if err = buff.Flush(); err == nil {
				if flshr != nil {
					flshr.Flush()
					return
				}
				if flshrerr != nil {
					err = flshrerr.Flush()
				}
			}
		}
	}
	return
}

func (rqw *writer) buffer() *bufio.Writer {
	if rqw != nil {
		buff := rqw.buff
		if buff == nil {
			if orgwtr := rqw.orgwrtr; orgwtr != nil {
				if rqw.FlushSize < 32768*2 {
					rqw.FlushSize = 32768 * 2
				}
				rqw.buff = bufio.NewWriterSize(orgwtr, rqw.FlushSize)
				buff = rqw.buff
			}
			return buff
		}
		if rqw.FlushSize < 32768*2 {
			rqw.FlushSize = 32768 * 2
		}
		if buff.Size() != rqw.FlushSize {
			buff.Flush()
			if orgwtr := rqw.orgwrtr; orgwtr != nil {
				rqw.buff = bufio.NewWriterSize(orgwtr, rqw.FlushSize)
			} else {
				rqw.buff = nil
			}
			buff = rqw.buff
			return buff
		}
		return buff

	}
	return nil
}

func (rqw *writer) Close() (err error) {
	if rqw != nil {
		if buff := rqw.buff; buff != nil {
			rqw.Flush()
			rqw.buff = nil
		}
		rqw.orgwrtr = nil
		rqw.stscdewtr = nil
		rqw.hdrwtr = nil
		rqw.header = nil
		rqw.flshr = nil
		rqw.flshrerr = nil
	}
	return
}

func (rqw *writer) Write(p []byte) (n int, err error) {
	if pl := len(p); rqw != nil && pl > 0 {
		if buf := rqw.buffer(); buf != nil {
			maxsize := rqw.MaxSize
			if maxsize > 0 {
				if int64(pl) >= maxsize {
					pl = int(maxsize)
				}
				if n, err = rqw.buffer().Write(p[:pl]); n > 0 {
					rqw.MaxSize -= int64(n)
				}
				return
			}
			if maxsize == -1 {
				n, err = rqw.buffer().Write(p)
				return
			}
		}
	}
	return 0, io.EOF
}

func (rqw *writer) Print(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fprint(rqw, a...); err == nil {
			err = rqw.Flush()
		}
	}
	return
}

func (rqw *writer) BPrint(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fbprint(rqw, a...); err == nil {
			err = rqw.Flush()
		}
	}
	return
}

func (rqw *writer) Println(a ...interface{}) (err error) {
	if rqw != nil {
		if err = iorw.Fprintln(rqw, a...); err != nil {
			err = rqw.Flush()
		}
	}
	return
}
