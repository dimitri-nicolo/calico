// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/tigera/deep-packet-inspection/pkg/calicoclient"
	"github.com/tigera/deep-packet-inspection/pkg/processor"

	"github.com/tigera/deep-packet-inspection/pkg/config"
	"github.com/tigera/deep-packet-inspection/pkg/handler"
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

	ctx, cancel := context.WithCancel(context.Background())
	healthCh := make(chan bool)
	go runHealthChecks(ctx, h, healthCh, cfg.HealthTimeout)

	// Indicate healthy.
	healthCh <- true

	defer cancel()

	nodeName := os.Getenv("NODENAME")
	if nodeName == "" {
		healthCh <- false
		log.Fatal("NODENAME environment is not set")
	}

	clientCfg, client := calicoclient.MustCreateClient()
	ctrl := handler.NewResourceController(client, nodeName, cfg, processor.NewProcessor)

	// Start a syncer that gets data from syncer server and handles it.
	log.Info("Starting Syncer")
	syncer.Run(ctx, ctrl, nodeName, healthCh, clientCfg, client)
}

// runHealthChecks receives updates on the healthCh and reports it, it also keeps track of the latest health and resends it on interval.
func runHealthChecks(ctx context.Context, h *health.HealthAggregator, healthCh chan bool, healthDuration time.Duration) {
	ticker := time.NewTicker(healthDuration)
	var lastHealth bool
	for {
		select {
		case lastHealth = <-healthCh:
			// send the latest health report
			h.Report(healthReporterName, &health.HealthReport{Live: lastHealth})
		case <-ticker.C:
			// resend the last health reported if health hasn't changed
			h.Report(healthReporterName, &health.HealthReport{Live: lastHealth})
		case <-ctx.Done():
			return
		}
	}
}
