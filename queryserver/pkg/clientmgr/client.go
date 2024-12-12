// Copyright (c) 2018-2019, 2022 Tigera, Inc. All rights reserved.
package clientmgr

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
)

const (
	DefaultConfigPath = ""
)

// LoadClientConfig loads the client config from file if the file exists,
// otherwise will load from environment variables.
func LoadClientConfig(cf string) (*apiconfig.CalicoAPIConfig, error) {
	if _, err := os.Stat(cf); err != nil {
		if cf != DefaultConfigPath {
			log.WithError(err).Fatalf("Error reading config file: %s", cf)
		}
		log.Infof("Config file: %s cannot be read - reading config from environment", cf)
		cf = ""
	}

	return apiconfig.LoadClientConfig(cf)
}
