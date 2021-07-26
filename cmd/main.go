// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	lmaauth "github.com/tigera/lma/pkg/auth"
	lmak8s "github.com/tigera/lma/pkg/k8s"
	cache2 "github.com/tigera/packetcapture-api/pkg/cache"
	"github.com/tigera/packetcapture-api/pkg/capture"

	"github.com/projectcalico/apiserver/pkg/authentication"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/packetcapture-api/pkg/config"
	"github.com/tigera/packetcapture-api/pkg/handlers"
	"github.com/tigera/packetcapture-api/pkg/middleware"
	"github.com/tigera/packetcapture-api/pkg/version"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")
)

func main() {
	// Parse all command-line flags
	flag.Parse()

	// For --version use case
	if *versionFlag {
		version.Version()
		os.Exit(0)
	}

	cfg := &config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, cfg); err != nil {
		log.Fatal(err)
	}

	// Configure logging
	config.ConfigureLogging(cfg.LogLevel)

	// Boostrap components
	var addr = fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	var csFactory = lmak8s.NewClientSetFactory(
		cfg.MultiClusterForwardingCA,
		cfg.MultiClusterForwardingEndpoint)
	var cache = cache2.NewClientCache(csFactory)

	var stop = make(chan struct{})
	defer close(stop)
	go func() {
		// Init the client cache with a default client
		var err = cache.Init()
		if err != nil {
			log.WithError(err).Fatal("Cannot init client cache")
		}
	}()
	var auth = middleware.NewAuth(mustGetAuthenticator(cfg), cache)
	var locator = capture.NewLocator(cache)
	var fileRetrieval = capture.NewFileRetrieval(cache)
	var download = handlers.NewDownload(cache, locator, fileRetrieval)

	log.Infof("PacketCapture API listening for HTTPS requests at %s", addr)
	// Define handlers
	http.Handle("/version", http.HandlerFunc(version.Handler))
	http.Handle("/health", http.HandlerFunc(handlers.Health))
	http.Handle("/download/", middleware.Parse(auth.Authenticate(auth.Authorize(download.Download))))

	// Start server
	log.Fatal(http.ListenAndServeTLS(addr, cfg.HTTPSCert, cfg.HTTPSKey, nil))
}

func mustGetAuthenticator(cfg *config.Config) authentication.Authenticator {
	authenticator, err := authentication.New()
	if err != nil {
		log.WithError(err).Panic("Unable to create auth configuration")
	}

	if cfg.DexEnabled {
		opts := []lmaauth.DexOption{
			lmaauth.WithGroupsClaim(cfg.DexGroupsClaim),
			lmaauth.WithJWKSURL(cfg.DexJwksUrl),
			lmaauth.WithUsernamePrefix(cfg.DexUsernamePrefix),
			lmaauth.WithGroupsPrefix(cfg.DexGroupsPrefix),
		}

		dex, err := lmaauth.NewDexAuthenticator(
			cfg.DexIssuer,
			cfg.DexClientID,
			cfg.DexUsernameClaim,
			opts...)
		if err != nil {
			log.WithError(err).Panic("Unable to create dex authenticator")
		}
		authenticator = lmaauth.NewAggregateAuthenticator(dex, authenticator)
	}

	return authenticator
}
