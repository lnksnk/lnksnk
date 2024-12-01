package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type proxy struct {
	httpproxy      *httputil.ReverseProxy
	targeturl      *url.URL
	DestinationUrl string
	endpoint       string
}

func newProxy(destinationurl string) (prx *proxy) {
	if desturl, _ := url.Parse(destinationurl); desturl != nil {
		prx = &proxy{httpproxy: httputil.NewSingleHostReverseProxy(desturl), targeturl: desturl, DestinationUrl: destinationurl}
	}
	return
}

func (prx *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if prxy, trgturl := prx.httpproxy, prx.targeturl; prxy != nil {
		r.URL.Host = trgturl.Host
		r.URL.Scheme = trgturl.Host
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Host = trgturl.Host

		// trim reverseProxyRouterPrefix
		path := r.URL.Path
		r.URL.Path = strings.TrimLeft(path, prx.endpoint)
		prxy.ServeHTTP(w, r)
	}
}

func init() {

}
