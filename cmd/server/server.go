// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/server"
)

func main() {
	logLevel := log.InfoLevel
	logLevelStr := os.Getenv("LOG_LEVEL")
	parsedLogLevel, err := log.ParseLevel(logLevelStr)
	if err == nil {
		logLevel = parsedLogLevel
	} else {
		log.Warnf("Could not parse log level %v, setting log level to %v", logLevelStr, logLevel)
	}
	log.SetLevel(logLevel)

	targetURLStr := os.Getenv("TARGET_URL")
	targetURL, err := url.Parse(targetURLStr)
	if err != nil || targetURLStr == "" {
		log.Fatalf("Cannot parse target URL %v", targetURLStr)
	}

	server.Start("127.0.0.1:8080", targetURL)

	server.Wait()
}
