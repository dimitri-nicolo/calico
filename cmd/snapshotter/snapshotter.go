// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/resources"
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

	// Starting snapshotter for each resource type.
	wg := sync.WaitGroup{}
	for _, rh := range resources.GetAllResourceHelpers() {
		wg.Add(1)
		go func(rh resources.ResourceHelper) {
			err := snapshot.Run(cxt, rh.TypeMeta(), datastoreClient, elasticClient)
			if err != nil {
				log.Errorf("Hit terminating error in snapshotter: %v", err)
			}

			// This snapshotter is exiting, so tell the others to exit.
			cancel()
			wg.Done()
		}(rh)
	}

	// Wait until all snapshotters have exited.
	wg.Wait()
}
