// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/linseed/pkg/config"
	"github.com/projectcalico/calico/linseed/pkg/handler"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")
)

func main() {
	// Parse all command-line flags
	flag.Parse()

	// For --version use case
	if *versionFlag {
		handler.Version()
		os.Exit(0)
	}

	cfg := config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, &cfg); err != nil {
		panic(err)
	}

	// Configure logging
	config.ConfigureLogging(cfg.LogLevel)

	log.Debugf("Starting with %#v", cfg)

	var addr = fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	log.Infof("Listening for HTTPS requests at %s", addr)
	// Define handlers
	http.Handle("/version", handler.VersionCheck())
	http.Handle("/health", handler.HealthCheck())
	http.Handle("/metrics", promhttp.Handler())

	// Start server
	server := &http.Server{
		Addr:      addr,
		TLSConfig: tls.NewTLSConfig(cfg.FIPSModeEnabled),
	}

	log.Fatal(server.ListenAndServeTLS(cfg.HTTPSCert, cfg.HTTPSKey))
}
