// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/kibana"
	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/middleware/aggregation"
	"github.com/tigera/es-proxy/pkg/middleware/event"
	"github.com/tigera/es-proxy/pkg/middleware/rawquery"
	"github.com/tigera/es-proxy/pkg/middleware/search"
	"github.com/tigera/es-proxy/pkg/middleware/servicegraph"
	"github.com/tigera/es-proxy/pkg/pip"
	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
	lmaauth "github.com/tigera/lma/pkg/auth"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	lmaindex "github.com/tigera/lma/pkg/elastic/index"
	"github.com/tigera/lma/pkg/k8s"
	"github.com/tigera/lma/pkg/list"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

// Some constants that we pass to our elastic client library.
// We don't need to define these because we don't ever create
// indices or index patterns from here and these parameters are
// only required in such cases.
const (
	voltronServiceURL = "https://localhost:9443"

	dnsLogsResourceName  = "dns"
	eventsResourceName   = "events"
	flowLogsResourceName = "flows"
	l7ResourceName       = "l7"
)

func Start(cfg *Config) error {
	sm := http.NewServeMux()

	var authn lmaauth.JWTAuth
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Unable to create client config", err)
	}
	k8sCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal("Unable to create kubernetes interface", err)
	}

	var options []lmaauth.JWTAuthOption
	if cfg.OIDCAuthEnabled {
		log.Debug("Configuring Dex for authentication")
		opts := []lmaauth.DexOption{
			lmaauth.WithGroupsClaim(cfg.OIDCAuthGroupsClaim),
			lmaauth.WithJWKSURL(cfg.OIDCAuthJWKSURL),
			lmaauth.WithUsernamePrefix(cfg.OIDCAuthUsernamePrefix),
			lmaauth.WithGroupsPrefix(cfg.OIDCAuthGroupsPrefix),
		}
		dex, err := lmaauth.NewDexAuthenticator(
			cfg.OIDCAuthIssuer,
			cfg.OIDCAuthClientID,
			cfg.OIDCAuthUsernameClaim,
			opts...)
		if err != nil {
			log.Fatal("Unable to add an issuer to the authenticator", err)
		}
		options = append(options, lmaauth.WithAuthenticator(cfg.OIDCAuthIssuer, dex))
	}
	authn, err = lmaauth.NewJWTAuth(restConfig, k8sCli, options...)
	if err != nil {
		log.Fatal("Unable to create authenticator", err)
	}

	// Install pip mutator
	k8sClientSet := datastore.MustGetClientSet()
	policyCalcConfig := pipcfg.MustLoadConfig()

	// initialize Elastic client factory
	envCfg := lmaelastic.MustLoadConfig()
	esClientFactory := lmaelastic.NewClusterContextClientFactory(envCfg)
	esClient, err := esClientFactory.ClientForCluster(cfg.ElasticIndexSuffix)
	if err != nil {
		log.WithError(err).Error("failed to create Elastic client from factory")
		return err
	}

	k8sClientFactory := datastore.NewClusterCtxK8sClientFactory(restConfig, cfg.VoltronCAPath, voltronServiceURL)
	authz := lmaauth.NewRBACAuthorizer(k8sCli)

	// For handlers that use the newer AuthorizationReview to perform RBAC checks, the k8sClientSetFactory provide
	// cluster and user aware k8s clients.
	k8sClientSetFactory := k8s.NewClientSetFactory(cfg.VoltronCAPath, voltronServiceURL)

	// Create a PIP backend.
	p := pip.New(policyCalcConfig, &clusterAwareLister{k8sClientFactory}, esClient)

	kibanaCli := kibana.NewClient(&http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}, cfg.ElasticKibanaEndpoint)

	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))

	// Service graph has additional RBAC control built in since it accesses multiple different tables. However, the
	// minimum requirements for accessing SG is access to the flow logs. Perform that check here so it is done at the
	// earliest opportunity.
	sm.Handle("/serviceGraph",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					servicegraph.NewServiceGraphHandler(
						context.Background(),
						authz,
						esClient,
						k8sClientSetFactory,
						&servicegraph.Config{
							ServiceGraphCacheMaxEntries:           cfg.ServiceGraphCacheMaxEntries,
							ServiceGraphCacheMaxBucketsPerQuery:   cfg.ServiceGraphCacheMaxBucketsPerQuery,
							ServiceGraphCacheMaxAggregatedRecords: cfg.ServiceGraphCacheMaxAggregatedRecords,
							ServiceGraphCachePolledEntryAgeOut:    cfg.ServiceGraphCachePolledEntryAgeOut,
							ServiceGraphCacheSlowQueryEntryAgeOut: cfg.ServiceGraphCacheSlowQueryEntryAgeOut,
							ServiceGraphCachePollLoopInterval:     cfg.ServiceGraphCachePollLoopInterval,
							ServiceGraphCachePollQueryInterval:    cfg.ServiceGraphCachePollQueryInterval,
							ServiceGraphCacheDataSettleTime:       cfg.ServiceGraphCacheDataSettleTime,
						},
					)))))
	sm.Handle("/flowLogs",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogsHandler(k8sClientFactory, esClient, p)))))
	sm.Handle("/flowLogs/aggregation",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					aggregation.NewAggregationHandler(esClient, k8sClientSetFactory, lmaindex.FlowLogs())))))
	sm.Handle("/flowLogs/search",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						lmaindex.FlowLogs(),
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSet,
						esClient.Backend(),
					)))))
	sm.Handle("/dnsLogs/aggregation",
		middleware.ClusterRequestToResource(dnsLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					aggregation.NewAggregationHandler(esClient, k8sClientSetFactory, lmaindex.DnsLogs())))))
	sm.Handle("/dnsLogs/search",
		middleware.ClusterRequestToResource(dnsLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						lmaindex.DnsLogs(),
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSet,
						esClient.Backend(),
					)))))
	sm.Handle("/l7Logs/aggregation",
		middleware.ClusterRequestToResource(l7ResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					aggregation.NewAggregationHandler(esClient, k8sClientSetFactory, lmaindex.L7Logs())))))
	sm.Handle("/l7Logs/search",
		middleware.ClusterRequestToResource(l7ResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						lmaindex.L7Logs(),
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSet,
						esClient.Backend(),
					)))))
	sm.Handle("/events/bulk",
		middleware.ClusterRequestToResource(eventsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					event.EventHandler(esClientFactory)))))
	sm.Handle("/events/search",
		middleware.ClusterRequestToResource(eventsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						lmaindex.Alerts(),
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSet,
						esClient.Backend(),
					)))))
	// Perform authn using KubernetesAuthn handler, but authz using PolicyRecommendationHandler.
	sm.Handle("/recommend",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.PolicyRecommendationHandler(k8sClientFactory, k8sClientSet, esClient)))))
	sm.Handle("/flowLogNamespaces",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogNamespaceHandler(k8sClientFactory, esClient)))))
	sm.Handle("/flowLogNames",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogNamesHandler(k8sClientFactory, esClient)))))
	sm.Handle("/flow",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.NewFlowHandler(esClient, k8sClientFactory)))))
	sm.Handle("/user",
		middleware.AuthenticateRequest(authn,
			middleware.NewUserHandler(k8sClientSet, cfg.OIDCAuthEnabled, cfg.OIDCAuthIssuer, cfg.ElasticLicenseType)))
	sm.Handle("/kibana/login",
		middleware.AuthenticateRequest(authn,
			middleware.NewKibanaLoginHandler(k8sClientSet, kibanaCli, cfg.OIDCAuthEnabled, cfg.OIDCAuthIssuer,
				middleware.ElasticsearchLicenseType(cfg.ElasticLicenseType))))
	sm.Handle("/.kibana/_search",
		middleware.KibanaIndexPattern(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					rawquery.RawQueryHandler(esClient.Backend())))))
	sm.Handle("/",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					rawquery.RawQueryHandler(esClient.Backend())))))

	server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: middleware.LogRequestHeaders(sm),
	}

	wg.Add(1)
	go func() {
		log.Infof("Starting server on %v", cfg.ListenAddr)
		err := server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.WithError(err).Error("Error when starting server")
		}
		wg.Done()
	}()

	return nil
}

func Wait() {
	wg.Wait()
}

func Stop() {
	if err := server.Shutdown(context.Background()); err != nil {
		log.WithError(err).Error("Error when stopping server")
	}
}

// clusterAwareLister implements the PIP ClusterAwareLister interface. It is simply a wrapper around the
// ClusterCtxK8sClientFactory to instantiate the appropriate client and invoke the List method on that client.
type clusterAwareLister struct {
	k8sClientFactory datastore.ClusterCtxK8sClientFactory
}

func (c *clusterAwareLister) RetrieveList(clusterID string, kind metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	clientset, err := c.k8sClientFactory.ClientSetForCluster(clusterID)
	if err != nil {
		return nil, err
	}

	return clientset.RetrieveList(kind)
}
