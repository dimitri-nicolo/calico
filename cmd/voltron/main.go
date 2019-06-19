// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"

	"github.com/tigera/voltron/internal/pkg/utils"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/config"
	"github.com/tigera/voltron/internal/pkg/server"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "VOLTRON"
)

func main() {
	cfg := config.Config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %v", EnvConfigPrefix, cfg)

	cert := fmt.Sprintf("%s/ca.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/ca.key", cfg.CertPath)

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	//TODO: These should be different that the ones baked in
	pemCert, _ := utils.LoadPEMFromFile(cert)
	pemKey, _ := utils.LoadPEMFromFile(key)
	tunnelCert, tunnelKey, _ := utils.LoadX509KeyPairFromPEM(pemCert, pemKey)

	srv, err := server.New(
		server.WithDefaultAddr(addr),
		server.WithCredsFiles(cert, key),
		server.WithKeepClusterKeys(),
		server.WithTunnelCreds(tunnelCert, tunnelKey),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	log.Infof("Starting web server on %s", addr)
	if err := srv.ListenAndServeHTTPS(); err != nil {
		log.Fatal(err)
	}
}
