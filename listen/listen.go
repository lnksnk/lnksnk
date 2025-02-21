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
dotcp:
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
					go http.Serve(ln, handler)
				}
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

var testcert = []byte(`-----BEGIN CERTIFICATE-----
MIIFdTCCA12gAwIBAgIUVK6VI9draNNd+sYHk66xY4znBQ8wDQYJKoZIhvcNAQEL
BQAwSjELMAkGA1UEBhMCREsxEzARBgNVBAcMCkNvcGVuaGFnZW4xDTALBgNVBAoM
BGttY2QxFzAVBgNVBAMMDmxvY2FsLmttY2QuZGV2MB4XDTI0MDcwNjA2MTQ0OFoX
DTI1MDcwNjA2MTQ0OFowSjELMAkGA1UEBhMCREsxEzARBgNVBAcMCkNvcGVuaGFn
ZW4xDTALBgNVBAoMBGttY2QxFzAVBgNVBAMMDmxvY2FsLmttY2QuZGV2MIICIjAN
BgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA2qFIf/BvUREnk5/Nycf7N+S+zSr2
PyMIRtpcEP/NxZQ5uccC4rNuucS1tUrUfObK1nSvP6hLq7aCoA49Xf1nQrViGk9R
WAlbkWK0FQNCGZx9NfnU3FDpPz56/2xzulLgyJA3uScsHXtjVe4xsVAb+ihnV09S
FF1hFsSP02XbtcJVUNSeAXN0uoHrLjJ+nJrs0yPLhj0KQdudTgzAE1OaobrjNo3d
HoUBl2pQXLs6Fwc8fsSxdSLyOhx9mbzd5D9aNNyESuB1f1DO9aE/rXX26nkgOV2J
NQNaTa2MzgMe8u3e/xbyvzVFke0nyM0gk/BF97rMdq6DsRsrEqQXB/O+RwJSkf6K
ZuXlASXKa/5kn1U34pkH23br/44hzTlkIELd5UpAQG0nDOYfY9vUtn67VXM2VGau
C/ElpsDUl0C1ngurT8AfN/NaKERiFXv7hwBO+XlrmS4fWozflWRuG58DjzzKMQVF
jK7ZxhkXtYJoiiihOcuUlKXQA6lyD2w9U20fYJYa0QWHlXy+eljvkgd+qdTb0OXG
cRBDKEPGyDJ7BJ7xynJWn4lgp95fS613gjdLFPB6yz19EbZktbCmEATcS818/3o7
kDdt9J9ATljUUtD/cZ2lEyLsKsfaCO+i7KxsUG7aXLMRF8OUgWweqgQznbY6jgdY
AIi2ZlzJWoMkT8cCAwEAAaNTMFEwHQYDVR0OBBYEFA7XjKFvHNM+To41/8gRwld0
5sM1MB8GA1UdIwQYMBaAFA7XjKFvHNM+To41/8gRwld05sM1MA8GA1UdEwEB/wQF
MAMBAf8wDQYJKoZIhvcNAQELBQADggIBAE+VmMt+qHXikjnZINTj9OsQpP9O0nH8
gJpQAmeDs84Kwm4arKc5T3ggdsaOjCHBPkBFU4GK/EMq/IIDcag/6Vuw7gYM/F/c
gXq8+fbpbHDX8SOiBkGNejKpudQFSaukqjmOVPdQ4TaXRPFU/OUCv/M4V1Sg0y8C
KSfRgqHxh16R7wtc3TQbRMG0wm10UVlfb0FiJihQEAGgaP56mz5BM5GPiWV0u5NL
X5ms9u/d6Och4jhaTuFcGINhyBMehJ4hX1lOYdjJyptucvBoVlkSizaoCc3h2ynG
5dt0ju2BrgvD86o6JYNUK7DLV1Xd5H2Jg2pRHORvOc99Gl9J/2FBGP7X74cQx+iy
WXAdjA00s8jbyS4SPpfa94Vjlf5c5cVRVnbxa0f1wc2uD/QSR7N0mN8OLCVdtuaH
AbtBl469ea1Q1AQw6FsRal0zIvLbqROC2rxdkkWnkG31ajuI8EY4v+FYE5ksRV7E
QUbhxQfXcmuKTz9fCqsUrVyaNReDbgXMloUMQg5iUzfT+7JZshMMQR3mW2/Xv+kL
XUhMyDXpfNxwbHukl62Gug/fqO6ciJGKcA8xwLUSXGGVQcaLSmMMUhvtjTmV9pvG
UfbzHQ/1XW+jZEXUEduCvTK+9nt3f916gmalCZ6YYM9a5+pifuY7ntg5wdXZwbFB
dmaqLMZGNtW7
-----END CERTIFICATE-----`)

var testkey = []byte(`-----BEGIN PRIVATE KEY-----
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
rfgXoifHXD6F31ofbggVq98pZunNtQ==
-----END PRIVATE KEY-----
`)
