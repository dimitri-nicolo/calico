// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/projectcalico/calico/es-gateway/pkg/metrics"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/es-gateway/pkg/clients/elastic"
	"github.com/projectcalico/calico/es-gateway/pkg/clients/kibana"
	"github.com/projectcalico/calico/es-gateway/pkg/clients/kubernetes"
	"github.com/projectcalico/calico/es-gateway/pkg/config"
	"github.com/projectcalico/calico/es-gateway/pkg/proxy"
	"github.com/projectcalico/calico/es-gateway/pkg/server"
	"github.com/projectcalico/calico/es-gateway/pkg/version"
)

var (
	versionFlag = flag.Bool("version", false, "Print version information")

	// Configuration object for ES Gateway server.
	cfg *config.Config

	// Catch-all Route for Elasticsearch and Kibana.
	elasticCatchAllRoute, kibanaCatchAllRoute *proxy.Route
)

// Initialize ES Gateway configuration.
func init() {
	// Parse all command-line flags.
	flag.Parse()

	// For --version use case (display version information and exit program).
	if *versionFlag {
		version.Version()
		os.Exit(0)
	}

	cfg = &config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, cfg); err != nil {
		log.WithError(err).Warn("failed to initialize environment variables")
		log.Fatal(err)
	}

	// Setup logging. Default to WARN log level.
	cfg.SetupLogging()

	// Print out configuration (minus sensitive fields)
	printCfg := &config.Config{}
	*printCfg = *cfg
	printCfg.ElasticPassword = "" // wipe out sensitive field

	log.Infof("Starting %s with %s", config.EnvConfigPrefix, printCfg)

	if len(cfg.ElasticCatchAllRoute) > 0 {
		// Catch-all route should ...
		elasticCatchAllRoute = &proxy.Route{
			Name:                          "es-catch-all",
			Path:                          cfg.ElasticCatchAllRoute,
			IsPathPrefix:                  true,       // ... always be a prefix route.
			HTTPMethods:                   []string{}, // ... not filter on HTTP methods.
			RequireAuth:                   true,
			RejectUnacceptableContentType: true,
		}
	}

	if len(cfg.KibanaCatchAllRoute) > 0 {
		// Catch-all route should ...
		kibanaCatchAllRoute = &proxy.Route{
			Name:         "kb-catch-all",
			Path:         cfg.KibanaCatchAllRoute,
			IsPathPrefix: true,       // ... always be a prefix route.
			HTTPMethods:  []string{}, // ... not filter on HTTP methods.
			RequireAuth:  false,
		}
	}

	if len(cfg.ElasticUsername) == 0 || len(cfg.ElasticPassword) == 0 {
		log.Fatal("Elastic credentials cannot be empty")
	}

	if len(cfg.ElasticEndpoint) == 0 {
		log.Fatal("Elastic endpoint cannot be empty")
	}

	if len(cfg.KibanaEndpoint) == 0 {
		log.Fatal("Kibana endpoint cannot be empty")
	}
}

// Start up HTTPS server for ES Gateway.
func main() {
	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	// Create Kibana target that will be used to configure all routing to Kibana target.
	kibanaTarget, err := proxy.CreateTarget(
		kibanaCatchAllRoute,
		config.KibanaRoutes,
		cfg.KibanaEndpoint,
		cfg.KibanaCABundlePath,
		cfg.KibanaClientCertPath,
		cfg.KibanaClientKeyPath,
		cfg.EnableKibanaMutualTLS,
		false,
		cfg.FIPSModeEnabled,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to configure Kibana target for ES Gateway.")
	}

	// Create Elasticsearch target that will be used to configure all routing to ES target.
	esTarget, err := proxy.CreateTarget(
		elasticCatchAllRoute,
		config.ElasticsearchRoutes,
		cfg.ElasticEndpoint,
		cfg.ElasticCABundlePath,
		cfg.ElasticClientCertPath,
		cfg.ElasticClientKeyPath,
		cfg.EnableElasticMutualTLS,
		false,
		cfg.FIPSModeEnabled,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to configure ES target for ES Gateway.")
	}

	// Create client for Elasticsearch API calls.
	esClient, err := elastic.NewClient(
		cfg.ElasticEndpoint,
		cfg.ElasticUsername,
		cfg.ElasticPassword,
		cfg.ElasticCABundlePath,
		cfg.ElasticClientCertPath,
		cfg.ElasticClientKeyPath,
		cfg.EnableElasticMutualTLS,
		cfg.FIPSModeEnabled,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to configure ES client for ES Gateway.")
	}

	// Create client for Kibana API calls.
	kbClient, err := kibana.NewClient(
		cfg.KibanaEndpoint,
		cfg.ElasticUsername,
		cfg.ElasticPassword,
		cfg.KibanaCABundlePath,
		cfg.KibanaClientCertPath,
		cfg.KibanaClientKeyPath,
		cfg.EnableKibanaMutualTLS,
		cfg.FIPSModeEnabled,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to configure Kibana client for ES Gateway.")
	}

	// Create client for Kube API calls.
	k8sClient, err := kubernetes.NewClient(cfg.K8sConfigPath)
	if err != nil {
		log.WithError(err).Fatal("failed to configure Kibana client for ES Gateway.")
	}

	opts := []server.Option{
		server.WithAddr(addr),
		server.WithESTarget(esTarget),
		server.WithKibanaTarget(kibanaTarget),
		server.WithInternalTLSFiles(cfg.HTTPSCert, cfg.HTTPSKey),
		server.WithESClient(esClient),
		server.WithKibanaClient(kbClient),
		server.WithK8sClient(k8sClient),
		server.WithAdminUser(cfg.ElasticUsername, cfg.ElasticPassword),
		server.WithFIPSModeEnabled(cfg.FIPSModeEnabled),
	}

	var collector metrics.Collector

	if cfg.MetricsEnabled {
		log.Debugf("starting a metrics server on port %v", cfg.MetricsPort)
		collector, err = metrics.NewCollector()
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, server.WithCollector(collector))
	}

	srv, err := server.New(opts...)
	if err != nil {
		log.WithError(err).Fatal("failed to create ES Gateway server.")
	}

	if cfg.MetricsEnabled {
		metricsAddr := fmt.Sprintf("%v:%v", cfg.Host, cfg.MetricsPort)
		go func() {
			log.Infof("ES Gateway listening for metrics requests at %s", metricsAddr)
			log.Fatal(collector.Serve(metricsAddr))
		}()
	}

	log.Infof("ES Gateway listening for HTTPS requests at %s", addr)
	log.Fatal(srv.ListenAndServeHTTPS())
}
