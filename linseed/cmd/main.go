// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/linseed/pkg/handler/audit"
	"github.com/projectcalico/calico/linseed/pkg/handler/bgp"
	"github.com/projectcalico/calico/linseed/pkg/handler/dns"
	"github.com/projectcalico/calico/linseed/pkg/handler/events"
	"github.com/projectcalico/calico/linseed/pkg/handler/l3"
	"github.com/projectcalico/calico/linseed/pkg/handler/l7"
	"github.com/projectcalico/calico/linseed/pkg/handler/processes"
	"github.com/projectcalico/calico/linseed/pkg/handler/waf"

	"github.com/projectcalico/calico/linseed/pkg/backend"

	auditbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/audit"
	bgpbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/bgp"
	dnsbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/dns"
	eventbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/events"
	flowbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	l7backend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
	procbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/processes"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	wafbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/waf"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/config"
	"github.com/projectcalico/calico/linseed/pkg/server"
)

var (
	ready bool
	live  bool
)

func init() {
	flag.BoolVar(&ready, "ready", false, "Set to get readiness information")
	flag.BoolVar(&live, "live", false, "Set to get liveness information")
}

func main() {
	flag.Parse()

	if ready {
		doHealthCheck("readiness")
	} else if live {
		doHealthCheck("liveness")
	} else {
		// Just run the server.
		run()
	}
}

func run() {
	// Read and reconcile configuration
	cfg := config.Config{}
	if err := envconfig.Process(config.EnvConfigPrefix, &cfg); err != nil {
		panic(err)
	}

	// Configure logging
	config.ConfigureLogging(cfg.LogLevel)
	logrus.Debugf("Starting with %#v", cfg)

	// Register for termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a health aggregator and mark us as alive.
	// For now, we don't do periodic updates to our health, so don't set a timeout.
	const healthName = "Startup"
	healthAggregator := health.NewHealthAggregator()
	healthAggregator.RegisterReporter(healthName, &health.HealthReport{Live: true}, 0)

	// TODO: check if we need to add es connection as part of the ready probe
	esClient := backend.MustGetElasticClient(toElasticConfig(cfg))
	cache := templates.NewTemplateCache(esClient, cfg.ElasticShards, cfg.ElasticReplicas)

	// Create all the necessary backends.
	flowLogBackend := flowbackend.NewFlowLogBackend(esClient, cache)
	eventBackend := eventbackend.NewBackend(esClient, cache)
	flowBackend := flowbackend.NewFlowBackend(esClient)
	dnsFlowBackend := dnsbackend.NewDNSFlowBackend(esClient)
	dnsLogBackend := dnsbackend.NewDNSLogBackend(esClient, cache)
	l7FlowBackend := l7backend.NewL7FlowBackend(esClient)
	l7LogBackend := l7backend.NewL7LogBackend(esClient, cache)
	auditBackend := auditbackend.NewBackend(esClient, cache)
	bgpBackend := bgpbackend.NewBackend(esClient, cache)
	procBackend := procbackend.NewBackend(esClient)
	wafBackend := wafbackend.NewBackend(esClient, cache)

	// Configure options used to launch the server.
	opts := []server.Option{
		server.WithMiddlewares(server.Middlewares(cfg)),
		server.WithAPIVersionRoutes("/api/v1", server.UnpackRoutes(
			l3.New(flowBackend, flowLogBackend),
			l7.New(l7FlowBackend, l7LogBackend),
			dns.New(dnsFlowBackend, dnsLogBackend),
			events.New(eventBackend),
			audit.New(auditBackend),
			bgp.New(bgpBackend),
			processes.New(procBackend),
			waf.New(wafBackend),
		)...),
		server.WithRoutes(server.UtilityRoutes()...),
	}

	if cfg.CACert != "" {
		opts = append(opts, server.WithClientCACerts(cfg.CACert))
	}

	// Start server, adding in handlers for the various API endpoints.
	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	server := server.NewServer(addr, cfg.FIPSModeEnabled, opts...)

	go func() {
		logrus.Infof("Listening for HTTPS requests at %s", addr)
		if err := server.ListenAndServeTLS(cfg.HTTPSCert, cfg.HTTPSKey); err != nil && err != http.ErrServerClosed {
			logrus.Fatal(err)
		}
	}()

	go func() {
		// We only want the health aggregator to be accessible from within the container.
		// Kubelet will use an exec probe to get status.
		healthAggregator.ServeHTTP(true, "localhost", 8080)
	}()

	// Indicate that we're ready to serve requests.
	healthAggregator.Report(healthName, &health.HealthReport{Live: true, Ready: true})

	// Listen for termination signals
	sig := <-signalChan
	logrus.WithField("signal", sig).Info("Received shutdown signal")

	// Graceful shutdown of the server
	shutDownCtx, shutDownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutDownCancel()
	if err := server.Shutdown(shutDownCtx); err != nil {
		logrus.Fatalf("server shutdown failed: %+v", err)
	}
	logrus.Info("Server is shutting down")
}

func toElasticConfig(cfg config.Config) backend.ElasticConfig {
	if cfg.ElasticUsername == "" || cfg.ElasticPassword == "" {
		logrus.Warn("No credentials were passed in for Elastic. Will connect to ES without credentials")

		return backend.ElasticConfig{
			URL:             cfg.ElasticEndpoint,
			LogLevel:        cfg.LogLevel,
			FIPSModeEnabled: cfg.FIPSModeEnabled,
			GZIPEnabled:     cfg.ElasticGZIPEnabled,
			Scheme:          cfg.ElasticScheme,
			EnableSniffing:  cfg.ElasticSniffingEnabled,
		}
	}

	return backend.ElasticConfig{
		URL:               cfg.ElasticEndpoint,
		Username:          cfg.ElasticUsername,
		Password:          cfg.ElasticPassword,
		LogLevel:          cfg.LogLevel,
		CACertPath:        cfg.ElasticCABundlePath,
		ClientCertPath:    cfg.ElasticClientCertPath,
		ClientCertKeyPath: cfg.ElasticClientKeyPath,
		FIPSModeEnabled:   cfg.FIPSModeEnabled,
		GZIPEnabled:       cfg.ElasticGZIPEnabled,
		Scheme:            cfg.ElasticScheme,
	}
}

// doHealthCheck checks the local readiness or liveness endpoint and prints its status.
// It exits with a status code based on the status.
func doHealthCheck(path string) {
	url := fmt.Sprintf("http://localhost:8080/%s", path)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("failed to build request: %s\n", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to check %s: %s\n", path, err)
		os.Exit(1)
	}
	if resp.StatusCode == http.StatusOK {
		os.Exit(0)
	} else {
		fmt.Printf("bad status code (%d) from %s endpoint\n", resp.StatusCode, path)
		os.Exit(1)
	}
}
