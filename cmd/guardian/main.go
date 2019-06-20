// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"io/ioutil"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/client"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "GUARDIAN"
)

type config struct {
	Port     int    `default:"5555"`
	Host     string `default:"localhost"`
	LogLevel string `default:"DEBUG"`
	CertPath string `default:"certs"`
	URL      string `required:"true" envconfig:"GUARDIAN_VOLTRON_URL"`
}

func main() {
	cfg := config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %v", EnvConfigPrefix, cfg)

	url := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	cert := fmt.Sprintf("%s/ca.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/ca.key", cfg.CertPath)
	log.Infof("Voltron Address: %s", cfg.URL)

	pemCert, err := ioutil.ReadFile(cert)
	if err != nil {
		log.Fatalf("Failed to load cert: %+v", err)
	}
	pemKey, err := ioutil.ReadFile(key)
	if err != nil {
		log.Fatalf("Failed to load key: %+v", err)
	}

	client, err := client.New(
		cfg.URL,
		client.WithProxyTargets(
			[]client.ProxyTarget{
				{Pattern: "^/api", Dest: "https://kubernetes.default"},
				{Pattern: "^/tigera-elasticsearch", Dest: "http://localhost:8002"},
			},
		),
		client.WithTunnelCreds(pemCert, pemKey, nil /* XXX use system CAs */),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	log.Infof("Starting web server on %v", url)

	if err := client.ServeTunnelHTTP(); err != nil {
		log.Fatal(err)
	}
}
