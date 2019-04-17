// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/caimeo/iniflags"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/projectcalico/libcalico-go/lib/logutils"

	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/server"
	"github.com/tigera/compliance/pkg/tls"
	"github.com/tigera/compliance/pkg/version"
)

var (
	versionFlag       = flag.Bool("version", false, "Print version information")
	certPath          = flag.String("certpath", "apiserver.local.config/certificates/apiserver.crt", "tls cert path")
	keyPath           = flag.String("keypath", "apiserver.local.config/certificates/apiserver.key", "tls key path")
	apiPort           = flag.String("api-port", "5443", "web api port to listen on")
	disableLogfmtFlag = flag.Bool("disable-logfmt", false, "disable logfmt style logging")
)

func main() {
	initIniFlags()
	handleFlags()

	// Set up logger.
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(logutils.SafeParseLogLevel(os.Getenv("LOG_LEVEL")))

	// Create the elastic and Calico clients.
	elastic := elastic.MustGetElasticClient()
	clientSet := datastore.MustGetCalicoClient()

	// Set up tls certs
	altIPs := []net.IP{net.ParseIP("127.0.0.1")}
	if err := tls.GenerateSelfSignedCertsIfNeeded("localhost", nil, altIPs, *certPath, *keyPath); err != nil {
		log.Errorf("Error creating self-signed certificates: %v", err)
		os.Exit(1)
	}

	// Create and start the server
	s := server.New(elastic, clientSet, ":"+*apiPort, *keyPath, *certPath)
	s.Start()

	// Setup signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		s.Stop()
	}()

	// Block until the server shuts down.
	s.Wait()
}

// Read command line flags and/or .settings
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
