// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/client"
	"github.com/tigera/voltron/internal/pkg/config"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "GUARDIAN"
)

func main() {
	cfg := config.Config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %v", EnvConfigPrefix, cfg)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	cert := fmt.Sprintf("%s/ca.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/ca.key", cfg.CertPath)
	log.Infof("Path: %s %s", cert, key)
	client, err := client.New(
		client.WithProxyTargets(
			[]client.ProxyTarget{
				{Pattern: "^/api", Dest: "https://kubernetes.default"},
				{Pattern: "^/tigera-elasticsearch", Dest: "http://localhost:8002"},
			},
		),
		client.WithDefaultAddr(url),
		client.WithCredsFiles(cert, key),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	log.Infof("Starting web server on %v", url)

	if err := client.ListenAndServeTLS(); err != nil {
		log.Fatal(err)
	}
}
