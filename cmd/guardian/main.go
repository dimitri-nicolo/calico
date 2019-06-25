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
	// until health check restored
	//Port       int    `default:"5555"`
	//Host       string `default:"localhost"`
	LogLevel   string `default:"DEBUG"`
	CertPath   string `default:"/certs" split_words:"true"`
	VoltronURL string `required:"true" split_words:"true"`
}

func main() {
	cfg := config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
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

	client, err := client.New(
		cfg.VoltronURL,
		client.WithProxyTargets(
			[]client.ProxyTarget{
				{Pattern: "^/api", Dest: "https://kubernetes.default"},
				{Pattern: "^/tigera-elasticsearch", Dest: "http://localhost:8002"},
			},
		),
		client.WithTunnelCreds(pemCert, pemKey, ca),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	if err := client.ServeTunnelHTTP(); err != nil {
		log.Fatalf("Tunnel exited with error: %s", err)
	}
}
