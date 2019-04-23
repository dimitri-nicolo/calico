// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/projectcalico/libcalico-go/lib/health"
	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/snapshot"
	"github.com/tigera/compliance/pkg/version"
)

const (
	healthHost       = "localhost"
	healthPort       = 55555
	keepAliveTimeout = 10 * time.Minute
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		version.Version()
		return
	}

	// Set up logger.
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(logutils.SafeParseLogLevel(os.Getenv("LOG_LEVEL")))

	// Init elastic.
	elasticClient, err := elastic.NewFromEnv()
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
	healthAgg.RegisterReporter(snapshot.HealthName, &health.HealthReport{true, true}, keepAliveTimeout)
	healthAgg.ServeHTTP(true, healthHost, healthPort)

	// Run snapshotter.
	if err := snapshot.Run(cxt, datastoreClient, elasticClient, healthAgg); err != nil {
		log.WithError(err).Error("Hit terminating error")
	}
}
