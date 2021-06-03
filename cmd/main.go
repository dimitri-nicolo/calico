package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/es-gateway/pkg/config"
	"github.com/tigera/es-gateway/pkg/proxy"
	"github.com/tigera/es-gateway/pkg/server"
	"github.com/tigera/es-gateway/pkg/version"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")
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

	cfg.SetupLogging()
	log.Infof("Starting %s with %s", config.EnvConfigPrefix, cfg)

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	esTarget, err := proxy.CreateTarget(
		cfg.ElasticPathPrefixes,
		cfg.ElasticEndpoint,
		cfg.ElasticCABundlePath,
		false,
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Kibana target for ES Gateway.")
	}

	kibanaTarget, err := proxy.CreateTarget(
		cfg.KibanaPathPrefixes,
		cfg.KibanaEndpoint,
		cfg.KibanaCABundlePath,
		false,
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to create ES target for ES Gateway.")
	}

	opts := []server.Option{
		server.WithAddr(addr),
		server.WithInternalTLSFiles(cfg.HTTPSCert, cfg.HTTPSKey),
	}

	srv, err := server.New(esTarget, kibanaTarget, opts...)
	if err != nil {
		log.WithError(err).Fatal("Failed to create ES Gateway server.")
	}

	log.Infof("ES Gateway listening for HTTPS requests at %s", addr)
	log.Fatal(srv.ListenAndServeHTTPS())
}
