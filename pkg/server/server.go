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

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/mutator"
	"github.com/tigera/es-proxy/pkg/pip"
)

var (
	server *http.Server
	wg     sync.WaitGroup
)

func Start(config *Config) error {
	sm := http.NewServeMux()

	var rootCAs *x509.CertPool
	if config.ElasticCAPath != "" {
		rootCAs = addCertToCertPool(config.ElasticCAPath)
	}
	var tlsConfig *tls.Config
	if rootCAs != nil {
		tlsConfig = &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: config.ElasticInsecureSkipVerify,
		}
	}

	pc := &handler.ProxyConfig{
		TargetURL:       config.ElasticURL,
		TLSConfig:       tlsConfig,
		ConnectTimeout:  config.ProxyConnectTimeout,
		KeepAlivePeriod: config.ProxyKeepAlivePeriod,
		IdleConnTimeout: config.ProxyIdleConnTimeout,
	}
	proxy := handler.NewProxy(pc)

	//hook up the pip response modifier
	listSrc := &pip.DummySource{} //TODO: not sure what listSrc is or where it is supposed to come from
	p := pip.New(listSrc)
	piphook := mutator.NewResponseHook(p)
	proxy.AddResponseModifier(piphook.ModifyResponse)

	k8sClient, k8sConfig := getKubernetestClientAndConfig()
	k8sAuth := middleware.NewK8sAuth(k8sClient, k8sConfig)

	sm.Handle("/version", http.HandlerFunc(handler.VersionHandler))

	switch config.AccessMode {
	case InsecureMode:
		sm.Handle("/", middleware.RequestToResource(
			k8sAuth.KubernetesAuthnAuthz(proxy)))
	case ServiceUserMode:
		sm.Handle("/", middleware.RequestToResource(
			k8sAuth.KubernetesAuthnAuthz(
				middleware.BasicAuthHeaderInjector(config.ElasticUsername, config.ElasticPassword, proxy))))
	case PassThroughMode:
		log.Fatal("PassThroughMode not implemented yet")
	default:
		log.WithField("AccessMode", config.AccessMode).Fatal("Unknown Elasticsearch access mode.")
	}

	server = &http.Server{
		Addr:    config.ListenAddr,
		Handler: middleware.LogRequestHeaders(sm),
	}

	wg.Add(1)
	go func() {
		log.Infof("Starting server on %v", config.ListenAddr)
		err := server.ListenAndServeTLS(config.CertFile, config.KeyFile)
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
	server.Shutdown(context.Background())
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
