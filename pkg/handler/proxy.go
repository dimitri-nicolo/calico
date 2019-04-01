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
	return &Proxy{
		proxy: httputil.NewSingleHostReverseProxy(targetURL),
	}
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.proxy.ServeHTTP(rw, req)
}
