// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"sync"

	"github.com/projectcalico/calico/ingress-collector/pkg/collector"
	"github.com/projectcalico/calico/ingress-collector/pkg/config"
	"github.com/projectcalico/calico/ingress-collector/pkg/felixclient"
	"github.com/projectcalico/calico/libcalico-go/lib/uds"
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		Version()
		return
	}

	// Create/read config
	// Load environment config.
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()

	// Instantiate the log collector
	c := collector.NewIngressCollector(cfg)

	// Instantiate the felix client
	opts := uds.GetDialOptions()
	felixClient := felixclient.NewFelixClient(cfg.DialTarget, opts)

	// Start the log collector
	CollectAndSend(context.Background(), felixClient, c)
}

func CollectAndSend(ctx context.Context, client felixclient.FelixClient, collector collector.IngressCollector) {
	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	// Start the log ingestion go routine.
	wg.Add(1)
	go func() {
		collector.ReadLogs(ctx)
		cancel()
		wg.Done()
	}()

	// Start the DataplaneStats reporting go routine.
	wg.Add(1)
	go func() {
		client.SendStats(ctx, collector)
		cancel()
		wg.Done()
	}()

	// Wait for the go routine to complete before exiting
	wg.Wait()
}
