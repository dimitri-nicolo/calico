// Copyright (c) 2022 Tigera All rights reserved.
package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/api"
	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
)

func main() {
	config, err := config.NewConfigFromEnv()

	if err != nil {
		log.WithError(err).Fatal("Configuration Error.")
	}

	api.Start(config)
	api.Wait()
}
