package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type ForwardHandler struct {
	Targets map[string]*url.URL
}

func (hd ForwardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("x-target")
	url, ok := hd.Targets[target]
	if !ok {
		http.Error(w, fmt.Sprintf("Configuration missing for target %#v", target), 400)
		return
	}
	serveReverseProxy(url, w, r)
}

// Serve a reverse proxy for a given url
func serveReverseProxy(url *url.URL, res http.ResponseWriter, req *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}
