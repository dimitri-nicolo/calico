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

	"k8s.io/klog"

	lmaauth "github.com/tigera/lma/pkg/auth"

	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/datastore"
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

func init() {
	// Tell glog (used by client-go) to log into STDERR. Otherwise, we risk
	// certain kinds of API errors getting logged into a directory not
	// available in a `FROM scratch` Docker container, causing glog to abort
	err := flag.Set("logtostderr", "true")
	if err != nil {
		log.WithError(err).Fatal("Failed to set logging configuration")
	}

	// Also tell klog to log to STDERR.
	var sflags flag.FlagSet
	klog.InitFlags(&sflags)
	err = sflags.Set("logtostderr", "true")
	if err != nil {
		log.WithError(err).Fatal("Failed to set logging configuration")
	}
}

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

	// Create and start the server
	auth := lmaauth.NewK8sAuth(datastore.MustGetKubernetesClient(), datastore.MustGetConfig(), cfg.EnableMultiClusterClient)
	rbhf := server.NewStandardRbacHelperFactory(auth)
	s := server.New(ec, clientSet, rbhf, ":"+*apiPort, *keyPath, *certPath)

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
