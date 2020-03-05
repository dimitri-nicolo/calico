// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"github.com/projectcalico/libcalico-go/lib/logutils"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/license-agent/pkg/metrics"
	"github.com/tigera/ts-queryserver/pkg/clientmgr"
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

	// Load the client configuration.  Currently we only support loading from environment.
	cfg, err := clientmgr.LoadClientConfig("")
	fmt.Println(cfg, err)

	//Create New Instance of License reporter
	lr := metrics.NewLicenseReporter("", "", "", "", 9081, 2)
	//Start License metric server and scraper
	lr.Start()
}
