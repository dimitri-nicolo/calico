package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/projectcalico/libcalico-go/lib/health"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/benchmark"
	"github.com/tigera/compliance/pkg/cis"
	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/version"
)

const (
	healthReporterName = "compliance-benchmarker"
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
	elasticClient, err := elastic.NewFromConfig(cfg)
	if err != nil {
		log.WithError(err).Panic("Unable to connect to Elasticsearch")
	}

	// Indicate healthy
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

	// Run benchmarker.
	if err := benchmark.Run(cxt, cfg, cis.NewBenchmarker(), elasticClient, elasticClient, healthy); err != nil {
		log.WithError(err).Error("Hit terminating error")
	}
}
