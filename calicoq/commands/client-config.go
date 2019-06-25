// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package commands

import (
	// "context"
	"os"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/backend"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"

	// "github.com/projectcalico/libcalico-go/lib/backend/model"
	log "github.com/sirupsen/logrus"
)

// LoadClientConfig loads the client config from file if the file exists,
// otherwise will load from environment variables.
func LoadClientConfig(cf string) (*apiconfig.CalicoAPIConfig, error) {
	if _, err := os.Stat(cf); err != nil {
		log.Infof("Config file cannot be read - reading config from environment")
		cf = ""
	}

	return apiconfig.LoadClientConfig(cf)
}

func GetClient(cf string) (bapi.Client, *apiconfig.CalicoAPIConfig) {
	apiConfig, err := LoadClientConfig(cf)
	if err != nil {
		log.Fatal("Failed loading client config")
		os.Exit(1)
	}
	bclient, err := backend.NewClient(*apiConfig)
	if err != nil {
		log.Fatal("Failed to create client")
		os.Exit(1)
	}
	/*
		// TODO: Need to reenable this when the EnsureInitialized method for etcdv3 is populated
		ctx := context.Background()
		if kv, err := bclient.Get(ctx, model.ReadyFlagKey{}, ""); err != nil {
			log.WithError(err).Fatal("Failed to read datastore 'Ready' flag - is your datastore configuration correct?")
			os.Exit(1)
		} else if kv.Value != true {
			log.Fatal("Datastore 'Ready' flag is false - can't run calicoq on a non-ready datastore")
			os.Exit(1)
		}
	*/

	return bclient, apiConfig
}
