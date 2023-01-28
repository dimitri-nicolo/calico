// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/projectcalico/calico/linseed/pkg/backend"

	"github.com/projectcalico/calico/linseed/pkg/backend/legacy"
	"github.com/projectcalico/calico/linseed/pkg/handler/l3"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/server"

	"github.com/projectcalico/calico/linseed/pkg/config"
)

func main() {
	// Read and reconcile configuration
	cfg := config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, &cfg); err != nil {
		panic(err)
	}

	// Configure logging
	config.ConfigureLogging(cfg.LogLevel)
	log.Debugf("Starting with %#v", cfg)

	// Register for termination signals
	var signalChan = make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	//TODO: check if we need to add es connection as part of the ready probe
	esClient := backend.MustGetElasticClient(toElasticConfig(cfg))
	logsBackend := legacy.NewFlowLogBackend(esClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err := logsBackend.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	flowBackend := legacy.NewFlowBackend(esClient)

	// Start server
	var addr = fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	server := server.NewServer(addr, cfg.FIPSModeEnabled,
		server.WithMiddlewares(server.Middlewares(cfg)),
		server.WithAPIVersionRoutes("/api/v1", server.UnpackRoutes(
			l3.NewNetworkFlows(flowBackend),
			&l3.NetworkLogs{},
		)...),
		server.WithRoutes(server.UtilityRoutes()...),
	)

	go func() {
		log.Infof("Listening for HTTPS requests at %s", addr)
		if err := server.ListenAndServeTLS(cfg.HTTPSCert, cfg.HTTPSKey); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Listen for termination signals
	<-signalChan

	// Graceful shutdown of the server
	shutDownCtx, shutDownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutDownCancel()
	if err := server.Shutdown(shutDownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
	log.Info("Server is shutting down")
}

func toElasticConfig(cfg config.Config) backend.ElasticConfig {
	if cfg.ElasticUsername == "" || cfg.ElasticPassword == "" {
		log.Warn("No credentials were passed in for Elastic. Will connect to ES without credentials")

		return backend.ElasticConfig{
			URL:             cfg.ElasticEndpoint,
			LogLevel:        cfg.LogLevel,
			FIPSModeEnabled: cfg.FIPSModeEnabled,
			GZIPEnabled:     cfg.ElasticGZIPEnabled,
			Scheme:          cfg.ElasticScheme,
			EnableSniffing:  cfg.ElasticSniffingEnabled,
		}
	}

	return backend.ElasticConfig{
		URL:               cfg.ElasticEndpoint,
		Username:          cfg.ElasticUsername,
		Password:          cfg.ElasticPassword,
		LogLevel:          cfg.LogLevel,
		CACertPath:        cfg.ElasticCABundlePath,
		ClientCertPath:    cfg.ElasticClientCertPath,
		ClientCertKeyPath: cfg.ElasticClientKeyPath,
		FIPSModeEnabled:   cfg.FIPSModeEnabled,
		GZIPEnabled:       cfg.ElasticGZIPEnabled,
		Scheme:            cfg.ElasticScheme,
	}
}
