// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/calicoq/web/pkg/clientmgr"
	"github.com/tigera/calicoq/web/queryserver/server"
)

func main() {
	// TODO: Make this a better check than just pulling variables
	// Possibly switch this to use the golang TLSConfig if necessary.
	webKey := os.Getenv("QUERYSERVER_KEY")
	webCert := os.Getenv("QUERYSERVER_CERT")

	// Load the client configuration.  Currently we only support loading from environment.
	cfg, err := clientmgr.LoadClientConfig("")
	if err != nil {
		log.Error("Error loading config")
	}
	log.Infof("Loaded client config: %#v", cfg.Spec)

	if webKey != "" && webCert != "" {
		server.Start(":10443", cfg, webCert, webKey)
	} else {
		server.Start(":8080", cfg, "", "")
	}

	// Wait while the server is running
	server.Wait()
}
