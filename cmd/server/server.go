// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"flag"
	"net"
	"os"

	"github.com/tigera/compliance/cmd/server/api"

	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/caimeo/iniflags"

	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/tls"
	"github.com/tigera/compliance/pkg/version"
)

var (
	versionFlag              = flag.Bool("version", false, "Print version information")
	complianceServerCertPath = flag.String("certpath", "apiserver.local.config/certificates/apiserver.crt", "tls cert path")
	complianceServerKeyPath  = flag.String("keypath", "apiserver.local.config/certificates/apiserver.key", "tls key path")
	apiPort                  = flag.String("api-port", "8080", "web api port to listen on")
	disableLogfmtFlag        = flag.Bool("disable-logfmt", false, "disable logfmt style logging")
	devFlagNoES              = flag.Bool("no-es", false, "")
)

var (
	els *elastic.Client
	sig chan os.Signal
	tf  *prefixed.TextFormatter
)

func main() {
	initIniFlags()
	handleFlags()
	initElastic()
	initAPIServer()
}

// read command line flags and/or .settings
func initIniFlags() {
	iniflags.SetConfigFile(".settings")
	iniflags.SetAllowMissingConfigFile(true)
	iniflags.Parse()
}

func handleFlags() {
	// --version
	if *versionFlag {
		version.Version()
		os.Exit(0)
	}
	// --disable_logfmt=true
	if *disableLogfmtFlag {
		log.SetFormatter(&prefixed.TextFormatter{
			ForceFormatting: true,
		})
	}
}

func initElastic() {
	if *devFlagNoES {
		return
	}
	els, err := elastic.NewFromEnv()
	if err != nil {
		log.WithError(err).Errorf("Error creating ES client.")
		os.Exit(3)
	}
	log.Infof("Created %s", els)
}

func initAPIServer() {
	//set up tls certs
	altIPs := []net.IP{net.ParseIP("127.0.0.1")}
	if err := tls.GenerateSelfSignedCertsIfNeeded("localhost", nil, altIPs, *complianceServerCertPath, *complianceServerKeyPath); err != nil {
		log.Errorf("Error creating self-signed certificates: %v", err)
		os.Exit(1)
	}

	//start the server
	if err := api.Start(":"+*apiPort, *complianceServerKeyPath, *complianceServerCertPath); err != nil {
		log.WithError(err).Error("Error starting compliance server")
		os.Exit(2)
	}

	//wait while running
	api.Wait()
}
