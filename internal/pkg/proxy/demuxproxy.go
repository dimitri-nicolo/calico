// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package proxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"time"

	log "github.com/sirupsen/logrus"
)

// DemuxProxy proxies HTTP based on the provided matcher
type DemuxProxy struct {
	matcher Matcher
	token   string
}

// New returns an initialized Proxy
func New(matcher Matcher, token string) *DemuxProxy {

	if token != "" {
		token = "Bearer " + token
	}

	return &DemuxProxy{
		matcher: matcher,
		token:   token,
	}
}

// ServeHTTP knows how to proxy HTTP requests to different named targets
func (mp *DemuxProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Received request %v", r)

	url, err := mp.matcher.Match(r)

	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	log.Debugf("Will proxy it to %v", url)

	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	if mp.token != "" {
		r.Header.Set("Authorization", mp.token)
	}
	r.Host = url.Host

	log.Debugf("New http request is %v", r)

	reverseProxy := httputil.NewSingleHostReverseProxy(url)
	reverseProxy.FlushInterval = 100 * time.Millisecond
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	reverseProxy.ServeHTTP(w, r)
}
