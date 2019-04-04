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

func Start(config *Config) error {
	sm := http.NewServeMux()

	// Initialize all handlers and middlewares here. Proxy is handler as well.
	proxy := handler.NewProxy(config.ElasticURL)

	// TODO(doublek):
	//  - This could be nicer. Seems a bit kludgy to add middlewares like this.
	//  - Logging only logs the frontend requests and not the backend response. We could
	//    move the logger to the end and make it log responses if present.
	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))

	switch config.AccessMode {
	case InsecureMode:
		sm.Handle("/", proxy)
	case ServiceUserMode:
		sm.Handle("/", middleware.BasicAuthHeaderInjector(config.ElasticUsername, config.ElasticPassword, proxy))
	case PassThroughMode:
		log.Fatal("PassThroughMode not implemented yet")
	default:
		log.WithField("AccessMode", config.AccessMode).Fatal("Unknown Elasticsearch access mode.")
	}

	server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: middleware.LogRequestHeaders(sm),
	}

	wg.Add(1)
	go func() {
		log.Infof("Starting server on %v", config.ListenAddr)
		err := server.ListenAndServeTLS(config.CertFile, config.KeyFile)
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
