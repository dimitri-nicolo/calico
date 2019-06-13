// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"context"

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
	nginxCollector := collector.NewNginxCollector()

	// TODO: Clean this up
	// Start the log collector
	StartCollector(nginxCollector, cfg)
}

func StartCollector(collector collector.IngressCollector, config *config.Config) {
	opts := uds.GetDialOptions()
	felixClient := felixclient.NewFelixClient(config.DialTarget, opts)
	felixClient.CollectAndSend(context.Background(), collector)
	return
}
