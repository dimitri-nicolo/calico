// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package main

import (
	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/license-agent/pkg/config"
	"github.com/tigera/license-agent/pkg/metrics"
	"os"
)

func main() {

	logLevel := log.InfoLevel
	logLevelStr := os.Getenv("LOG_LEVEL")
	log.SetFormatter(&logutils.Formatter{})
	parsedLogLevel, err := log.ParseLevel(logLevelStr)
	if err == nil {
		logLevel = parsedLogLevel
	} else {
		log.Warnf("Could not parse log level %v, setting log level to %v", logLevelStr, logLevel)
	}
	log.SetLevel(logLevel)

	// Load env config
	cfg := config.MustLoadConfig()

	//Create New Instance of License reporter
	lr := metrics.NewLicenseReporter("", "", "", "", cfg.MetricsPort, cfg.MetricPollTime)
	//Start License metric server and scraper
	lr.Start()
}
