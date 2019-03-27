package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"

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

	// Set up logger.
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(logutils.SafeParseLogLevel(os.Getenv("LOG_LEVEL")))

	// Init elastic.
	writer, err := elastic.NewFromEnv()
	if err != nil {
		panic(err)
	}
	// Check elastic index.
	if err = writer.EnsureIndices(); err != nil {
		panic(err)
	}

	// setup signals.
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- true
	}()

	// run.
}
