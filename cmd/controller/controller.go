// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"

	"github.com/projectcalico/libcalico-go/lib/health"

	"os"
	"os/signal"
	"syscall"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/controller"
	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/version"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	// The health name for the controller component.
	healthReporterName = "compliance-controller"
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		version.Version()
		return
	}

	// Load environment config.
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

	// Create the clientset.
	cs := datastore.MustGetClientSet()

	// Create the elastic client. We only use this to determine the last recorded report.
	rr := elastic.MustGetElasticClient()

	// Indicate healthy.
	healthy()

	// Create and run the controller.
	ctrl, err := controller.NewComplianceController(cfg, cs, rr, healthy)
	if err != nil {
		panic(err)
	}

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cxt, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	ctrl.Run(cxt)
}
