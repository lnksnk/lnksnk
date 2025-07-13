package listening

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type LISTENER interface {
	SwapHandler(hndlr http.Handler)
	Startup() error
	Shutdown() error
}

type listener struct {
	http.Handler
	ln       net.Listener
	orghndlr http.Handler
}

// SwapHandler implements LISTENER.
func (lstnr *listener) SwapHandler(hndlr http.Handler) {
	if lstnr == nil || hndlr == nil {
		return
	}
	//if orghndlr := lstnr.orghndlr; orghndlr != hndlr {
	lstnr.orghndlr = hndlr
	//}
}

func nextlistener(network string, addr string, handler http.Handler, tlsconf ...*tls.Config) (lstnr LISTENER) {
dotcp:
	if strings.Contains(network, "tcp") {

		if handler != nil {
			if ln, err := net.Listen(network, addr); err == nil { //net.Listen(network, addr); err == nil {

				if tlsconfL := len(tlsconf); tlsconfL > 0 && tlsconf[0] != nil {
					ln = tls.NewListener(ln, tlsconf[0].Clone())
				}
				lstnr = &listener{orghndlr: handler, ln: ln}
				lstnr.Startup()
				return
			}
		}
		return
	}
	if strings.Contains(network, "quic") {
		if tlsconfL := len(tlsconf); tlsconfL > 0 && tlsconf[0] != nil {

			//certs, _ := tls.X509KeyPair(testcert, testkey)
			dflttls := tlsconf[0]
			server := http3.Server{
				Addr:       addr,
				TLSConfig:  http3.ConfigureTLSConfig(dflttls.Clone()), // use your tls.Config here
				QUICConfig: &quic.Config{Versions: []quic.Version{quic.Version1, quic.Version2}},
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handler.ServeHTTP(w, r)
				}),
			}

			h1and2 := http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					server.SetQUICHeaders(w.Header())
					handler.ServeHTTP(w, r)
				})}

			go server.ListenAndServe()

			go h1and2.Serve(func() (ln net.Listener) {
				ln, _ = net.Listen("tcp", addr)
				return tls.NewListener(ln, dflttls.Clone())
			}())
			return
		}
		network = "tcp"
		goto dotcp
	}
	return
}

func (lstnr *listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if lstnr == nil {
		return
	}
	if orghndlr := lstnr.orghndlr; orghndlr != nil {
		orghndlr.ServeHTTP(w, r)
	}
}

func (lstnr *listener) Startup() (err error) {
	if lstnr == nil {
		return
	}
	if ln := lstnr.ln; ln != nil {
		go http.Serve(ln, h2c.NewHandler(lstnr, &http2.Server{}))
	}
	return
}

func (lstnr *listener) Shutdown() (err error) {
	if lstnr == nil {
		return
	}
	if ln := lstnr.ln; ln != nil {
		lstnr.ln = nil
		ln.Close()
		lstnr.orghndlr = nil
	}
	return
}
