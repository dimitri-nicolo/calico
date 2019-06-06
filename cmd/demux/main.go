// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/config"
	"github.com/tigera/voltron/internal/pkg/server"
)

func main() {
	cfg := config.Config{}
	if err := envconfig.Process("VOLTRON", &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting VOLTRON with configuration %v", cfg)

	cert := fmt.Sprintf("%s/ca.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/ca.key", cfg.CertPath)

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	srv, err := server.New(
		server.WithDefaultAddr(addr),
		server.WithProxyTargets(
			[]server.ProxyTarget{{Pattern: "api", Dest: "http://localhost:3000"}},
		),
		server.WithCredsFiles(cert, key),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	log.Infof("Starting web server on", addr)
	if err := srv.ListenAndServeHTTPS(); err != nil {
		log.Fatal(err)
	}
}
