// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
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

	var rootCAs *x509.CertPool
	if config.ElasticCAPath != "" {
		rootCAs = addCertToCertPool(config.ElasticCAPath)
	}
	var tlsConfig *tls.Config
	if rootCAs != nil {
		tlsConfig = &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: config.ElasticInsecureSkipVerify,
		}
	}

	pc := &handler.ProxyConfig{
		TargetURL:       config.ElasticURL,
		TLSConfig:       tlsConfig,
		ConnectTimeout:  config.ProxyConnectTimeout,
		KeepAlivePeriod: config.ProxyKeepAlivePeriod,
		IdleConnTimeout: config.ProxyIdleConnTimeout,
	}
	proxy := handler.NewProxy(config.ElasticURL, tlsConfig)

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

func addCertToCertPool(caPath string) *x509.CertPool {
	caContent, err := ioutil.ReadFile(caPath)
	if err != nil {
		log.WithError(err).WithField("CA-Path", caPath).Fatal("Could not read CA file")
	}

	systemCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.WithError(err).Fatal("Could not parse CA file")
	}

	ok := systemCertPool.AppendCertsFromPEM(caContent)
	if !ok {
		log.WithError(err).Fatal("Could not add CA to pool")
	}
	return systemCertPool
}
