// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/datastore"
	lmaauth "github.com/tigera/lma/pkg/auth"
	celastic "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/list"

	"github.com/projectcalico/apiserver/pkg/authentication"

	eselastic "github.com/tigera/es-proxy/pkg/elastic"
	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/k8s"
	"github.com/tigera/es-proxy/pkg/kibana"
	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/middleware/servicegraph"
	"github.com/tigera/es-proxy/pkg/pip"
	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
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
	shardsNotRequired   = 0
	replicasNotRequired = 0
	voltronServiceURL   = "https://localhost:9443"
)

func Start(cfg *Config) error {
	sm := http.NewServeMux()

	var rootCAs *x509.CertPool
	if cfg.ElasticCAPath != "" {
		rootCAs = addCertToCertPool(cfg.ElasticCAPath)
	}
	var tlsConfig *tls.Config
	if rootCAs != nil {
		tlsConfig = &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: cfg.ElasticInsecureSkipVerify,
		}
	}

	pc := &handler.ProxyConfig{
		TargetURL:       cfg.ElasticURL,
		TLSConfig:       tlsConfig,
		ConnectTimeout:  cfg.ProxyConnectTimeout,
		KeepAlivePeriod: cfg.ProxyKeepAlivePeriod,
		IdleConnTimeout: cfg.ProxyIdleConnTimeout,
	}
	proxy := handler.NewProxy(pc)
	authenticator, err := authentication.New()
	if err != nil {
		log.WithError(err).Panic("Unable to create auth configuration")
	}

	if cfg.OIDCAuthEnabled {
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
			log.WithError(err).Panic("Unable to create dex authenticator")
		}
		authenticator = lmaauth.NewAggregateAuthenticator(dex, authenticator)
	}

	// Install pip mutator
	k8sClientSet := datastore.MustGetClientSet()
	policyCalcConfig := pipcfg.MustLoadConfig()

	// initialize the esclient for pip
	h := &http.Client{}
	if cfg.ElasticCAPath != "" {
		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCAs}}
	}

	esClient, err := celastic.New(h,
		cfg.ElasticURL,
		cfg.ElasticUsername,
		cfg.ElasticPassword,
		cfg.ElasticIndexSuffix,
		cfg.ElasticConnRetries,
		cfg.ElasticConnRetryInterval,
		cfg.ElasticEnableTrace,
		replicasNotRequired,
		shardsNotRequired,
	)
	if err != nil {
		return err
	}

	restConfig := datastore.MustGetConfig()

	//TODO(rlb): I think we can remove this factory in favor of the user and cluster aware factory that can do user based
	//  authorization review that performs multiple checks in a single request.
	k8sClientFactory := datastore.NewClusterCtxK8sClientFactory(restConfig, cfg.VoltronCAPath, voltronServiceURL)
	k8sCli, err := k8sClientFactory.ClientSetForCluster(datastore.DefaultCluster)
	if err != nil {
		panic(err)
	}
	authz := lmaauth.NewRBACAuthorizer(k8sCli)

	// For handlers that use the newer AuthorizationReview to perform RBAC checks, the k8sClientSetFactory provide
	// cluster and user aware k8s clients.
	k8sClientSetFactory := k8s.NewClientSetFactory(cfg.VoltronCAPath, voltronServiceURL)

	// Create a service graph handler.
	serviceGraph := servicegraph.NewServiceGraph(context.Background(), esClient, k8sClientSetFactory)

	// Create a PIP backend.
	p := pip.New(policyCalcConfig, &clusterAwareLister{k8sClientFactory}, esClient)

	kibanaCli := kibana.NewClient(&http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}, cfg.ElasticKibanaEndpoint, cfg.ElasticVersion)

	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))
	sm.Handle("/serviceGraph",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					serviceGraph.Handler()))))
	sm.Handle("/flowLogs",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogsHandler(k8sClientFactory, esClient, p)))))
	sm.Handle("/flowLogs/search",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.SearchHandler(eselastic.GetFlowsIndex, esClient.Backend())))))
	sm.Handle("/dnsLogs/search",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.SearchHandler(eselastic.GetDnsIndex, esClient.Backend())))))
	// Perform authn using KubernetesAuthn handler, but authz using PolicyRecommendationHandler.
	sm.Handle("/recommend",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.PolicyRecommendationHandler(k8sClientFactory, k8sClientSet, esClient)))))
	sm.Handle("/flowLogNamespaces",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogNamespaceHandler(k8sClientFactory, esClient)))))
	sm.Handle("/flowLogNames",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.FlowLogNamesHandler(k8sClientFactory, esClient)))))
	sm.Handle("/flow",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.NewFlowHandler(esClient, k8sClientFactory)))))
	sm.Handle("/user",
		middleware.AuthenticateRequest(authenticator,
			middleware.NewUserHandler(k8sClientSet, cfg.OIDCAuthEnabled, cfg.OIDCAuthIssuer, cfg.ElasticLicenseType)))
	sm.Handle("/kibana/login",
		middleware.SetAuthorizationHeaderFromCookie(
			middleware.AuthenticateRequest(authenticator,
				middleware.NewKibanaLoginHandler(k8sCli, kibanaCli, cfg.OIDCAuthEnabled, cfg.OIDCAuthIssuer,
					middleware.ElasticsearchLicenseType(cfg.ElasticLicenseType)))))
	sm.Handle("/.kibana/_search",
		middleware.KibanaIndexPattern(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.BasicAuthHeaderInjector(cfg.ElasticUsername, cfg.ElasticPassword, proxy)))))
	sm.Handle("/",
		middleware.RequestToResource(
			middleware.AuthenticateRequest(authenticator,
				middleware.AuthorizeRequest(authz,
					middleware.BasicAuthHeaderInjector(cfg.ElasticUsername, cfg.ElasticPassword, proxy)))))

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

func addCertToCertPool(caPath string) *x509.CertPool {
	caContent, err := ioutil.ReadFile(caPath)
	if err != nil {
		log.WithError(err).WithField("CA-Path", caPath).Fatal("Could not read CA file")
	}

	systemCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.WithError(err).Fatal("Could not parse CA file")
	}

	ok := systemCertPool.AppendCertsFromPEM(caContent)
	if !ok {
		log.WithError(err).Fatal("Could not add CA to pool")
	}
	return systemCertPool
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
