// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/config"
	"github.com/tigera/voltron/internal/pkg/proxy"
	targets2 "github.com/tigera/voltron/internal/pkg/targets"
	"net/http"
)

func main() {
	cfg := config.Config{}
	if err := envconfig.Process("VOLTRON_AGENT", &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting VOLTRON_AGENT with configuration %v", cfg)

	targets := targets2.CreateStaticTargets()
	log.Infof("Targets are: %v", targets)
	handler := proxy.New(proxy.NewPathMatcher(targets))
	http.Handle("/", handler)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	log.Infof("Starting web server on %v", url)
	cert := fmt.Sprintf("%s/ca.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/ca.key", cfg.CertPath)
	if err := http.ListenAndServeTLS(url, cert, key, nil); err != nil {
		log.Fatal(err)
	}
}
