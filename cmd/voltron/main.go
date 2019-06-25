// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"net"

	"github.com/tigera/voltron/internal/pkg/utils"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/server"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "VOLTRON"
)

// Config is a configuration used for Voltron
type config struct {
	Port         int `default:"5555"`
	Host         string
	TunnelPort   int    `default:"5566" split_words:"true"`
	TunnelHost   string `split_words:"true"`
	LogLevel     string `default:"DEBUG"`
	CertPath     string `default:"/certs" split_words:"true"`
	TemplatePath string `default:"/tmp/guardian.yaml.tmpl" split_words:"true"`
	PublicIP     string `default:"127.0.0.1:32453" split_words:"true"`
}

func main() {
	cfg := config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %+v", EnvConfigPrefix, cfg)

	cert := fmt.Sprintf("%s/voltron.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/voltron.key", cfg.CertPath)

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	//TODO: These should be different that the ones baked in
	pemCert, _ := utils.LoadPEMFromFile(cert)
	pemKey, _ := utils.LoadPEMFromFile(key)
	tunnelCert, tunnelKey, _ := utils.LoadX509KeyPairFromPEM(pemCert, pemKey)

	srv, err := server.New(
		server.WithDefaultAddr(addr),
		server.WithCredsFiles(cert, key),
		server.WithTemplate(cfg.TemplatePath),
		server.WithPublicAddr(cfg.PublicIP),
		server.WithKeepClusterKeys(),
		server.WithTunnelCreds(tunnelCert, tunnelKey),
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	lisTun, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.TunnelHost, cfg.TunnelPort))
	if err != nil {
		log.Fatalf("Failedto create tunnel listener: %s", err)
	}
	err = srv.ServeTunnelsTLS(lisTun)
	if err != nil {
		log.Fatalf("Tunnel server did not start: %s", err)
	}
	log.Infof("Voltron listens for tunnels at %s", lisTun.Addr().String())

	log.Infof("Voltron listens for HTTP request at %s", addr)
	if err := srv.ListenAndServeHTTPS(); err != nil {
		log.Fatal(err)
	}
}
