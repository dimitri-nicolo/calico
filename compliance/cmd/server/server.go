// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/caimeo/iniflags"

	"github.com/projectcalico/calico/compliance/pkg/config"
	"github.com/projectcalico/calico/compliance/pkg/datastore"
	"github.com/projectcalico/calico/compliance/pkg/server"
	"github.com/projectcalico/calico/compliance/pkg/tls"
	"github.com/projectcalico/calico/compliance/pkg/version"
	"github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/elastic"

	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var (
	versionFlag       = flag.Bool("version", false, "Print version information")
	certPath          = flag.String("certpath", "apiserver.local.config/certificates/apiserver.crt", "tls cert path")
	keyPath           = flag.String("keypath", "apiserver.local.config/certificates/apiserver.key", "tls key path")
	apiPort           = flag.String("api-port", "5443", "web api port to listen on")
	disableLogfmtFlag = flag.Bool("disable-logfmt", false, "disable logfmt style logging")
)

func init() {
	// Tell klog to log into STDERR.
	var sflags flag.FlagSet
	klog.InitFlags(&sflags)
	err := sflags.Set("logtostderr", "true")
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
	restConfig := datastore.MustGetConfig()
	k8sClientFactory := datastore.NewClusterCtxK8sClientFactory(restConfig, cfg.MultiClusterForwardingCA,
		cfg.MultiClusterForwardingEndpoint)
	// Set up tls certs
	altIPs := []net.IP{net.ParseIP("127.0.0.1")}
	if err := tls.GenerateSelfSignedCertsIfNeeded("localhost", nil, altIPs, *certPath, *keyPath); err != nil {
		log.Errorf("Error creating self-signed certificates: %v", err)
		os.Exit(1)
	}

	var authn auth.JWTAuth
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Unable to create client config", err)
	}
	//restConfig.
	k8sCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal("Unable to create kubernetes interface", err)
	}

	var options []auth.JWTAuthOption
	if cfg.OIDCAuthEnabled {
		log.Debug("Configuring Dex for authentication")
		opts := []auth.DexOption{
			auth.WithGroupsClaim(cfg.OIDCAuthGroupsClaim),
			auth.WithJWKSURL(cfg.OIDCAuthJWKSURL),
			auth.WithUsernamePrefix(cfg.OIDCAuthUsernamePrefix),
			auth.WithGroupsPrefix(cfg.OIDCAuthGroupsPrefix),
		}
		if cfg.CalicoCloudRequireTenantClaim {
			if cfg.CalicoCloudTenantClaim == "" {
				log.Panic("Tenant claim not specified")
			}
			opts = append(opts, auth.WithCalicoCloudTenantClaim(cfg.CalicoCloudTenantClaim))
		}
		dex, err := auth.NewDexAuthenticator(
			cfg.OIDCAuthIssuer,
			cfg.OIDCAuthClientID,
			cfg.OIDCAuthUsernameClaim,
			opts...)
		if err != nil {
			log.Fatal("Unable to add an issuer to the authenticator", err)
		}
		options = append(options, auth.WithAuthenticator(cfg.OIDCAuthIssuer, dex))
	}
	authn, err = auth.NewJWTAuth(restConfig, k8sCli, options...)
	if err != nil {
		log.Fatal("Unable to create authenticator", err)
	}

	esClientFactory := elastic.NewClusterContextClientFactory(elastic.MustLoadConfig())
	s := server.New(k8sClientFactory, esClientFactory, authn, ":"+*apiPort, *keyPath, *certPath, cfg.FIPSModeEnabled)
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
