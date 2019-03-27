// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/middleware"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

func Start(listenAddr string, targetURL *url.URL) error {
	sm := http.NewServeMux()

	// Initialize all handlers and middlewares here. Proxy is handler as well.
	proxy := handler.NewProxy(targetURL)

	// TODO(doublek): This could probably be nicer.
	// For now we are a proxy for all requests so only add the default
	// handler.
	sm.Handle("/", middleware.LogRequestHeaders((proxy)))

	server = &http.Server{
		Addr:    listenAddr,
		Handler: sm,
	}

	wg.Add(1)
	go func() {
		log.Infof("Starting server on %v", listenAddr)
		// TODO(doublek): Make this TLS.
		err := server.ListenAndServe()
		if err != nil {
			log.WithError(err).Error("Error when starting server")
		}
		wg.Done()
	}()

	return nil
}

func Wait() {
	wg.Wait()
}

func Stop() {
	server.Shutdown(context.Background())
}
