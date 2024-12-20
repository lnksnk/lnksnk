package serveio

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Reader interface {
	RangeOffset() int64
	RangeType() string
	io.ReadCloser
	io.RuneReader
	Header() http.Header
	Context() context.Context
	Cancel()
	HttpR() *http.Request
	Path() string
	IsMobile() bool
}

type reader struct {
	ctx         context.Context
	ctxcnl      func()
	httpr       *http.Request
	path        string
	bufr        *bufio.Reader
	rangetype   string
	rangeoffset int64
}

func (rqr *reader) RangeOffset() int64 {
	if rqr != nil {
		return rqr.rangeoffset
	}
	return -1
}

func (rqr *reader) HttpR() (httpr *http.Request) {
	if rqr != nil {
		httpr = rqr.httpr
	}
	return
}

var mobileRE, _ = regexp.Compile(`/(android|bb\d+|meego).+mobile|armv7l|avantgo|bada\/|blackberry|blazer|compal|elaine|fennec|hiptop|iemobile|ip(hone|od)|iris|kindle|lge |maemo|midp|mmp|mobile.+firefox|netfront|opera m(ob|in)i|palm( os)?|phone|p(ixi|re)\/|plucker|pocket|psp|redmi|series[46]0|samsungbrowser.*mobile|symbian|treo|up\.(browser|link)|vodafone|wap|windows (ce|phone)|xda|xiino/i`)
var notMobileRE, _ = regexp.Compile(`/CrOS/`)
var tabletRE, _ = regexp.Compile(`/android|ipad|playbook|silk/i`)

func (rqr *reader) IsMobile() (mobile bool) {
	if rqr != nil {
		if hr := rqr.Header(); hr != nil {
			if au := hr.Get("User-Agent"); au != "" {
				if mobile = (mobileRE.MatchString(au) && !notMobileRE.MatchString(au)) || tabletRE.MatchString(au); !mobile {
					mobile = strings.Contains(strings.ToLower(au), "mobile")
				}
			}
		}
	}
	return
}

func (rqr *reader) Headers() (hdrs []string) {
	if rqr != nil {
		if httpr := rqr.httpr; httpr != nil {
			for h := range httpr.Header {
				hdrs = append(hdrs, h)
			}
		}
	}
	return
}

func (rqr *reader) Header() http.Header {
	if rqr != nil {
		if httpr := rqr.httpr; httpr != nil {
			return httpr.Header
		}
	}
	return nil
}

func (rqr *reader) Path() string {
	if rqr != nil {
		if rqr.path == "" && rqr.httpr != nil {
			rqr.path = rqr.httpr.URL.Path
		}
		return rqr.path
	}
	return ""
}

func (rqr *reader) RangeType() string {
	if rqr != nil {
		return rqr.rangetype
	}
	return ""
}

func (rqr *reader) buffer() (bufr *bufio.Reader) {
	if rqr != nil {
		if bufr = rqr.bufr; bufr == nil {
			if httpr := rqr.httpr; httpr != nil {
				if r := httpr.Body; r != nil {
					bufr = bufio.NewReaderSize(r, 65536)
				}
			}
		}
	}
	return
}

func (rqr *reader) Read(p []byte) (n int, err error) {
	if rqr != nil {
		if bufr := rqr.buffer(); bufr != nil {
			n, err = bufr.Read(p)
		}
	}
	return
}

func (rqr *reader) ReadRune() (r rune, size int, err error) {
	if rqr != nil {
		if bufr := rqr.buffer(); bufr != nil {
			r, size, err = bufr.ReadRune()
		}
	}
	return
}

func (rqr *reader) Context() (ctx context.Context) {
	if rqr != nil {
		ctx = rqr.ctx
	}
	return
}

func (rqr *reader) Cancel() {
	if rqr != nil {
		if cncl := rqr.ctxcnl; cncl != nil {
			cncl()
		}
	}
	return
}

func (rqr *reader) Close() (err error) {
	if rqr != nil {
		if rqr.httpr != nil {
			rqr.httpr = nil
		}
		if rqr.bufr != nil {
			rqr.bufr = nil
		}
	}
	return
}

func NewReader(httpr *http.Request) (rdr *reader) {
	rdr = &reader{httpr: httpr, rangeoffset: -1, ctx: httpr.Context()}
	if rdr.ctx != nil {
		rdr.ctx, rdr.ctxcnl = context.WithCancel(rdr.ctx)
	}
	if httpr != nil {
		prtclrangetype := ""
		prtclrangeoffset := int64(-1)
		if prtclrange := httpr.Header.Get("Range"); prtclrange != "" && strings.Index(prtclrange, "=") > 0 {
			if prtclrangetype = prtclrange[:strings.Index(prtclrange, "=")]; prtclrange != "" {
				if prtclrange = prtclrange[strings.Index(prtclrange, "=")+1:]; strings.Index(prtclrange, "-") > 0 {
					prtclrangeoffset, _ = strconv.ParseInt(prtclrange[:strings.Index(prtclrange, "-")], 10, 64)
				}
			}
		}
		rdr.rangeoffset = prtclrangeoffset
		rdr.rangetype = prtclrangetype
	}
	return
}
