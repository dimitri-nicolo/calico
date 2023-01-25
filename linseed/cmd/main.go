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

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/linseed/pkg/backend/legacy"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"

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

	//TODO: secure connections with ES
	//TODO: check if we need to add es connection as part of the ready probe
	esClient, err := elastic.NewClient(
		elastic.SetURL(cfg.ElasticEndpoint),
		elastic.SetErrorLog(log.New()),
		elastic.SetInfoLog(log.New()))
	if err != nil {
		log.Fatal(err)
	}
	client := lmaelastic.NewWithClient(esClient)
	logsBackend := legacy.NewFlowLogBackend(client)
	err = logsBackend.Initialize(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	flowBackend := legacy.NewFlowBackend(client)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
	log.Info("Server is shutting down")
}
