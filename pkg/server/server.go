// Copyright (c) 2021 Tigera. All rights reserved.
package server

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	log "github.com/sirupsen/logrus"
	handler "github.com/tigera/prometheus-service/pkg/handler/proxy"
	"github.com/tigera/prometheus-service/pkg/middleware"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

func Start(config *Config) {
	sm := http.NewServeMux()

	reverseProxy := getReverseProxy(config.PrometheusUrl)

	sm.Handle("/", handler.Proxy(reverseProxy))

	server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: middleware.LogRequestHeaders(sm),
	}

	wg.Add(1)

	go func() {
		log.Infof("Starting server on %v", config.ListenAddr)
		err := server.ListenAndServe()
		if err != nil {
			log.WithError(err).Error("Error when starting server.")
		}
		wg.Done()
	}()
}

func getReverseProxy(target *url.URL) *httputil.ReverseProxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	// applies the proemetheus target URL to the request
	reverseProxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
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
