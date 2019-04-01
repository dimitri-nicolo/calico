// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
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

	config, err := server.NewConfigFromEnv()
	if err != nil {
		log.WithError(err).Fatal("Configuration Error.")
	}

	server.Start(config)

	server.Wait()
}
