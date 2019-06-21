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
type Config struct {
	Port         int `default:"5555"`
	Host         string
	Tunnel_Port  int `default:"5566"`
	Tunnel_Host  string
	LogLevel     string `default:"DEBUG"`
	CertPath     string `default:"certs"`
	TemplatePath string `default:"/tmp/guardian.yaml"`
	PublicIP     string `default:"127.0.0.1:32453"`
}

func main() {
	cfg := Config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %v", EnvConfigPrefix, cfg)

	cert := fmt.Sprintf("%s/ca.crt", cfg.CertPath)
	key := fmt.Sprintf("%s/ca.key", cfg.CertPath)

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

	lisTun, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Tunnel_Host, cfg.Tunnel_Port))
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
