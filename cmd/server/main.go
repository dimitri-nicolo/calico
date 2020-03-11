// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package main

import (
	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/license-agent/pkg/config"
	"github.com/tigera/license-agent/pkg/metrics"
	"os"
	"time"
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

	//Find file path
	cert := checkFileExists(cfg.MetricsCertFile)
	ca := checkFileExists(cfg.MetricsCaFile)
	key := checkFileExists(cfg.MetricsKeyFile)

	interval, err := time.ParseDuration(cfg.MetricPollInterval)
	if err != nil {
		log.Fatal("Failed to parse Poll Interval err: %s pollInterval:%s", err, cfg.MetricPollInterval)
		os.Exit(1)
	}
	//Create New Instance of License reporter
	lr := metrics.NewLicenseReporter(cfg.MetricsHost, cert, key, ca, interval, cfg.MetricsPort)
	//Start License metric server and scraper
	lr.Start()
}

//checkFileExist checks valid file exist, if so returns
//File Path else return empty string
func checkFileExists(name string) string {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return ""
	}

	return name
}
