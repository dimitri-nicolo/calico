// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/projectcalico/libcalico-go/lib/health"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/snapshot"
	"github.com/tigera/compliance/pkg/version"
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		version.Version()
		return
	}

	// Load config.
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()

	// Init elastic.
	elasticClient, err := elastic.NewFromConfig(cfg)
	if err != nil {
		panic(err)
	}

	// Create clientset.
	datastoreClient := datastore.MustGetClientSet()

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cxt, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	// Setup healthchecker.
	healthAgg := health.NewHealthAggregator()
	healthAgg.RegisterReporter(snapshot.HealthName, &health.HealthReport{true, true}, cfg.HealthTimeout)
	healthAgg.ServeHTTP(true, cfg.HealthHost, cfg.HealthPort)

	// Run snapshotter.
	if err := snapshot.Run(cxt, cfg, datastoreClient, elasticClient, healthAgg); err != nil {
		log.WithError(err).Error("Hit terminating error")
	}
}
