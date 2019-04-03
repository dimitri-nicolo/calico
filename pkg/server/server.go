// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/middleware"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

// TODO(doublek): This should be moved to a config file.
var (
	flowLogIndexQuery  = "/tigera_secure_ee_flows*/_search"
	auditLogIndexQuery = "/tigera_secure_ee_audit*/_search"
	eventsIndexQuery   = "/tigera_secure_ee_event*/_search"
)

func Start(config *Config) error {
	sm := http.NewServeMux()

	// Initialize all handlers and middlewares here. Proxy is handler as well.
	proxy := handler.NewProxy(config.ElasticURL)

	validPaths := map[string]struct{}{
		flowLogIndexQuery:  struct{}{},
		auditLogIndexQuery: struct{}{},
		eventsIndexQuery:   struct{}{},
	}
	// For now we are a proxy for all requests so only add the default
	// handler.
	// TODO(doublek):
	//  - This could be nicer. Seems a bit kludgy to add middlewares like this.
	//  - Logging only logs the frontend requests and not the backend response. We could
	//    move the logger to the end and make it log responses if present.
	sm.Handle("/version", middleware.LogRequestHeaders(http.HandlerFunc(handler.VersionHandler)))
	sm.Handle("/", middleware.LogRequestHeaders(middleware.PathValidator(validPaths, proxy)))

	server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: sm,
	}

	wg.Add(1)
	go func() {
		log.Infof("Starting server on %v", config.ListenAddr)
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
