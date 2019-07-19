// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"crypto/x509"
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

// Config is a configuration used for Guardian
type config struct {
	LogLevel          string            `default:"DEBUG"`
	CertPath          string            `default:"/certs" split_words:"true"`
	VoltronURL        string            `required:"true" split_words:"true"`
	ProxyTargets      bootstrap.Targets `required:"true" split_words:"true"`
	KeepAliveEnable   bool              `default:"true" split_words:"true"`
	KeepAliveInterval int               `default:"100" split_words:"true"`
}

func main() {
	cfg := config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	// Configure ProxyTarget
	if len(cfg.ProxyTargets) == 0 {
		log.Fatal("No targets configured")
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %+v", EnvConfigPrefix, cfg)

	cert := fmt.Sprintf("%s/guardian.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/guardian.key", cfg.CertPath)
	serverCrt := fmt.Sprintf("%s/voltron.crt", cfg.CertPath)
	log.Infof("Voltron Address: %s", cfg.VoltronURL)

	pemCert, err := ioutil.ReadFile(cert)
	if err != nil {
		log.Fatalf("Failed to load cert: %+v", err)
	}
	pemKey, err := ioutil.ReadFile(key)
	if err != nil {
		log.Fatalf("Failed to load key: %+v", err)
	}

	ca := x509.NewCertPool()
	content, _ := ioutil.ReadFile(serverCrt)
	if ok := ca.AppendCertsFromPEM(content); !ok {
		log.Fatalf("Cannot append voltron cert to ca pool: %+v", err)
	}

	tgts, err := bootstrap.ProxyTargets(cfg.ProxyTargets)
	if err != nil {
		log.Fatalf("Failed to fill targets: %s", err)
	}

	health, err := client.NewHealth()

	if err != nil {
		log.Fatalf("Failed to create health server: %s", err)
	}

	client, err := client.New(
		cfg.VoltronURL,
		client.WithKeepAliveSettings(cfg.KeepAliveEnable, cfg.KeepAliveInterval),
		client.WithProxyTargets(tgts),
		client.WithTunnelCreds(pemCert, pemKey, ca),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	go func() {
		// Health checks start meaning everything before has worked.
		if err = health.ListenAndServeHTTP(); err != nil {
			log.Fatalf("Health exited with error: %s", err)
		}
	}()

	if err := client.ServeTunnelHTTP(); err != nil {
		log.Fatalf("Tunnel exited with error: %s", err)
	}
}
