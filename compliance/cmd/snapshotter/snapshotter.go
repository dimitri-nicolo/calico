// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog"

	"github.com/projectcalico/calico/libcalico-go/lib/health"

	log "github.com/sirupsen/logrus"
	"github.com/projectcalico/calico/compliance/pkg/config"
	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/compliance/pkg/snapshot"
	"github.com/projectcalico/calico/compliance/pkg/version"
	"github.com/projectcalico/calico/lma/pkg/elastic"
)

const (
	// The health name for the snapshotter component.
	healthReporterName = "compliance-snapshotter"
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

	// Load config.
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()
	log.WithField("config", cfg).Info("Loaded configuration")

	// Create a health check aggregator and start the health check service.
	h := health.NewHealthAggregator()
	h.ServeHTTP(cfg.HealthEnabled, cfg.HealthHost, cfg.HealthPort)
	h.RegisterReporter(healthReporterName, &health.HealthReport{Live: true}, cfg.HealthTimeout)

	// Define a function that can be used to report health.
	healthy := func(healthy bool) {
		h.Report(healthReporterName, &health.HealthReport{Live: healthy})
	}

	// Init elastic.
	elasticClient := elastic.MustGetElasticClient()

	// Create clientset.
	datastoreClient := datastore.MustGetClientSet()

	// Indicate healthy.
	healthy(true)

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cxt, cancel := context.WithCancel(context.Background())

	go func() {
		signal := <-sigs
		log.WithField("signal", signal).Warn("Received signal, canceling context")
		cancel()
	}()

	// Run snapshotter.
	if err := snapshot.Run(cxt, cfg, datastoreClient, elasticClient, healthy); err != nil {
		log.WithError(err).Error("Hit terminating error")
	}
}
