// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"crypto/x509"
	"encoding/json"
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

type proxyTarget []client.ProxyTarget

// Decode deserializes the list of proxytargets
func (pt *proxyTarget) Decode(envVar string) error {
	var mapping map[string]string
	targetConfig := []client.ProxyTarget{}

	err := json.Unmarshal([]byte(envVar), &mapping)
	if err != nil {
		return err
	}

	for k, v := range mapping {
		targetConfig = append(targetConfig, client.ProxyTarget{Pattern: k, Dest: v})
	}

	*pt = targetConfig
	return nil
}

// Config is a configuration used for Guardian
type config struct {
	// until health check restored
	//Port       int    `default:"5555"`
	//Host       string `default:"localhost"`
	LogLevel            string      `default:"DEBUG"`
	CertPath            string      `default:"/certs" split_words:"true"`
	VoltronURL          string      `required:"true" split_words:"true"`
	ProxyTargets        proxyTarget `required:"true" split_words:"true"`
	ServiceAccountToken string      `default:"/var/run/secrets/kubernetes.io/serviceaccount/token" split_words:"true"`
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

	opts := []client.Option{
		client.WithProxyTargets(
			cfg.ProxyTargets,
		),
		client.WithTunnelCreds(pemCert, pemKey, ca),
	}

	if cfg.ServiceAccountToken != "" {
		token, err := ioutil.ReadFile(cfg.ServiceAccountToken)
		if err != nil {
			log.Fatalf("Failed to read ServiceAccountToken from %s: %s",
				cfg.ServiceAccountToken, err)
		}
		opts = append(opts, client.WithAuthBearerToken(string(token)))
	}

	client, err := client.New(cfg.VoltronURL, opts...)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	if err := client.ServeTunnelHTTP(); err != nil {
		log.Fatalf("Tunnel exited with error: %s", err)
	}
}
