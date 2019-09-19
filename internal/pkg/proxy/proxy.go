// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Target describes which path is proxied to what destination URL
type Target struct {
	Path  string
	Dest  *url.URL
	Token string
	CA    *x509.Certificate

	// PathRegexp, if not nil, check if Regexp matches the path
	PathRegexp *regexp.Regexp
	// PathReplace if not nil will be used to replace PathRegexp matches
	PathReplace []byte

	// Transport to use for this target. If nil, Proxy will provide one
	Transport http.RoundTripper
}

// Proxy proxies HTTP based on the provided list of targets
type Proxy struct {
	mux *http.ServeMux
}

// New returns an initialized Proxy
func New(tgts []Target) (*Proxy, error) {
	p := &Proxy{
		mux: http.NewServeMux(),
	}

	// Wrapped in a func to be able to recover from a possible panic in HandleFunc
	err := func() (e error) {
		defer func() {
			if r := recover(); r != nil {
				e = errors.Errorf("mux.HandleFunc paniced: %s", r)
			}
		}()

		for i, t := range tgts {
			if t.Dest == nil {
				return errors.Errorf("bad target %d, no destination", i)
			}
			if t.CA != nil && t.Dest.Scheme != "https" {
				return errors.Errorf("CA configured for url scheme %q", t.Dest.Scheme)
			}
			p.mux.HandleFunc(t.Path, newTargetHandler(t))
			log.Debugf("Proxy target %q -> %q", t.Path, t.Dest)
		}

		return nil
	}()

	if err != nil {
		p = nil
	}

	return p, err
}

func newTargetHandler(tgt Target) func(http.ResponseWriter, *http.Request) {
	p := httputil.NewSingleHostReverseProxy(tgt.Dest)
	p.FlushInterval = -1

	if tgt.Transport != nil {
		p.Transport = tgt.Transport
	} else if tgt.Dest.Scheme == "https" {
		var tlsCfg *tls.Config

		if tgt.CA == nil {
			tlsCfg = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			ca := x509.NewCertPool()
			ca.AddCert(tgt.CA)
			tlsCfg = &tls.Config{
				RootCAs: ca,
			}
		}

		p.Transport = &http.Transport{
			TLSClientConfig: tlsCfg,
		}
	}

	var token string
	if tgt.Token != "" {
		token = "Bearer " + tgt.Token
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if tgt.PathRegexp != nil {
			if !tgt.PathRegexp.MatchString(r.URL.Path) {
				http.Error(w, "Not found", 404)
				log.Debugf("Received request %+v rejected by PathRegexp %q", r, tgt.PathRegexp)
				return
			}
			if tgt.PathReplace != nil {
				r.URL.Path = tgt.PathRegexp.ReplaceAllString(r.URL.Path, string(tgt.PathReplace))
			}
		}

		if token != "" {
			r.Header.Set("Authorization", token)
		}

		log.Debugf("Received request %+v will proxy to %s", r, tgt.Dest)

		p.ServeHTTP(w, r)
	}
}

// ServeHTTP knows how to proxy HTTP requests to different named targets
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

	p.mux.ServeHTTP(w, r)
}
