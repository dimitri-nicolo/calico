// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/health"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/report"
	"github.com/tigera/compliance/pkg/version"
)

const (
	// The health name for the reporter component.
	healthReporterName = "compliance-reporter"
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		version.Version()
		return
	}

	// Load the config.
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()

	// Create a health check aggregator and start the health check service.
	h := health.NewHealthAggregator()
	h.ServeHTTP(cfg.HealthEnabled, cfg.HealthHost, cfg.HealthPort)
	h.RegisterReporter(healthReporterName, &health.HealthReport{Live: true}, cfg.HealthTimeout)

	// Define a function that can be used to report health.
	healthy := func() {
		h.Report(healthReporterName, &health.HealthReport{Live: true})
	}

	// Init elastic.
	elasticClient := elastic.MustGetElasticClient(cfg)

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cxt, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	// Indicate healthy.
	healthy()

	// Run the reporter.
	log.Debug("Running reporter")
	if err := report.Run(
		cxt, cfg, healthy, elasticClient, elasticClient, elasticClient,
		elasticClient, elasticClient, elasticClient,
	); err != nil {
		log.Panicf("Hit terminating error in reporter: %v", err)
	}
}
