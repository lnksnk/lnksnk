package listen

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lnksnk/lnksnk/concurrent"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var listerens = concurrent.NewMap()

func ShutdownAll() {
	for key, value := range listerens.Iterate() {
		if lstnr, _ := value.(*http.Server); lstnr != nil {
			lstnr.Shutdown(context.Background())
			fmt.Println("Shutdown - ", key)
		}
	}
	/*listerens.Range(func(key, value any) bool {
		if lstnr, _ := value.(*http.Server); lstnr != nil {
			lstnr.Shutdown(context.Background())
			fmt.Println("Shutdown - ", key)
		}
		return true
	})*/
}

func Shutdown(keys ...interface{}) {
	if len(keys) > 0 {
		keys = append(keys, func(delkeys []interface{}, delvalues []interface{}) {
			for kn, k := range delkeys {
				if lstnr, _ := delvalues[kn].(*http.Server); lstnr != nil {
					lstnr.Shutdown(context.Background())
					fmt.Println("Shutdown - ", k)
				}
			}
		})
		listerens.Del(keys...)
	}
}

func AddrHosts(network, addr string) (host string, err error) {
	rsvldaddr, _ := net.ResolveTCPAddr(func() string {
		if network == "quic" {
			return "tcp"
		}
		return network
	}(), addr)
	host = rsvldaddr.IP.String()
	if host == "" || host == "<nil>" {
		host = "localhost"
	}
	addresses, _ := net.LookupAddr(host)
	for _, hst := range addresses {
		host = hst
	}
	return
}

type listen struct {
	handler   http.Handler
	TLSConfig *tls.Config
}

func (lsnt *listen) Serve(network string, addr string, tlsconf ...*tls.Config) {
	if addr == "" && network != "" {
		addr = network
		network = "tcp"
	}
	if lsnt != nil {
		Serve(network, addr, lsnt.handler, tlsconf...)
	}
}

func (lsnt *listen) ServeTLS(network string, addr string, orgname string, tlsconf ...*tls.Config) {
	if lsnt != nil {
		host, _ := AddrHosts(network, addr)
		certhost := host
		if len(tlsconf) == 0 {
			if tlscnf, _ := GenerateTlsConfig(certhost, orgname); tlscnf != nil {
				tlsconf = append(tlsconf, tlscnf)
			}
		}
		Serve(network, addr, lsnt.handler, tlsconf...)
	}
}

func GenerateTlsConfig(certhost, orgname string) (tslconf *tls.Config, err error) {
	publc, prv, crterr := GenerateTestCertificate(certhost, orgname)
	if crterr != nil {
		return
	}
	cert, certerr := tls.X509KeyPair(publc, prv)
	if certerr != nil {
		return
	}
	tslconf = &tls.Config{InsecureSkipVerify: true}
	tslconf.Certificates = append(tslconf.Certificates, cert)
	if oscph, oscperr := NewOcspHandler(cert); oscperr == nil {
		go oscph.Start()
		tslconf.InsecureSkipVerify = false
		tslconf.GetCertificate = oscph.GetCertificate
	}
	return
}

func (lsnt *listen) Shutdown(keys ...interface{}) {
	if lsnt != nil {
		Shutdown(keys...)
	}
}

func NewListen(handerfunc http.HandlerFunc) *listen {
	if handerfunc == nil && DefaultHandler != nil {
		handerfunc = http.HandlerFunc(DefaultHandler.ServeHTTP)
	}
	return &listen{handler: handerfunc, TLSConfig: &tls.Config{}}
}

var lstnr *listener

func Serve(network string, addr string, handler http.Handler, tlsconf ...*tls.Config) {
	if handler == nil && DefaultHandler != nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			DefaultHandler.ServeHTTP(w, r)
		})
	}
	if strings.Contains(network, "tcp") {

		if handler != nil {
			if ln, err := net.Listen(network, addr); err == nil { //net.Listen(network, addr); err == nil {

				if tlsconfL := len(tlsconf); tlsconfL > 0 && tlsconf[0] != nil {
					ln = tls.NewListener(ln, tlsconf[0].Clone())
				}
				if lstnr == nil {
					lstnr = &listener{lstnsrs: &sync.Map{}, hndlrs: &sync.Map{}}
					//	lstnr.Start()
				}
				hndlv, hndlvok := lstnr.hndlrs.Load(network + addr)
				if !hndlvok {
					handler = h2c.NewHandler(handler, &http2.Server{})
					lstnr.hndlrs.LoadOrStore(network+addr, hndlv)
				} else {
					handler, _ = hndlv.(http.Handler)
				}
				lstnv, lstnvok := lstnr.lstnsrs.Load(network + addr)
				if !lstnvok {
					lstnr.lstnsrs.LoadOrStore(network+addr, ln)
				} else {
					ln, _ = lstnv.(net.Listener)
				}
				if ln != nil && handler != nil {
					go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx, ctxcncl := context.WithCancel(r.Context())
						go func() {
							defer ctxcncl()
							handler.ServeHTTP(w, r)
						}()
						<-ctx.Done()
					}))
				}
				return
			}
		}
	} else if strings.Contains(network, "quic") {
		//if len(tlsconf) > 0 {
		handler = h2c.NewHandler(handler, &http2.Server{})
		adrhost, adrport, _ := net.SplitHostPort(addr)
		htpport, _ := strconv.ParseInt(adrport, 10, 64)

		server := http3.Server{
			Addr:       fmt.Sprintf("%s:%d", adrhost, int(htpport+1)),
			Port:       int(htpport + 1),
			TLSConfig:  http3.ConfigureTLSConfig(&tls.Config{}), // use your tls.Config here
			QUICConfig: &quic.Config{Versions: []quic.Version{quic.Version1, quic.Version2}},
		}
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			server.SetQUICHeaders(w.Header())
			handler.ServeHTTP(w, r)
		})
		h1and2 := http.Server{
			Addr: addr,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				server.SetQUICHeaders(w.Header())
				handler.ServeHTTP(w, r)
			})}

		go server.ListenAndServe()

		go h1and2.ListenAndServe()

		//}
		//http3.ListenAndServeQUIC(addr, "/path/to/cert", "/path/to/key", handler)
	}
}

var DefaultHandler http.Handler = nil

func init() {
	go func() {
		//httpsrv.Serve(lstnr)
	}()
}

// GenerateTestCertificate generates a test certificate and private key based on the given host.
func GenerateTestCertificate(host string, orgname string) ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	if orgname == "" {
		orgname = "LNKSNK"
	}
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{orgname},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		DNSNames:              []string{host},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader, cert, cert, &priv.PublicKey, priv,
	)

	p := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)

	b := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		},
	)

	return b, p, err
}
