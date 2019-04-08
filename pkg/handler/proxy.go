// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package handler

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

const unknownClientIP = "_"

// LoggingTransport wraps an existing http.Transport RoundTripper and then
// logs some request headers along with the response status code.
type LoggingTransport struct {
	*http.Transport
}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	resp, respErr := lt.Transport.RoundTrip(req)
	if respErr != nil {
		log.WithError(respErr).Debugf("Proxy received error response during round trip")
	}

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.WithError(err).Errorf("Could not parse remote addr %v", req.RemoteAddr)
		clientIP = unknownClientIP
	}
	fields := log.Fields{
		"clientIP": clientIP,
		"method":   req.Method,
		"path":     req.URL.Path,
	}
	if respErr == nil {
		fields["responseCode"] = resp.StatusCode
	} else {
		fields["responseError"] = respErr.Error()
	}
	log.WithFields(fields).Info("Access Log")
	return resp, respErr
}

// Proxy is a HTTP Handler that proxies HTTP requests a target URL.
// TODO(doublek):
//  - Check liveness of backend.
//  - Support multiple backends.
type Proxy struct {
	proxy http.Handler
}

func NewProxy(targetURL *url.URL, tlsConfig *tls.Config) *Proxy {
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
	// Extend http.DefaultTransport RoundTripper with custom TLS config.
	p.Transport = &LoggingTransport{
		&http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &Proxy{
		proxy: p,
	}
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.proxy.ServeHTTP(rw, req)
}
