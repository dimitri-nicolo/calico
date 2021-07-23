// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"os"

	"github.com/tigera/deep-packet-inspection/pkg/config"
	"github.com/tigera/deep-packet-inspection/pkg/syncer"
	"github.com/tigera/deep-packet-inspection/pkg/version"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/health"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")
)

const (
	healthReporterName = "deep-packet-inspection"
)

func main() {
	// Parse all command-line flags
	flag.Parse()

	// For --version use case
	if *versionFlag {
		version.Version()
		os.Exit(0)
	}

	cfg := &config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, cfg); err != nil {
		log.Fatal(err)
	}

	// Configure logging
	config.ConfigureLogging(cfg.LogLevel)

	// Create a health check aggregator and start the health check service.
	h := health.NewHealthAggregator()
	h.ServeHTTP(cfg.HealthEnabled, cfg.HealthHost, cfg.HealthPort)
	h.RegisterReporter(healthReporterName, &health.HealthReport{Live: true}, cfg.HealthTimeout)

	// Define a function that can be used to report health.
	healthy := func(live bool) {
		h.Report(healthReporterName, &health.HealthReport{Live: live})
	}

	// Indicate healthy.
	healthy(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a syncer that gets data from syncer server and handles it.
	syncer.Run(ctx, healthy)
}
