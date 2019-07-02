// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/client"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/utils"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "GUARDIAN"
)

type target struct {
	// Path is the path portion of the URL based on which we proxy
	Path string `json:"path"`
	// Dest is the destination URL
	Dest string `json:"url"`
	// TokenPath is where we read the Bearer token from (if non-empty)
	TokenPath string `json:"tokenPath,omitempty"`
	// CABundlePath is where we read the CA bundle from to authenticate the
	// destination (if non-empty)
	CABundlePath string `json:"caBundlePath,omitempty"`
}

type proxyTarget []target

// Decode deserializes the list of proxytargets
func (pt *proxyTarget) Decode(envVar string) error {
	err := json.Unmarshal([]byte(envVar), pt)
	if err != nil {
		return err
	}

	return nil
}

// Config is a configuration used for Guardian
type config struct {
	// until health check restored
	//Port       int    `default:"5555"`
	//Host       string `default:"localhost"`
	LogLevel     string      `default:"DEBUG"`
	CertPath     string      `default:"/certs" split_words:"true"`
	VoltronURL   string      `required:"true" split_words:"true"`
	ProxyTargets proxyTarget `required:"true" split_words:"true"`
}

func fillTargets(tgts proxyTarget) ([]proxy.Target, error) {
	var ret []proxy.Target

	for _, t := range tgts {
		pt := proxy.Target{
			Path: t.Path,
		}

		var err error
		pt.Dest, err = url.Parse(t.Dest)
		if err != nil {
			return nil, errors.Errorf("Incorrect URL %q for path %q: %s", t.Dest, t.Path, err)
		}

		if t.TokenPath != "" {
			token, err := ioutil.ReadFile(t.TokenPath)
			if err != nil {
				return nil, errors.Errorf("Failed reading token from %s: %s", t.TokenPath, err)
			}

			pt.Token = string(token)
		}

		if t.CABundlePath != "" {
			pt.CA, err = utils.LoadX509FromFile(t.CABundlePath)
			if err != nil {
				return nil, errors.WithMessage(err, "LoadX509FromFile")
			}
		}

		ret = append(ret, pt)
	}

	return ret, nil
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

	tgts, err := fillTargets(cfg.ProxyTargets)
	if err != nil {
		log.Fatalf("Failed to fill targets: %s", err)
	}

	client, err := client.New(
		cfg.VoltronURL,
		client.WithProxyTargets(tgts),
		client.WithTunnelCreds(pemCert, pemKey, ca),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	if err := client.ServeTunnelHTTP(); err != nil {
		log.Fatalf("Tunnel exited with error: %s", err)
	}
}
