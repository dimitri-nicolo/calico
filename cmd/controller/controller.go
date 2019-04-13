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
	"github.com/tigera/compliance/pkg/elastic"
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

	// Load environment config.
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()

	// Create the clientset.
	cs := datastore.MustGetClientSet()

	// Create the elastic client. We only use this to determine the last recorded report.
	rr := elastic.MustGetElasticClient(cfg)

	// Create a health check aggregator and start the health check service.
	h := health.NewHealthAggregator()
	h.ServeHTTP(cfg.HealthEnabled, cfg.HealthHost, cfg.HealthPort)

	ctrl, err := controller.NewComplianceController(cfg, cs, rr, h)
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
