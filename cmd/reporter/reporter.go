// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"k8s.io/klog"

	"github.com/projectcalico/calico/libcalico-go/lib/health"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/report"
	"github.com/tigera/compliance/pkg/version"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	// The health name for the reporter component.
	healthReporterName = "compliance-reporter"
)

var ver bool

func init() {
	// Tell klog to log into STDERR.
	var sflags flag.FlagSet
	klog.InitFlags(&sflags)
	err := sflags.Set("logtostderr", "true")
	if err != nil {
		log.WithError(err).Fatal("Failed to set logging configuration")
	}

	// Add a flag to check the version.
	flag.BoolVar(&ver, "version", false, "Print version information")
}

func main() {
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
	elasticClient := elastic.MustGetElasticClient()

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
