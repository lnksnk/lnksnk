package serveio

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/lnksnk/lnksnk/parameters"
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
	IsTablet() bool
	RemoteAddr() string
	LocalAddr() string
	Method() string
	Protocol() string
	IsSSL() bool
	Proto() string
	Params() parameters.Parameters
}

type reader struct {
	ctx         context.Context
	mthd        string
	prtcl       string
	rmtaddr     string
	lkladdr     string
	ctxcnl      func()
	httpr       *http.Request
	header      http.Header
	isssl       bool
	path        string
	bufr        *bufio.Reader
	orgrdr      io.ReadCloser
	rangetype   string
	rangeoffset int64
	params      parameters.Parameters
}

// Params implements Reader.
func (rqr *reader) Params() parameters.Parameters {
	if rqr == nil {
		return nil
	}
	return rqr.params
}

func (rqr *reader) IsSSL() bool {
	if rqr == nil {
		return false
	}
	return rqr.isssl
}

func (rqr *reader) RemoteAddr() string {
	if rqr == nil {
		return ""
	}
	return rqr.rmtaddr
}

func (rqr *reader) Method() string {
	if rqr == nil {
		return ""
	}
	return rqr.mthd
}

func (rqr *reader) Protocol() string {
	if rqr == nil {
		return ""
	}
	return rqr.prtcl
}

func (rqr *reader) Proto() string {
	if rqr == nil {
		return ""
	}
	proto := rqr.Protocol()
	if pri := strings.Index(proto, "/"); pri > -1 {
		proto = proto[:pri]
	}
	proto = strings.ToLower(proto)
	if rqr.isssl {
		proto += "s"
	}
	return proto
}

func (rqr *reader) LocalAddr() string {
	if rqr == nil {
		return ""
	}
	return rqr.lkladdr
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

func (rqr *reader) IsTablet() (tablet bool) {
	if rqr != nil {
		if hr := rqr.Header(); hr != nil {
			if au := hr.Get("User-Agent"); au != "" {
				if tablet = tabletRE.MatchString(au); !tablet {
					tablet = strings.Contains(strings.ToLower(au), "mobile")
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
			if orgrdr := rqr.orgrdr; orgrdr != nil {
				rqr.bufr = bufio.NewReaderSize(orgrdr, 32768*2)
				bufr = rqr.bufr
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
	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (rqr *reader) ReadRune() (r rune, size int, err error) {
	if rqr != nil {
		if bufr := rqr.buffer(); bufr != nil {
			r, size, err = bufr.ReadRune()
		}
	}
	if size == 0 && err == nil {
		err = io.EOF
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

func NewContextPathReader(ctx context.Context, in interface{}, path string) (rdr *reader) {
	httpr, _ := in.(*http.Request)
	orgqrdr, _ := in.(io.ReadCloser)
	if orgqrdr == nil && httpr != nil {
		orgqrdr = httpr.Body
	}
	if ctx == nil && httpr != nil {
		ctx = httpr.Context()
	}
	rdr = &reader{httpr: httpr, orgrdr: orgqrdr, rangeoffset: -1, ctx: ctx}
	if rdr.ctx != nil {
		if lcaddr, _ := rdr.ctx.Value(http.LocalAddrContextKey).(net.Addr); lcaddr != nil {
			rdr.lkladdr = lcaddr.String()
		} else if httpr != nil {
			if httprctx := httpr.Context(); httprctx != nil {
				if lcaddr, _ := httprctx.Value(http.LocalAddrContextKey).(net.Addr); lcaddr != nil {
					rdr.lkladdr = lcaddr.String()
				}
			}
		}
		rdr.ctx, rdr.ctxcnl = context.WithCancel(rdr.ctx)
	} else if httprctx := httpr.Context(); httprctx != nil {
		if lcaddr, _ := httprctx.Value(http.LocalAddrContextKey).(net.Addr); lcaddr != nil {
			rdr.lkladdr = lcaddr.String()
		}
	}
	if httpr != nil {
		rdr.path = httpr.URL.Path
		rdr.rmtaddr = httpr.RemoteAddr
		rdr.mthd = httpr.Method
		rdr.prtcl = httpr.Proto
		rdr.isssl = httpr.TLS != nil
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
		prms := parameters.NewParameters()
		parameters.LoadParametersFromHTTPRequest(prms, httpr)
		rdr.params = prms
		rdr.header = httpr.Header
	}
	if rdr.path == "" {
		rdr.path = path
	}
	return
}

func NewContextReader(ctx context.Context, in interface{}) (rdr *reader) {
	return NewContextPathReader(ctx, in, "")
}

func NewReader(in interface{}) (rdr *reader) {
	return NewContextPathReader(context.Background(), in, "")
}

func NewPathReader(in interface{}, path string) (rdr *reader) {
	return NewContextPathReader(context.Background(), in, path)
}
