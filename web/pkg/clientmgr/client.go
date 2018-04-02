// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package clientmgr

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
)

const (
	DefaultConfigPath = ""
)

// LoadClientConfig loads the client config from file if the file exists,
// otherwise will load from environment variables.
func LoadClientConfig(cf string) (*apiconfig.CalicoAPIConfig, error) {
	if _, err := os.Stat(cf); err != nil {
		if cf != DefaultConfigPath {
			fmt.Printf("Error reading config file: %s\n", cf)
			os.Exit(1)
		}
		log.Infof("Config file: %s cannot be read - reading config from environment", cf)
		cf = ""
	}

	return apiconfig.LoadClientConfig(cf)
}
