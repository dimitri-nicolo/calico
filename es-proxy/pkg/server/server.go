// Copyright (c) 2019-2023 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcalico/calico/compliance/pkg/datastore"
	calicotls "github.com/projectcalico/calico/crypto/pkg/tls"
	"github.com/projectcalico/calico/es-proxy/pkg/handler"
	"github.com/projectcalico/calico/es-proxy/pkg/kibana"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/aggregation"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/application"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/audit"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/event"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/exceptions"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/process"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/search"
	"github.com/projectcalico/calico/es-proxy/pkg/middleware/servicegraph"
	"github.com/projectcalico/calico/es-proxy/pkg/pip"
	pipcfg "github.com/projectcalico/calico/es-proxy/pkg/pip/config"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	lsrest "github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/lma/pkg/httputils"
	"github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/lma/pkg/list"
	queryserverclient "github.com/projectcalico/calico/ts-queryserver/queryserver/client"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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
	dnsLogsResourceName   = "dns"
	eventsResourceName    = "events"
	flowLogsResourceName  = "flows"
	l7ResourceName        = "l7"
	auditLogsResourceName = "audit*"
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

	// Create authenticator and authorizer.
	authn, err = lmaauth.NewJWTAuth(restConfig, k8sCli, options...)
	if err != nil {
		log.Fatal("Unable to create authenticator", err)
	}

	// Create an authorizer to use for lma.tigera.io resources. If a tenant namespace is configured, the authorizer
	// will use LocalSubjectAccessReviews to check access to the tenant namespace. Otherwise, it will use SubjectAccessReviews
	// to check access at the cluster scope.
	authz := lmaauth.NewNamespacedRBACAuthorizer(k8sCli, cfg.TenantNamespace)

	// Create clients for access the management cluster apiserver.
	k8sClientSet := datastore.MustGetClientSet()

	// We use a controller-runtime client for accessing ManagedCluster resources because unlike the ClientSet, it functions in both Enterprise and
	// multi-tenant environments where the ManagedCluster CRD may be namespaced.
	scheme := runtime.NewScheme()
	if err = v3.AddToScheme(scheme); err != nil {
		log.WithError(err).Fatal("Failed to configure controller runtime client")
	}
	client, err := ctrlclient.NewWithWatch(restConfig, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		log.WithError(err).Fatal("Failed to configure controller runtime client with watch")
	}

	// Create linseed Client.
	config := lsrest.Config{
		URL:            cfg.LinseedURL,
		CACertPath:     cfg.LinseedCA,
		ClientKeyPath:  cfg.LinseedClientKey,
		ClientCertPath: cfg.LinseedClientCert,
	}

	linseed, err := lsclient.NewClient(cfg.TenantID, config, lsrest.WithTokenPath(cfg.LinseedToken))
	if err != nil {
		log.WithError(err).Error("failed to create linseed client")
		return err
	}

	// Create queryserver Config.
	qsConfig := &queryserverclient.QueryServerConfig{
		QueryServerTunnelURL: fmt.Sprintf("%s%s", cfg.VoltronURL, cfg.QueryServerURL),
		QueryServerURL:       cfg.QueryServerEndpoint,
		QueryServerCA:        cfg.QueryServerCA,
	}

	k8sClientFactory := datastore.NewClusterCtxK8sClientFactory(restConfig, cfg.VoltronCAPath, cfg.VoltronURL)

	// For handlers that use the newer AuthorizationReview to perform RBAC checks, the k8sClientSetFactory provide
	// cluster and user aware k8s clients.
	k8sClientSetFactory := k8s.NewClientSetFactory(cfg.VoltronCAPath, cfg.VoltronURL)

	// For multi-tenant clusters, we need to configure the clientset factory to impersonate the canonical
	// system:serviceaccount:tigera-manager:tigera-manager service account. This is because each tenant runs
	// this component in its own namespace, whereas the managed cluster isn't tenant aware and expects the
	// canonical name.
	if cfg.TenantNamespace != "" {
		impersonationInfo := user.DefaultInfo{
			Name:   "system:serviceaccount:tigera-manager:tigera-manager",
			Groups: []string{},
		}
		k8sClientSetFactory = k8sClientSetFactory.Impersonate(&impersonationInfo)
		k8sClientFactory = k8sClientFactory.Impersonate(&impersonationInfo)
	}

	// Create a PIP backend.
	p := pip.New(pipcfg.MustLoadConfig(), &clusterAwareLister{k8sClientFactory}, linseed)

	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))

	// Service graph has additional RBAC control built in since it accesses multiple different tables. However, the
	// minimum requirements for accessing SG is access to the flow logs. Perform that check here so it is done at the
	// earliest opportunity.
	sm.Handle("/serviceGraph",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					servicegraph.NewServiceGraphHandler(
						authz,
						client,
						linseed,
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
							ServiceGraphCacheDataPrefetch:         cfg.ServiceGraphCacheDataPrefetch,
							TenantNamespace:                       cfg.TenantNamespace,
						},
					)))))
	sm.Handle("/flowLogs",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogsHandler(k8sClientFactory, linseed, p)))))
	sm.Handle("/flowLogs/aggregation",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					aggregation.NewHandler(linseed, k8sClientSetFactory, aggregation.TypeFlows)))))
	sm.Handle("/flowLogs/search",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						search.SearchTypeFlows,
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSetFactory,
						linseed,
					)))))
	// endpoints/aggregation requires both queryserver and flowlogs rbacs.
	// defer authorizarion to:
	// 1. queryserver endpoints API to queryserver
	// 2. and to flowLogsResourceName within the EndpointsAggregationHandler only for the call to linseed
	sm.Handle("/endpoints/aggregation",
		middleware.AuthenticateRequest(authn,
			middleware.EndpointsAggregationHandler(authz,
				middleware.NewAuthorizationReview(k8sClientSetFactory),
				qsConfig,
				linseed,
			)))
	sm.Handle("/dnsLogs/aggregation",
		middleware.ClusterRequestToResource(dnsLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					aggregation.NewHandler(linseed, k8sClientSetFactory, aggregation.TypeDNS)))))
	sm.Handle("/dnsLogs/search",
		middleware.ClusterRequestToResource(dnsLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						search.SearchTypeDNS,
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSetFactory,
						linseed,
					)))))
	sm.Handle("/l7Logs/aggregation",
		middleware.ClusterRequestToResource(l7ResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					aggregation.NewHandler(linseed, k8sClientSetFactory, aggregation.TypeL7)))))
	sm.Handle("/l7Logs/search",
		middleware.ClusterRequestToResource(l7ResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						search.SearchTypeL7,
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSetFactory,
						linseed,
					)))))
	sm.Handle("/events/bulk",
		middleware.ClusterRequestToResource(eventsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					event.EventHandler(linseed)))))
	sm.Handle("/events/search",
		middleware.ClusterRequestToResource(eventsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					search.SearchHandler(
						search.SearchTypeEvents,
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSetFactory,
						linseed,
					)))))
	sm.Handle("/events/statistics",
		middleware.ClusterRequestToResource(eventsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					event.EventStatisticsHandler(k8sClientSet, linseed)))))

	sm.Handle("/event-exceptions",
		middleware.ClusterRequestToResource(eventsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					exceptions.EventExceptionsHandler(
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						k8sClientSetFactory,
						linseed,
					)))))

	sm.Handle("/auditlogs",
		middleware.ClusterRequestToResource(auditLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					audit.NewHandler(linseed, cfg.ExcludeDryRuns)))))
	sm.Handle("/processes",
		middleware.ClusterRequestToResource(flowLogsResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					process.ProcessHandler(
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						linseed,
					)))))
	sm.Handle("/services",
		middleware.ClusterRequestToResource(l7ResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					application.ApplicationHandler(
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						linseed,
						application.ApplicationTypeService,
					)))))
	sm.Handle("/urls",
		middleware.ClusterRequestToResource(l7ResourceName,
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					application.ApplicationHandler(
						middleware.NewAuthorizationReview(k8sClientSetFactory),
						linseed,
						application.ApplicationTypeURL,
					)))))
	// Perform authn using KubernetesAuthn handler, but authz using PolicyRecommendationHandler.
	sm.Handle("/batchActions",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.BatchStagedActionsHandler(authn, k8sClientSetFactory, k8sClientFactory))))
	sm.Handle("/pagedRecommendations",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.PagedRecommendationsHandler(authn, k8sClientSetFactory, k8sClientFactory))))
	sm.Handle("/recommend",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.PolicyRecommendationHandler(k8sClientSetFactory, k8sClientFactory, linseed)))))
	sm.Handle("/flowLogNamespaces",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogNamespaceHandler(k8sClientFactory, linseed)))))
	sm.Handle("/flowLogNames",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogNamesHandler(k8sClientFactory, linseed)))))
	sm.Handle("/flow",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authn,
				middleware.AuthorizeRequest(authz,
					middleware.NewFlowHandler(linseed, k8sClientFactory)))))
	sm.Handle("/user",
		middleware.AuthenticateRequest(authn,
			middleware.NewUserHandler(k8sClientSet, cfg.OIDCAuthEnabled, cfg.OIDCAuthIssuer, cfg.ElasticLicenseType)))

	if !cfg.ElasticKibanaDisabled {
		kibanaTLSConfig := calicotls.NewTLSConfig()
		kibanaTLSConfig.InsecureSkipVerify = true
		kibanaCli := kibana.NewClient(&http.Client{
			Transport: &http.Transport{TLSClientConfig: kibanaTLSConfig},
		}, cfg.ElasticKibanaEndpoint)

		// Kibana endpoints are only served if configured to have Kibana enabled.
		sm.Handle("/kibana/login",
			middleware.AuthenticateRequest(authn,
				middleware.NewKibanaLoginHandler(k8sClientSet, kibanaCli, cfg.OIDCAuthEnabled, cfg.OIDCAuthIssuer,
					middleware.ElasticsearchLicenseType(cfg.ElasticLicenseType))))
	}

	server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: httputils.LogRequestHeaders(sm),
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
