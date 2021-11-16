// Copyright (c) 2021 Tigera. All rights reserved.
package server

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/lma/pkg/auth"
	health "github.com/tigera/prometheus-service/pkg/handler/health"
	proxy "github.com/tigera/prometheus-service/pkg/handler/proxy"
	"github.com/tigera/prometheus-service/pkg/middleware"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

func Start(config *Config) {
	sm := http.NewServeMux()

	reverseProxy := getReverseProxy(config.PrometheusUrl)

	var authn auth.JWTAuthenticator
	if config.AuthenticationEnabled {
		options := []auth.JWTAuthenticatorOption{
			auth.WithInClusterConfiguration(),
		}

		if config.DexEnabled {
			log.Debug("Configuring Dex for authentication")
			opts := []auth.DexOption{
				auth.WithGroupsClaim(config.OIDCAuthGroupsClaim),
				auth.WithJWKSURL(config.OIDCAuthJWKSURL),
				auth.WithUsernamePrefix(config.OIDCAuthUsernamePrefix),
				auth.WithGroupsPrefix(config.OIDCAuthGroupsPrefix),
			}
			dex, err := auth.NewDexAuthenticator(
				config.OIDCAuthIssuer,
				config.OIDCAuthClientID,
				config.OIDCAuthUsernameClaim,
				opts...)
			if err != nil {
				log.Fatal("Unable to add an issuer to the authenticator", err)
			}
			options = append(options, auth.WithAuthenticator(config.OIDCAuthIssuer, dex))
		}
		var err error
		authn, err = auth.NewJWTAuthenticator(options...)
		if err != nil {
			log.Fatal("Unable to create authenticator", err)
		}
	}

	proxyHandler, err := proxy.Proxy(reverseProxy, authn)
	if err != nil {
		log.Fatal("Unable to create proxy handler", err)
	}

	sm.Handle("/health", health.HealthCheck())

	sm.Handle("/", proxyHandler)

	server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: middleware.LogRequestHeaders(sm),
	}

	wg.Add(1)

	go func() {
		log.Infof("Starting server on %v", config.ListenAddr)
		err := server.ListenAndServeTLS(config.TLSCert, config.TLSKey)
		if err != nil {
			log.WithError(err).Error("Error when starting server.")
		}
		wg.Done()
	}()
}

func getReverseProxy(target *url.URL) *httputil.ReverseProxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	// applies the prometheus target URL to the request
	reverseProxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	}

	return reverseProxy
}

func Wait() {
	wg.Wait()
}

func Stop() {
	if err := server.Shutdown(context.Background()); err != nil {
		log.WithError(err).Error("Error when stopping server")
	}
}
