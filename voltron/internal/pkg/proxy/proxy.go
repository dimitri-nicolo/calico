// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package proxy

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"

	"github.com/pkg/errors"

	"github.com/projectcalico/calico/crypto/pkg/tls"

	log "github.com/sirupsen/logrus"
)

// Target describes which path is proxied to what destination URL
type Target struct {
	Path  string
	Dest  *url.URL
	Token string
	CAPem string

	// PathRegexp, if not nil, check if Regexp matches the path
	PathRegexp *regexp.Regexp
	// PathReplace if not nil will be used to replace PathRegexp matches
	PathReplace []byte

	// Transport to use for this target. If nil, Proxy will provide one
	Transport        http.RoundTripper
	AllowInsecureTLS bool

	// Enables FIPS 140-2 verified mode.
	FIPSModeEnabled bool
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

	for i, t := range tgts {
		if t.Dest == nil {
			return nil, errors.Errorf("bad target %d, no destination", i)
		}
		if len(t.CAPem) != 0 && t.Dest.Scheme != "https" {
			log.Debugf("Configuring CA cert for secure communication %s for %s", t.CAPem, t.Dest.Scheme)
			return nil, errors.Errorf("CA configured for url scheme %q", t.Dest.Scheme)
		}
		hdlr, err := newTargetHandler(t)
		if err != nil {
			return nil, err
		}
		p.mux.HandleFunc(t.Path, hdlr)
		log.Debugf("Proxy target %q -> %q", t.Path, t.Dest)
	}

	return p, nil
}

func newTargetHandler(tgt Target) (func(http.ResponseWriter, *http.Request), error) {
	p := httputil.NewSingleHostReverseProxy(tgt.Dest)
	p.FlushInterval = -1

	if tgt.Transport != nil {
		p.Transport = tgt.Transport
	} else if tgt.Dest.Scheme == "https" {
		tlsCfg := tls.NewTLSConfig(tgt.FIPSModeEnabled)

		if tgt.AllowInsecureTLS {
			tlsCfg.InsecureSkipVerify = true
		} else {
			if len(tgt.CAPem) == 0 {
				return nil, errors.Errorf("failed to create target handler for path %s: ca bundle was empty", tgt.Path)
			}

			log.Debugf("Detected secure transport for %s. Will pick up system cert pool", tgt.Dest)
			var ca *x509.CertPool
			ca, err := x509.SystemCertPool()
			if err != nil {
				log.WithError(err).Warn("failed to get system cert pool, creating a new one")
				ca = x509.NewCertPool()
			}

			file, err := ioutil.ReadFile(tgt.CAPem)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not read cert from file %s", tgt.CAPem))
			}

			ca.AppendCertsFromPEM(file)
			tlsCfg.RootCAs = ca
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
				log.Debugf("Received request %s rejected by PathRegexp %q", r.RequestURI, tgt.PathRegexp)
				return
			}
			if tgt.PathReplace != nil {
				r.URL.Path = tgt.PathRegexp.ReplaceAllString(r.URL.Path, string(tgt.PathReplace))
			}
		}

		if token != "" {
			r.Header.Set("Authorization", token)
		}

		log.Debugf("Received request %s will proxy to %s", r.RequestURI, tgt.Dest)

		p.ServeHTTP(w, r)
	}, nil
}

// ServeHTTP knows how to proxy HTTP requests to different named targets
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

	p.mux.ServeHTTP(w, r)
}

// Get target returns the target that would be used
func (p *Proxy) GetTargetPath(r *http.Request) string {
	_, pat := p.mux.Handler(r)
	return pat
}
