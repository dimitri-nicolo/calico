// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	lma "github.com/tigera/lma/pkg/elastic"

	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/dpisyncer"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/typha/pkg/buildinfo"
	"github.com/projectcalico/calico/typha/pkg/syncclientutils"
	"github.com/projectcalico/calico/typha/pkg/syncproto"

	"github.com/tigera/deep-packet-inspection/pkg/calicoclient"
	"github.com/tigera/deep-packet-inspection/pkg/config"
	"github.com/tigera/deep-packet-inspection/pkg/dispatcher"
	"github.com/tigera/deep-packet-inspection/pkg/dpiupdater"
	"github.com/tigera/deep-packet-inspection/pkg/elastic"
	"github.com/tigera/deep-packet-inspection/pkg/eventgenerator"
	"github.com/tigera/deep-packet-inspection/pkg/file"
	"github.com/tigera/deep-packet-inspection/pkg/processor"
	"github.com/tigera/deep-packet-inspection/pkg/syncer"
	"github.com/tigera/deep-packet-inspection/pkg/version"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")
)

const (
	healthReporterName       = "deep-packet-inspection"
	elasticRetrySendInterval = 30 * time.Second
	fileMaintenanceInterval  = 5 * time.Minute
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
	h.RegisterReporter(healthReporterName, &health.HealthReport{Ready: true}, cfg.HealthTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthCh := make(chan bool)
	go runHealthChecks(ctx, h, healthCh, cfg.HealthTimeout)

	// Indicate healthy.
	healthCh <- true

	if cfg.NodeName == "" {
		healthCh <- false
		log.Fatal("NODENAME environment is not set")
	}

	lmaESClient, err := lma.NewFromConfig(lma.MustLoadConfig())
	if err != nil {
		log.WithError(err).Fatal("Could not connect to Elasticsearch")
	}

	esForwarder, err := elastic.NewESForwarder(lmaESClient, elasticRetrySendInterval)
	if err != nil {
		log.Fatal(err)
	}

	alertFileMaintainer := file.NewFileMaintainer(fileMaintenanceInterval)
	_, client := calicoclient.MustCreateClient()
	dpiStatusUpdater := dpiupdater.NewDPIStatusUpdater(client, cfg.NodeName)

	// Either create a typha syncclient or a local syncerClient depending on configuration. This calls back into the
	// syncer to trigger updates when necessary.
	syncerCb := syncer.NewSyncerCallbacks(healthCh)
	typhaConfig := syncclientutils.ReadTyphaConfig([]string{"DPI_"})
	if syncclientutils.MustStartSyncerClientIfTyphaConfigured(
		&typhaConfig, syncproto.SyncerTypeDPI,
		buildinfo.GitVersion, cfg.NodeName, fmt.Sprintf("dpi %s", buildinfo.GitVersion),
		syncerCb,
	) {
		log.Debug("Using typha syncerClient")
	} else {
		log.Debug("Using local syncerClient")
		syncerClient := dpisyncer.New(client.(backendClientAccessor).Backend(), syncerCb)
		syncerClient.Start()
		defer syncerClient.Stop()
	}

	dispatcher := dispatcher.NewDispatcher(cfg,
		processor.NewProcessor,
		eventgenerator.NewEventGenerator,
		esForwarder,
		dpiStatusUpdater,
		alertFileMaintainer)
	defer dispatcher.Close()

	esForwarder.Run(ctx)
	alertFileMaintainer.Run(ctx)

	// Run the syncer to receive resource updates from either typha or local syncerClient and pass it to dispatcher.
	syncerCb.Sync(ctx, dispatcher)
}

// runHealthChecks receives updates on the healthCh and reports it, it also keeps track of the latest health and resends it on interval.
func runHealthChecks(ctx context.Context, h *health.HealthAggregator, healthCh chan bool, healthDuration time.Duration) {
	ticker := time.NewTicker(healthDuration)
	defer ticker.Stop()

	var lastHealth bool
	for {
		select {
		case lastHealth = <-healthCh:
			// send the latest health report
			h.Report(healthReporterName, &health.HealthReport{Ready: lastHealth})
		case <-ticker.C:
			// resend the last health reported if health hasn't changed
			h.Report(healthReporterName, &health.HealthReport{Ready: lastHealth})
		case <-ctx.Done():
			return
		}
	}
}

// backendClientAccessor is an interface to access the backend client from the bapi client.
type backendClientAccessor interface {
	Backend() bapi.Client
}
