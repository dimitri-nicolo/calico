// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package main

import (
	"fmt"
	"net"

	"github.com/tigera/voltron/internal/pkg/utils"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/voltron/internal/pkg/bootstrap"
	"github.com/tigera/voltron/internal/pkg/proxy"
	"github.com/tigera/voltron/internal/pkg/server"
)

const (
	// EnvConfigPrefix represents the prefix used to load ENV variables required for startup
	EnvConfigPrefix = "VOLTRON"
)

// Config is a configuration used for Voltron
type config struct {
	Port              int `default:"5555"`
	Host              string
	TunnelPort        int    `default:"5566" split_words:"true"`
	TunnelHost        string `split_words:"true"`
	TunnelCert        string `default:"/certs/tunnel/cert" split_words:"true"`
	TunnelKey         string `default:"/certs/tunnel/key" split_words:"true"`
	LogLevel          string `default:"DEBUG"`
	TemplatePath      string `default:"/tmp/guardian.yaml.tmpl" split_words:"true"`
	PublicIP          string `default:"127.0.0.1:32453" split_words:"true"`
	HTTPSCert         string `default:"/certs/https/cert" split_words:"true"`
	HTTPSKey          string `default:"/certs/https/key" split_words:"true"`
	K8sConfigPath     string `split_words:"true"`
	KeepAliveEnable   bool   `default:"true" split_words:"true"`
	KeepAliveInterval int    `default:"100" split_words:"true"`
	DefaultK8sProxy   bool   `default:"true" split_words:"true"`
	DefaultK8sDest    string `default:"https://kubernetes.default" split_words:"true"`
	PProf             bool   `default:"false"`
}

func main() {
	cfg := config{}
	if err := envconfig.Process(EnvConfigPrefix, &cfg); err != nil {
		log.Fatal(err)
	}

	bootstrap.ConfigureLogging(cfg.LogLevel)
	log.Infof("Starting %s with configuration %+v", EnvConfigPrefix, cfg)

	if cfg.PProf {
		go func() {
			err := bootstrap.StartPprof()
			log.Fatalf("PProf exited: %s", err)
		}()
	}

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	pemCert, err := utils.LoadPEMFromFile(cfg.TunnelCert)
	if err != nil {
		log.WithError(err).Fatal("couldn't load tunnel cert from file")
	}

	pemKey, err := utils.LoadPEMFromFile(cfg.TunnelKey)
	if err != nil {
		log.WithError(err).Fatal("couldn't load tunnel key from file")
	}

	tunnelX509Cert, tunnelX509Key, err := utils.LoadX509KeyPairFromPEM(pemCert, pemKey)
	if err != nil {
		log.WithError(err).Fatal("couldn't load tunnel X509 key pair")
	}

	k8s, config := bootstrap.ConfigureK8sClient(cfg.K8sConfigPath)
	opts := []server.Option{
		server.WithDefaultAddr(addr),
		server.WithKeepAliveSettings(cfg.KeepAliveEnable, cfg.KeepAliveInterval),
		server.WithCredsFiles(cfg.HTTPSCert, cfg.HTTPSKey),
		server.WithTemplate(cfg.TemplatePath),
		server.WithPublicAddr(cfg.PublicIP),
		server.WithKeepClusterKeys(),
		server.WithTunnelCreds(tunnelX509Cert, tunnelX509Key),
		server.WithAuthentication(config),

		// TODO: remove when voltron starts using k8s resources, probably by SAAS-178
		server.WithAutoRegister(),
	}

	if cfg.DefaultK8sProxy {
		tgts, err := bootstrap.ProxyTargets([]bootstrap.Target{
			{
				Path:         "/api/",
				Dest:         cfg.DefaultK8sDest,
				CABundlePath: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			},
			{
				Path:         "/apis/",
				Dest:         cfg.DefaultK8sDest,
				CABundlePath: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
			},
			// This fixes https://tigera.atlassian.net/browse/SAAS-240
			{
				Path:        "/tigera-elasticsearch/",
				Dest:        "https://cnx-es-proxy-local.calico-monitoring.svc.cluster.local:8443",
				PathRegexp:  []byte("^/tigera-elasticsearch/?"),
				PathReplace: []byte("/"),
			},
		})

		if err != nil {
			log.Fatalf("Failed to parse default proxy targets: %s", err)
		}

		defaultK8sProxy, err := proxy.New(tgts)
		if err != nil {
			log.Fatalf("Failed to create a default k8s proxy: %s", err)
		}
		opts = append(opts, server.WithDefaultProxy(defaultK8sProxy))
	}

	srv, err := server.New(
		k8s,
		opts...,
	)

	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	lisTun, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.TunnelHost, cfg.TunnelPort))
	if err != nil {
		log.Fatalf("Failed to create tunnel listener: %s", err)
	}

	go func() {
		err := srv.ServeTunnelsTLS(lisTun)
		log.Fatalf("Tunnel server exited: %s", err)
	}()

	go func() {
		err := srv.WatchK8s()
		log.Fatalf("K8s watcher exited: %s", err)
	}()

	log.Infof("Voltron listens for tunnels at %s", lisTun.Addr().String())

	log.Infof("Voltron listens for HTTP request at %s", addr)
	if err := srv.ListenAndServeHTTPS(); err != nil {
		log.Fatal(err)
	}
}
