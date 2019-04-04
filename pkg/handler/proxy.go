// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Proxy is a HTTP Handler that proxies HTTP requests a target URL.
// TODO(doublek):
//  - Check liveness of backend.
//  - Support multiple backends.
type Proxy struct {
	proxy http.Handler
}

func NewProxy(targetURL *url.URL) *Proxy {
	p := httputil.NewSingleHostReverseProxy(targetURL)
	origDirector := p.Director
	// Rewrite host header. We do just enough and call the Director
	// defined in the standard library to do the rest of the request
	// fiddling.
	p.Director = func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		origDirector(req)
		req.Host = targetURL.Host
	}
	return &Proxy{
		proxy: p,
	}
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.proxy.ServeHTTP(rw, req)
}
