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

// Transport configuration
const (
	defaultMaxIdleConns          = 100
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
)

type ProxyConfig struct {
	TargetURL *url.URL

	TLSConfig *tls.Config

	ConnectTimeout  time.Duration
	KeepAlivePeriod time.Duration
	IdleConnTimeout time.Duration
}

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
	proxy  http.Handler
	config *ProxyConfig
}

func NewProxy(proxyConfig *ProxyConfig) *Proxy {

	p := httputil.NewSingleHostReverseProxy(proxyConfig.TargetURL)
	origDirector := p.Director
	// Rewrite host header. We do just enough and call the Director
	// defined in the standard library to do the rest of the request
	// fiddling.
	p.Director = func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)

		// TODO: handle gzip requests to elasticsearch instead of disabling it.
		req.Header.Del("Accept-Encoding")
		origDirector(req)
		req.Host = proxyConfig.TargetURL.Host
	}

	// Extend http.DefaultTransport RoundTripper with custom TLS config.
	p.Transport = &LoggingTransport{
		&http.Transport{
			TLSClientConfig: proxyConfig.TLSConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   proxyConfig.ConnectTimeout,
				KeepAlive: proxyConfig.KeepAlivePeriod,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          defaultMaxIdleConns,
			IdleConnTimeout:       proxyConfig.IdleConnTimeout,
			TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
			ExpectContinueTimeout: defaultExpectContinueTimeout,
		},
	}

	nProxy := &Proxy{
		proxy:  p,
		config: proxyConfig,
	}

	return nProxy
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.proxy.ServeHTTP(rw, req)
}
