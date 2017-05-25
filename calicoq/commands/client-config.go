// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package commands

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
)

// LoadClientConfig loads the client config from file if the file exists,
// otherwise will load from environment variables.
func LoadClientConfig(cf string) (*api.CalicoAPIConfig, error) {
	if _, err := os.Stat(cf); err != nil {
		log.Infof("Config file cannot be read - reading config from environment")
		cf = ""
	}

	return client.LoadClientConfig(cf)
}
