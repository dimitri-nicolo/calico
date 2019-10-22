// Copyright 2019 Tigera Inc. All rights reserved.
package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	version := flag.Bool("version", false, "prints version information")
	flag.Parse()

	if *version {
		PrintVersion()
		return
	}

	config, err := LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Error loading configuration.")
	}

	// init time setup
	// TODO: @realgaurav:  move to init()
	es, err := ESSetup(config)
	if err != nil {
		log.WithError(err).Fatal("Error setting up elastic client.")
	}

	logs := AwsSetupLogSession()

	// Get start-time from ES and get the token based on that
	startTime, err := ESGetStartTime(config, es)
	if err != nil {
		log.WithError(err).Fatal("Error getting start-time from elastic.")
	}

	stateFileTokenMap, err := AwsGetStateFileWithToken(logs, config.EKSCloudwatchLogGroup, config.EKSCloudwatchLogStreamPrefix, startTime)
	if err != nil {
		log.WithError(err).Fatal("Error getting token for given start-time.")
	}

	if err = generateStateFile(config.EKSStateFileDir, stateFileTokenMap); err != nil {
		log.WithError(err).Fatal("Error generating state-file for log-stream.")
	}
}

func generateStateFile(path string, stateTokens map[string]string) error {
	for s, t := range stateTokens {
		f, err := os.OpenFile(path+s, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err = f.WriteString(t); err != nil {
			return err
		}
	}

	return nil
}
