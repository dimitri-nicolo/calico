// Copyright (c) 2022 Tigera All rights reserved.
package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/clusters"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/handler/health"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/middleware/auth"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/middleware/logging"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	log "github.com/sirupsen/logrus"
)

var (
	server    *http.Server
	waitGroup sync.WaitGroup
)

func Start(config *config.Config) {
	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
		log.WithError(err).Warnf("Unavailable LOG_LEVEL continuining with default INFO level")
	}

	log.SetLevel(logLevel)

	modelStorageHandler := clusters.NewClustersEndpointHandler(config)
	sm := http.NewServeMux()

	sm.Handle("/health", health.HealthCheck())
	sm.Handle("/clusters/", modelStorageHandler.HandleClusters())

	var apiHandler http.Handler
	if config.DebugRunWithRBACDisabled {
		// FV setting
		log.Warn("Running with RBAC disabled, this setting should only be enabled for testing purposes")
		apiHandler = sm
	} else {
		jwtAuth, err := getJWTAuthenticator()
		if err != nil {
			log.Fatal("unable to create authenticator")
		}

		apiHandler = auth.Auth(sm, jwtAuth)
	}

	handler := logging.LogRequestHeaders(apiHandler)
	server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: handler,
	}

	waitGroup.Add(1)

	go func() {
		log.Infof("Starting server on %v", config.ListenAddr)
		err := server.ListenAndServeTLS(config.TLSCert, config.TLSKey)
		if err != nil {
			log.WithError(err).Error("Error when starting server.")
		}
		waitGroup.Done()
	}()
}

func getJWTAuthenticator() (lmaauth.JWTAuth, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	k8sCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	var options []lmaauth.JWTAuthOption

	authn, err := lmaauth.NewJWTAuth(restConfig, k8sCli, options...)
	if err != nil {
		return nil, err
	}

	return authn, nil
}

func Wait() {
	waitGroup.Wait()
}

func Stop() {
	if err := server.Shutdown(context.Background()); err != nil {
		log.WithError(err).Error("Error when stopping server")
	}
}
