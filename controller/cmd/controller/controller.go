// Copyright 2019 Tigera Inc. All rights reserved.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
	calicoclient "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	alertElastic "github.com/tigera/intrusion-detection/controller/pkg/alert/elastic"
	alertWatcher "github.com/tigera/intrusion-detection/controller/pkg/alert/watcher"
	anomalyWatcher "github.com/tigera/intrusion-detection/controller/pkg/anomaly/watcher"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
	syncElastic "github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	feedsWatcher "github.com/tigera/intrusion-detection/controller/pkg/feeds/watcher"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
)

const (
	DefaultElasticScheme = "https"
	DefaultElasticHost   = "elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"
	DefaultElasticPort   = 9200
	DefaultElasticUser   = "elastic"
	ConfigMapNamespace   = "calico-monitoring"
	SecretsNamespace     = "calico-monitoring"
)

func main() {
	var ver, debug bool
	var healthzSockPort int
	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.IntVar(&healthzSockPort, "port", health.DefaultHealthzSockPort, "Healthz port")
	flag.Parse()

	if ver {
		Version()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeconfig := os.Getenv("KUBECONFIG")
	var config *rest.Config
	var err error
	if kubeconfig == "" {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {
		// creates a config from supplied kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	calicoClient, err := calicoclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	var u *url.URL
	uri := os.Getenv("ELASTIC_URI")
	if uri != "" {
		var err error
		u, err = url.Parse(uri)
		if err != nil {
			panic(err)
		}
	} else {
		scheme := os.Getenv("ELASTIC_SCHEME")
		if scheme == "" {
			scheme = DefaultElasticScheme
		}

		host := os.Getenv("ELASTIC_HOST")
		if host == "" {
			host = DefaultElasticHost
		}

		portStr := os.Getenv("ELASTIC_PORT")
		var port int64
		if portStr == "" {
			port = DefaultElasticPort
		} else {
			var err error
			port, err = strconv.ParseInt(portStr, 10, 16)
			if err != nil {
				panic(err)
			}
		}

		u = &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", host, port),
		}
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}
	user := os.Getenv("ELASTIC_USER")
	if user == "" {
		user = DefaultElasticUser
	}
	pass := os.Getenv("ELASTIC_PASSWORD")
	pathToCA := os.Getenv("ELASTIC_CA")

	ca, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}
	if pathToCA != "" {
		cert, err := ioutil.ReadFile(pathToCA)
		if err != nil {
			panic(err)
		}
		ok := ca.AppendCertsFromPEM(cert)
		if !ok {
			panic("failed to add CA")
		}
	}
	h := &http.Client{}
	if u.Scheme == "https" {
		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{
			RootCAs:            ca,
			InsecureSkipVerify: os.Getenv("INSECURE_SKIP_VERIFY") == "yes",
		}}
	}

	indexSettings := elastic.DefaultIndexSettings()

	if replicas := os.Getenv("ELASTIC_REPLICAS"); replicas != "" {
		if indexSettings.Replicas, err = strconv.Atoi(replicas); err != nil || indexSettings.Replicas < 0 {
			panic("ELASTIC_REPLICAS must be a non negative integer")
		}
	}

	if shards := os.Getenv("ELASTIC_SHARDS"); shards != "" {
		if indexSettings.Shards, err = strconv.Atoi(shards); err != nil || indexSettings.Shards < 1 {
			panic("ELASTIC_SHARDS must be a positive integer")
		}
	}

	e, err := elastic.NewElastic(h, u, user, pass, indexSettings)
	if err != nil {
		log.WithError(err).Fatal("Could not connect to Elastic")
	}
	e.Run(ctx)
	defer e.Close()

	gns := globalnetworksets.NewController(calicoClient.ProjectcalicoV3().GlobalNetworkSets())
	eip := syncElastic.NewIPSetController(e)
	edn := syncElastic.NewDomainNameSetController(e)
	ean := alertElastic.NewAlertController(e)
	sIP := events.NewSuspiciousIP(e)
	sDN := events.NewSuspiciousDomainNameSet(e)

	s := feedsWatcher.NewWatcher(
		k8sClient.CoreV1().ConfigMaps(ConfigMapNamespace),
		k8sClient.CoreV1().Secrets(SecretsNamespace),
		calicoClient.ProjectcalicoV3().GlobalThreatFeeds(),
		gns,
		eip,
		edn,
		&http.Client{},
		e, e, sIP, sDN, e)
	if os.Getenv("DISABLE_FEEDS") != "yes" {
		s.Run(ctx)
		defer s.Close()
	}

	a := anomalyWatcher.NewWatcher(e, e, e)
	if os.Getenv("DISABLE_ANOMALY") != "yes" {
		a.Run(ctx)
		defer a.Close()
	}

	a2 := alertWatcher.NewWatcher(
		calicoClient.ProjectcalicoV3().GlobalAlerts(),
		ean,
		e,
		&http.Client{},
	)
	if os.Getenv("DISABLE_ALERTS") != "yes" {
		a2.Run(ctx)
		defer a2.Close()
	}

	hs := health.NewServer(health.Pingers{s, a, a2}, health.Readiers{health.AlwaysReady{}}, healthzSockPort)
	go func() {
		err := hs.Serve()
		if err != nil {
			log.WithError(err).Error("failed to start healthz server")
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("got signal; shutting down")
	err = hs.Close()
	if err != nil {
		log.WithError(err).Error("failed to stop healthz server")
	}
}
