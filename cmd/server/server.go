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

	esprox "github.com/tigera/es-proxy/pkg/middleware"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/policyrec"
	"github.com/tigera/compliance/pkg/server"
	"github.com/tigera/compliance/pkg/tls"
	"github.com/tigera/compliance/pkg/version"
	"github.com/tigera/lma/pkg/elastic"
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

	// Load config
	cfg := config.MustLoadConfig()
	cfg.InitializeLogging()

	// Create the elastic and Calico clients.
	ec := elastic.MustGetElasticClient()
	clientSet := datastore.MustGetCalicoClient()

	// Set up tls certs
	altIPs := []net.IP{net.ParseIP("127.0.0.1")}
	if err := tls.GenerateSelfSignedCertsIfNeeded("localhost", nil, altIPs, *certPath, *keyPath); err != nil {
		log.Errorf("Error creating self-signed certificates: %v", err)
		os.Exit(1)
	}

	// Setup policy recommendation helper.
	prec := policyrec.NewPolicyRecommendationEngine(datastore.MustGetKubernetesClient(), ec)

	// Create and start the server
	auth := esprox.NewK8sAuth(datastore.MustGetKubernetesClient(), datastore.MustGetConfig(), cfg.EnableMultiClusterClient)
	rbhf := server.NewStandardRbacHelperFactory(auth)
	s := server.New(ec, clientSet, rbhf, prec, ":"+*apiPort, *keyPath, *certPath)

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
