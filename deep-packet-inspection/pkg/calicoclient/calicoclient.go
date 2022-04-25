// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package calicoclient

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	client "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
)

// MustCreateClient loads the client config from environments and creates the
// Calico client.
func MustCreateClient() (*apiconfig.CalicoAPIConfig, client.Interface) {
	// Load the client config from environment.
	cfg, err := apiconfig.LoadClientConfig("")
	if err != nil {
		log.Fatalf("Error loading datastore config: %s", err)
	}
	c, err := client.New(*cfg)
	if err != nil {
		log.Fatalf("Error accessing the Calico datastore: %s", err)
	}

	return cfg, c
}
