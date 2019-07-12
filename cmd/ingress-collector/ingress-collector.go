// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"sync"

	"github.com/tigera/ingress-collector/pkg/collector"
	"github.com/tigera/ingress-collector/pkg/config"
	"github.com/tigera/ingress-collector/pkg/felixclient"
	"github.com/tigera/ingress-collector/uds"
)

func main() {
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
