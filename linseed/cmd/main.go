// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/calico/libcalico-go/lib/health"
	"github.com/projectcalico/calico/linseed/pkg/controller/token"
	"github.com/projectcalico/calico/linseed/pkg/handler"
	"github.com/projectcalico/calico/linseed/pkg/handler/audit"
	"github.com/projectcalico/calico/linseed/pkg/handler/bgp"
	"github.com/projectcalico/calico/linseed/pkg/handler/compliance"
	"github.com/projectcalico/calico/linseed/pkg/handler/dns"
	"github.com/projectcalico/calico/linseed/pkg/handler/events"
	"github.com/projectcalico/calico/linseed/pkg/handler/l3"
	"github.com/projectcalico/calico/linseed/pkg/handler/l7"
	"github.com/projectcalico/calico/linseed/pkg/handler/processes"
	"github.com/projectcalico/calico/linseed/pkg/handler/runtime"
	"github.com/projectcalico/calico/linseed/pkg/handler/waf"
	"github.com/projectcalico/calico/linseed/pkg/middleware"
	"github.com/projectcalico/calico/lma/pkg/auth"

	"github.com/projectcalico/calico/linseed/pkg/backend"

	rest "k8s.io/client-go/rest"

	auditbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/audit"
	bgpbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/bgp"
	compliancebackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/compliance"
	dnsbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/dns"
	eventbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/events"
	flowbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	l7backend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
	procbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/processes"
	runtimebackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/runtime"
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

	esClient := backend.MustGetElasticClient(cfg)

	// Create template caches for indices with special shards / replicas configuration
	defaultCache := templates.NewTemplateCache(esClient, cfg.ElasticShards, cfg.ElasticReplicas)
	flowCache := templates.NewTemplateCache(esClient, cfg.ElasticFlowShards, cfg.ElasticFlowReplicas)
	dnsCache := templates.NewTemplateCache(esClient, cfg.ElasticDNSShards, cfg.ElasticDNSReplicas)
	l7Cache := templates.NewTemplateCache(esClient, cfg.ElasticL7Shards, cfg.ElasticL7Replicas)
	auditCache := templates.NewTemplateCache(esClient, cfg.ElasticAuditShards, cfg.ElasticAuditReplicas)
	bgpCache := templates.NewTemplateCache(esClient, cfg.ElasticBGPShards, cfg.ElasticBGPReplicas)

	// Create all the necessary backends.
	flowLogBackend := flowbackend.NewFlowLogBackend(esClient, flowCache)
	eventBackend := eventbackend.NewBackend(esClient, defaultCache)
	flowBackend := flowbackend.NewFlowBackend(esClient)
	dnsFlowBackend := dnsbackend.NewDNSFlowBackend(esClient)
	dnsLogBackend := dnsbackend.NewDNSLogBackend(esClient, dnsCache)
	l7FlowBackend := l7backend.NewL7FlowBackend(esClient)
	l7LogBackend := l7backend.NewL7LogBackend(esClient, l7Cache)
	auditBackend := auditbackend.NewBackend(esClient, auditCache)
	bgpBackend := bgpbackend.NewBackend(esClient, bgpCache)
	procBackend := procbackend.NewBackend(esClient)
	wafBackend := wafbackend.NewBackend(esClient, defaultCache)
	reportsBackend := compliancebackend.NewReportsBackend(esClient, defaultCache)
	snapshotsBackend := compliancebackend.NewSnapshotBackend(esClient, defaultCache)
	benchmarksBackend := compliancebackend.NewBenchmarksBackend(esClient, defaultCache)
	runtimeBackend := runtimebackend.NewBackend(esClient, defaultCache)

	// Create a Kuberentes client to use for authorization.
	var kc *rest.Config
	var err error
	if cfg.Kubeconfig == "" {
		// creates the in-cluster k8sConfig
		kc, err = rest.InClusterConfig()
	} else {
		// creates a k8sConfig from supplied kubeconfig
		kc, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	}
	if err != nil {
		logrus.WithError(err).Fatal("Unable to load Kubernetes config")
	}

	// We can only perform authentication / authorization if our Kubernetes configuration
	// has a bearer token present.
	k, err := kubernetes.NewForConfig(kc)
	if err != nil {
		logrus.WithError(err).Fatal("Unable to create Kubernetes client")
	}
	pc, err := clientset.NewForConfig(kc)
	if err != nil {
		logrus.WithError(err).Fatal("Unable to create Calico client")
	}

	authOpts := []auth.JWTAuthOption{}
	if cfg.TokenControllerEnabled {
		// Get our token signing key.
		key, err := tokenCredentials(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("Unable to acquire token signing key")
		}

		// Build a token controller to generate tokens for Linseed clients
		// in managed clusters. We'll create tokens in each managed cluster for the following
		// service account users.
		//
		// Each client that connects from a managed cluster will provide these tokens, which will map
		// back to the permissions assigned to its serviceaccount in the management cluster.
		users := []token.UserInfo{
			{Namespace: "tigera-fluentd", Name: "fluentd-node"},
			{Namespace: "tigera-compliance", Name: "tigera-compliance-benchmarker"},
			{Namespace: "tigera-compliance", Name: "tigera-compliance-controller"},
			{Namespace: "tigera-compliance", Name: "tigera-compliance-reporter"},
			{Namespace: "tigera-compliance", Name: "tigera-compliance-snapshotter"},
			{Namespace: "tigera-dpi", Name: "tigera-dpi"},
			{Namespace: "tigera-intrusion-detection", Name: "intrusion-detection-controller"},
		}
		opts := []token.ControllerOption{
			token.WithIssuer(token.LinseedIssuer),
			token.WithIssuerName("tigera-linseed"),
			token.WithUserInfos(users),
			token.WithExpiry(24 * time.Hour),
			token.WithClient(pc),
			token.WithPrivateKey(key),
			token.WithMultiClusterEndpoint(cfg.MultiClusterForwardingEndpoint, cfg.MultiClusterForwardingCA),
			token.WithTenant(cfg.ExpectedTenantID),
		}
		tokenController, err := token.NewController(opts...)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to start token controller")
		}

		// Start the token controller.
		stop := make(chan struct{})
		defer close(stop)
		go tokenController.Run(stop)

		// Add an authenticator for JWTs issued by this tenant's Linseed.
		lsa := auth.NewLocalAuthenticator(token.LinseedIssuer, key.Public(), token.ParseClaimsLinseed)
		authOpts = append(authOpts, auth.WithAuthenticator(token.LinseedIssuer, lsa))
	}

	// Create a JWT authenticator for Linseed to use.
	authn, err := auth.NewJWTAuth(kc, k, authOpts...)
	if err != nil {
		logrus.WithError(err).Fatal("Unable to create authenticator")
	}

	// Create an RBAC authorizer to use for authorizing requests.
	authz := auth.NewRBACAuthorizer(k)
	authzHelper := middleware.NewKubernetesAuthzTracker(authz)

	// Create the full list of handlers, and register them
	// for authorization.
	handlers := []handler.Handler{
		l3.New(flowBackend, flowLogBackend),
		l7.New(l7FlowBackend, l7LogBackend),
		dns.New(dnsFlowBackend, dnsLogBackend),
		events.New(eventBackend),
		audit.New(auditBackend),
		bgp.New(bgpBackend),
		processes.New(procBackend),
		waf.New(wafBackend),
		compliance.New(benchmarksBackend, snapshotsBackend, reportsBackend),
		runtime.New(runtimeBackend),
	}

	// Configure options used to launch the server.
	opts := []server.Option{
		server.WithMiddlewares(server.Middlewares(cfg, authn, authzHelper)),
		server.WithAPIVersionRoutes("/api/v1", server.UnpackRoutes(handlers...)...),
		server.WithRoutes(server.UtilityRoutes()...),
	}
	if cfg.CACert != "" {
		opts = append(opts, server.WithClientCACerts(cfg.CACert))
	}

	// Make sure we register our APIs for authorization.
	for _, h := range handlers {
		for _, api := range h.APIS() {
			if api.AuthzAttributes != nil {
				authzHelper.Register(api.Method, api.URL, api.AuthzAttributes)
			}
		}
	}

	// Register the /version endpoint without authorization.
	authzHelper.Disable("GET", "/version")

	// Start the server.
	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)
	server := server.NewServer(addr, cfg.FIPSModeEnabled, opts...)

	go func() {
		logrus.Infof("Listening for HTTPS requests at %s", addr)
		if err := server.ListenAndServeTLS(cfg.HTTPSCert, cfg.HTTPSKey); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to listen for new requests for Linseed APIs")
		}
	}()

	go func() {
		// We only want the health aggregator to be accessible from within the container.
		// Kubelet will use an exec probe to get status.
		healthAggregator.ServeHTTP(true, "localhost", 8080)
	}()

	if cfg.EnableMetrics {
		go func() {
			metricsAddr := fmt.Sprintf("%v:%v", cfg.Host, cfg.MetricsPort)
			http.Handle("/metrics", promhttp.Handler())
			err := http.ListenAndServeTLS(metricsAddr, cfg.MetricsCert, cfg.MetricsKey, nil)
			if err != nil {
				logrus.WithError(err).Fatal("Failed to listen for new requests to query metrics")
			}
		}()
	}

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

func tokenCredentials(cfg config.Config) (*rsa.PrivateKey, error) {
	// Load the signing key.
	bs, err := os.ReadFile(cfg.TokenKey)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(bs)
	if p == nil {
		return nil, fmt.Errorf("failed to decode token signing key")
	}
	return x509.ParsePKCS1PrivateKey(p.Bytes)
}
