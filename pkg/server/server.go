// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"

	k8s "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	compconfig "github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/datastore"
	"github.com/tigera/compliance/pkg/elastic"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/pip"
	pipcfg "github.com/tigera/es-proxy/pkg/pip/config"
)

var (
	server *http.Server
	wg     sync.WaitGroup
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

	k8sClient, k8sConfig := getKubernetestClientAndConfig()
	k8sAuth := middleware.NewK8sAuth(k8sClient, k8sConfig)

	// Install pip mutator
	k8sClientSet := datastore.MustGetClientSet()
	policyCalcConfig := pipcfg.MustLoadConfig()

	esClient := elastic.MustGetElasticClient(compconfig.MustLoadConfig())
	p := pip.New(policyCalcConfig, k8sClientSet, esClient, esClient)

	// initialize the esclient for pip
	h := &http.Client{}
	if cfg.ElasticCAPath != "" {
		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCAs}}
	}
	esClient, err := elastic.New(h,
		cfg.ElasticURL,
		cfg.ElasticUsername,
		cfg.ElasticPassword,
		cfg.ElasticIndexSuffix,
		cfg.ElasticConnRetries,
		cfg.ElasticConnRetryInterval,
		cfg.ElasticEnableTrace,
	)
	if err != nil {
		return err
	}

	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))

	switch cfg.AccessMode {
	case InsecureMode:
		sm.Handle("/",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.PolicyImpactHandler(k8sAuth, p, esClient, proxy))))
	case ServiceUserMode:
		sm.Handle("/",
			middleware.RequestToResource(
				k8sAuth.KubernetesAuthnAuthz(
					middleware.PolicyImpactHandler(k8sAuth, p, esClient,
						middleware.BasicAuthHeaderInjector(cfg.ElasticUsername, cfg.ElasticPassword, proxy)))))
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

// getKubernetestClientAndConfig figures out a k8s client, either using a
// incluster config or a provided KUBECONFIG environment variable.
// This function doesn't return an error but instead panics on error.
func getKubernetestClientAndConfig() (k8s.Interface, *restclient.Config) {
	var (
		k8sConfig *restclient.Config
		err       error
	)
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		// Create client with provided kubeconfig
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.WithError(err).Fatal("Could not process kubeconfig file")
		}
	} else {
		k8sConfig, err = restclient.InClusterConfig()
		if err != nil {
			log.WithError(err).Fatal("Could not get in cluster config")
		}
	}
	k8sClient := k8s.NewForConfigOrDie(k8sConfig)
	return k8sClient, k8sConfig
}
