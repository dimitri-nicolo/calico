// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

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
	LogLevel          string            `default:"INFO"`
	CertPath          string            `default:"/certs" split_words:"true" json:"-"`
	VoltronURL        string            `required:"true" split_words:"true"`
	ProxyTargets      bootstrap.Targets `required:"true" split_words:"true"`
	KeepAliveEnable   bool              `default:"true" split_words:"true"`
	KeepAliveInterval int               `default:"100" split_words:"true"`
	PProf             bool              `default:"false"`

	TunnelDialRetryAttempts         int           `default:"20" split_words:"true"`
	TunnelDialRetryInterval         time.Duration `default:"5s" split_words:"true"`
	TunnelDialRecreateOnTunnelClose bool          `default:"true" split_words:"true"`

	Listen     bool   `default:"true"`
	ListenHost string `default:"" split_words:"true"`
	ListenPort string `default:"8080" split_words:"true"`
}

func (cfg config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(data)
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
	log.Infof("Starting %s with %s", EnvConfigPrefix, cfg)

	if cfg.PProf {
		go func() {
			err := bootstrap.StartPprof()
			log.Fatalf("PProf exited: %s", err)
		}()
	}

	cert := fmt.Sprintf("%s/guardian.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/guardian.key", cfg.CertPath)
	serverCrt := fmt.Sprintf("%s/voltron.crt", cfg.CertPath)
	log.Infof("Voltron Address: %s", cfg.VoltronURL)

	pemCert, err := ioutil.ReadFile(cert)
	if err != nil {
		log.Fatalf("Failed to load cert: %s", err)
	}
	pemKey, err := ioutil.ReadFile(key)
	if err != nil {
		log.Fatalf("Failed to load key: %s", err)
	}

	ca := x509.NewCertPool()
	content, _ := ioutil.ReadFile(serverCrt)
	if ok := ca.AppendCertsFromPEM(content); !ok {
		log.Fatalf("Cannot append the certificate to ca pool: %s", err)
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
		client.WithTunnelDialRetryAttempts(cfg.TunnelDialRetryAttempts),
		client.WithTunnelDialRetryInterval(cfg.TunnelDialRetryInterval),
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

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := client.ServeTunnelHTTP(); err != nil {
			log.WithError(err).Fatal("Serving the tunnel exited with an error")
		}
	}()

	if cfg.Listen {
		log.Infof("Listening on %s:%s for connections to proxy to voltron", cfg.ListenHost, cfg.ListenPort)

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.ListenHost, cfg.ListenPort))
		if err != nil {
			log.WithError(err).Fatalf("Failed to listen on %s:%s", cfg.ListenHost, cfg.ListenPort)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := client.AcceptAndProxy(listener); err != nil {
				log.WithError(err).Error("AcceptAndProxy returned with an error")
			}
		}()
	}

	wg.Wait()
}
