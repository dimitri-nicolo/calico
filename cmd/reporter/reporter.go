package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/elastic"
)

func main() {
	var ver bool
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.Parse()

	if ver {
		Version()
		return
	}
	e := elastic.NewElasticFromEnv()

	log.Infof("Created %s", e)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
