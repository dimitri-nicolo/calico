package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/logutils"

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

	// Set up logger.
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(logutils.SafeParseLogLevel(os.Getenv("LOG_LEVEL")))

	// Init elastic.
	elasticClient := elastic.MustGetElasticClient()

	// Create a Calico client and query the report and corresponding report type.
	config := report.MustLoadReportConfig()

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	cxt, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigs
		cancel()
	}()

	// Starting snapshotter for each resource type.
	log.Debugf("Elastic: %v", elasticClient)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := report.Run(cxt, config, elasticClient, elasticClient, elasticClient, elasticClient)
		if err != nil {
			log.Errorf("Hit terminating error in reporter: %v", err)
		}
		wg.Done()
	}()

	// Wait until all snapshotters have exited.
	wg.Wait()
}
