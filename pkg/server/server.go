// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/tigera/compliance/pkg/datastore"
	celastic "github.com/tigera/lma/pkg/elastic"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/middleware"
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

	mcmAuth := middleware.NewMCMAuth(cfg.VoltronCAPath)
	k8sAuth := mcmAuth.DefaultK8sAuth()

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
	p := pip.New(policyCalcConfig, k8sClientSet, esClient)

	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))

	switch cfg.AccessMode {
	case InsecureMode:
		// Perform authn using KubernetesAuthn handler, but authz using PolicyRecommendationHandler.
		sm.Handle("/recommend",
			k8sAuth.KubernetesAuthn(
				middleware.PolicyRecommendationHandler(mcmAuth, k8sClientSet, esClient)))
		sm.Handle("/.kibana/_search",
			middleware.KibanaIndexPattern(
				k8sAuth.KubernetesAuthnAuthz(proxy)))
		sm.Handle("/",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(proxy)))
		sm.Handle("/flowLogNamespaces",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.FlowLogNamespaceHandler(mcmAuth, esClient))))
		sm.Handle("/flowLogNames",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.FlowLogNamesHandler(mcmAuth, esClient))))
		sm.Handle("/flowLogs",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.FlowLogsHandler(mcmAuth, esClient, p))))
	case ServiceUserMode:
		// Perform authn using KubernetesAuthn handler, but authz using PolicyRecommendationHandler.
		sm.Handle("/recommend",
			k8sAuth.KubernetesAuthn(
				middleware.PolicyRecommendationHandler(mcmAuth, k8sClientSet, esClient)))
		sm.Handle("/.kibana/_search",
			middleware.KibanaIndexPattern(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.BasicAuthHeaderInjector(cfg.ElasticUsername, cfg.ElasticPassword, proxy))))
		sm.Handle("/",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.BasicAuthHeaderInjector(cfg.ElasticUsername, cfg.ElasticPassword, proxy))))
		sm.Handle("/flowLogNamespaces",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.FlowLogNamespaceHandler(mcmAuth, esClient))))
		sm.Handle("/flowLogNames",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.FlowLogNamesHandler(mcmAuth, esClient))))
		sm.Handle("/flowLogs",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.FlowLogsHandler(mcmAuth, esClient, p))))
	case PassThroughMode:
		log.Fatal("PassThroughMode not implemented yet")
	default:
		log.WithField("AccessMode", cfg.AccessMode).Fatal("Indeterminate Elasticsearch access mode.")
	}

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
