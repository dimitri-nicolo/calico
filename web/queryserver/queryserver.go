// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package main

import (
	"fmt"
	"net"
	"os"

	"github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	certutil "k8s.io/client-go/util/cert"

	"github.com/projectcalico/libcalico-go/lib/logutils"
	"github.com/tigera/calicoq/web/pkg/clientmgr"
	"github.com/tigera/calicoq/web/queryserver/server"
)

// Use the same certificates as the cnx-apiserver.
const queryServerCertPath = "apiserver.local.config/certificates/apiserver.crt"
const queryServerKeyPath = "apiserver.local.config/certificates/apiserver.key"

func main() {
	// Set the logging level (default to warning).
	logLevel := log.WarnLevel
	logLevelStr := os.Getenv("LOGLEVEL")
	if logLevelStr != "" {
		logLevel = logutils.SafeParseLogLevel(logLevelStr)
	}
	log.SetLevel(logLevel)

	// Load the client configuration.  Currently we only support loading from environment.
	cfg, err := clientmgr.LoadClientConfig("")
	if err != nil {
		log.Error("Error loading config")
	}
	log.Infof("Loaded client config: %#v", cfg.Spec)

	// Generate self signed cert and key files for use with the kube proxy.
	// Allowed to use self signed certs since the kube proxy will not be using
	// the certificate for identity verification.
	if err := MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		log.Errorf("Error creating self-signed certificates: %v", err)
		os.Exit(1)
	}

	// Start the server
	server.Start(":8080", cfg, queryServerKeyPath, queryServerCertPath)

	// Wait while the server is running.
	server.Wait()
}

// MaybeDefaultWithSelfSignedCerts is a minimal copy of the version in the kubernetes apiserver in order to avoid versioning issues.
func MaybeDefaultWithSelfSignedCerts(publicAddress string, alternateDNS []string, alternateIPs []net.IP) error {
	canReadCertAndKey, err := certutil.CanReadCertAndKey(queryServerCertPath, queryServerKeyPath)
	if err != nil {
		return err
	}
	if !canReadCertAndKey {
		// add localhost to the valid alternates
		alternateDNS = append(alternateDNS, "localhost")

		if cert, key, err := certutil.GenerateSelfSignedCertKey(publicAddress, alternateIPs, alternateDNS); err != nil {
			return fmt.Errorf("unable to generate self signed cert: %v", err)
		} else {
			if err := certutil.WriteCert(queryServerCertPath, cert); err != nil {
				return err
			}

			if err := certutil.WriteKey(queryServerKeyPath, key); err != nil {
				return err
			}
			glog.Infof("Generated self-signed cert (%s, %s)", queryServerCertPath, queryServerKeyPath)
		}
	}

	return nil
}
