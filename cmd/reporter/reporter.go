package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/health"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/report"
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

	// Load the config.
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()

	// Init elastic.
	elasticClient := elastic.MustGetElasticClient(cfg)

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cxt, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	// Create a health check aggregator and start the health check service.
	h := health.NewHealthAggregator()
	h.ServeHTTP(cfg.HealthEnabled, cfg.HealthHost, cfg.HealthPort)

	// Run the reporter.
	log.Debugf("Elastic: %v", elasticClient)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := report.Run(cxt, cfg, h, elasticClient, elasticClient, elasticClient, elasticClient, elasticClient)
		if err != nil {
			log.Errorf("Hit terminating error in reporter: %v", err)
		}
		wg.Done()
	}()

	// Wait until all snapshotters have exited.
	wg.Wait()
}
