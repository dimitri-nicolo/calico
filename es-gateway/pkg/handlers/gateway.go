// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package handlers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/errors"
	"github.com/projectcalico/calico/crypto/tigeratls"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/es-gateway/pkg/middlewares"
	"github.com/projectcalico/calico/es-gateway/pkg/proxy"
)

// GetProxyHandler generates an HTTP proxy handler based on the given Target.
func GetProxyHandler(t *proxy.Target, modifyResponseFunc func(*http.Response) error) (http.HandlerFunc, error) {
	p := httputil.NewSingleHostReverseProxy(t.Dest)
	p.FlushInterval = -1

	// Augment the default director that is created by httputil.NewSingleHostReverseProxy
	// because we need to explicitly set the Host header, which is not done by default.
	defaultDirector := p.Director
	p.Director = func(req *http.Request) {
		// Run logic from the defaultDirector first to set things up for the request.
		defaultDirector(req)

		// Set the request Host explicitly, so it's not set to the default value.
		// Request URL Host should be the correct value at this point.
		req.Host = req.URL.Host
	}

	if t.Transport != nil {
		p.Transport = t.Transport
	} else if t.Dest.Scheme == "https" {
		tlsCfg := tigeratls.NewTLSConfig(t.FIPSModeEnabled)

		if t.AllowInsecureTLS {
			tlsCfg.InsecureSkipVerify = true
		} else {
			if len(t.CAPem) == 0 {
				return nil, errors.Errorf("failed to create target handler for path %s: ca bundle was empty", t.Dest)
			}

			log.Debugf("Detected secure transport for %s. Will pick up system cert pool", t.Dest)
			var ca *x509.CertPool
			ca, err := x509.SystemCertPool()
			if err != nil {
				log.WithError(err).Warn("failed to get system cert pool, creating a new one")
				ca = x509.NewCertPool()
			}

			file, err := ioutil.ReadFile(t.CAPem)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not read cert from file %s", t.CAPem))
			}

			ca.AppendCertsFromPEM(file)
			tlsCfg.RootCAs = ca

			if t.EnableMutualTLS {
				clientCert, err := tls.LoadX509KeyPair(t.ClientCert, t.ClientKey)
				if err != nil {
					return nil, err
				}
				tlsCfg.Certificates = []tls.Certificate{clientCert}
			}
		}

		p.Transport = &http.Transport{
			TLSClientConfig: tlsCfg,
		}

		// Use the modify response hook function to log the return value for response.
		// This is useful for troubleshooting and debugging.
		p.ModifyResponse = modifyResponseFunc
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(middlewares.ESUserKey)
		// User could be nil if this is a path that does not require authentication.
		if user != nil {
			user, ok := r.Context().Value(middlewares.ESUserKey).(*middlewares.User)
			// This should never happen (logical bug somewhere else in the code). But we'll
			// leave this check here to help catch it.
			if !ok {
				log.Error("unable to authenticate user: ES user cannot be pulled from context (this is a logical bug)")
				http.Error(w, "unable to authenticate user", http.StatusUnauthorized)
				return
			}
			log.Debugf("Received request %s from %s (authenticated for user %s), will proxy to %s", r.RequestURI, r.RemoteAddr, user.Username, t.Dest)
		}

		p.ServeHTTP(w, r)
	}, nil
}
