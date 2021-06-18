package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-gateway/pkg/clients/elastic"
	"github.com/tigera/es-gateway/pkg/clients/kibana"
	"github.com/tigera/es-gateway/pkg/clients/kubernetes"
	"github.com/tigera/es-gateway/pkg/config"
	"github.com/tigera/es-gateway/pkg/proxy"
	"github.com/tigera/es-gateway/pkg/server"
	"github.com/tigera/es-gateway/pkg/version"
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
		log.WithError(err).Warn("failed to get system cert pool, creating a new one")
		log.Fatal(err)
	}

	// Setup logging. Default to WARN log level.
	cfg.SetupLogging()

	log.Infof("Starting %s with %s", config.EnvConfigPrefix, cfg)

	if len(cfg.ElasticCatchAllRoute) > 0 {
		// Catch-all route should ...
		elasticCatchAllRoute = &proxy.Route{
			Name:         "es-catch-all",
			Path:         cfg.ElasticCatchAllRoute,
			IsPathPrefix: true,       // ... always be a prefix route.
			HTTPMethods:  []string{}, // ... not filter on HTTP methods.
			RequireAuth:  true,
		}
	}

	if len(cfg.KibanaCatchAllRoute) > 0 {
		// Catch-all route should ...
		kibanaCatchAllRoute = &proxy.Route{
			Name:         "kb-catch-all",
			Path:         cfg.KibanaCatchAllRoute,
			IsPathPrefix: true,       // ... always be a prefix route.
			HTTPMethods:  []string{}, // ... not filter on HTTP methods.
			RequireAuth:  true,
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
		false,
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
		false,
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
		server.WithInternalTLSFiles(cfg.HTTPSCert, cfg.HTTPSKey),
		server.WithESClient(esClient),
		server.WithKibanaClient(kbClient),
		server.WithK8sClient(k8sClient),
	}

	srv, err := server.New(esTarget, kibanaTarget, opts...)
	if err != nil {
		log.WithError(err).Fatal("failed to create ES Gateway server.")
	}

	log.Infof("ES Gateway listening for HTTPS requests at %s", addr)
	log.Fatal(srv.ListenAndServeHTTPS())
}
