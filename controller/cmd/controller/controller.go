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

	calicoclient "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	log "github.com/sirupsen/logrus"
	lclient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"
	"github.com/tigera/licensing/monitor"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	alertElastic "github.com/tigera/intrusion-detection/controller/pkg/alert/elastic"
	alertWatcher "github.com/tigera/intrusion-detection/controller/pkg/alert/watcher"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/rbac"
	syncElastic "github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	feedsWatcher "github.com/tigera/intrusion-detection/controller/pkg/feeds/watcher"
	"github.com/tigera/intrusion-detection/controller/pkg/forwarder"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
)

const (
	DefaultElasticScheme      = "https"
	DefaultElasticHost        = "elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"
	DefaultElasticPort        = 9200
	DefaultElasticUser        = "elastic"
	DefaultConfigMapNamespace = "tigera-intrusion-detection"
	DefaultSecretsNamespace   = "tigera-intrusion-detection"
)

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() bapi.Client
}

func main() {
	var ver, debug bool
	var healthzSockPort int

	flag.BoolVar(&ver, "version", false, "Print version information")
	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.IntVar(&healthzSockPort, "port", health.DefaultHealthzSockPort, "Healthz port")
	// enable klog flags for API call logging (to stderr).
	klog.InitFlags(flag.CommandLine)
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

	// This allows us to use "calico-monitoring" in helm if we want to
	configMapNamespace := os.Getenv("CONFIG_MAP_NAMESPACE")
	if configMapNamespace == "" {
		configMapNamespace = DefaultConfigMapNamespace
	}
	secretsNamespace := os.Getenv("SECRETS_NAMESPACE")
	if secretsNamespace == "" {
		secretsNamespace = DefaultSecretsNamespace
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

	debugElastic := false
	if elasticDebugFlag := os.Getenv("ELASTIC_DEBUG"); elasticDebugFlag != "" {
		if debugElastic, err = strconv.ParseBool(elasticDebugFlag); err != nil {
			panic("ELASTIC_DEBUG must be a valid boolean")
		}
	}

	e, err := elastic.NewElastic(h, u, user, pass, indexSettings, debugElastic)
	if err != nil {
		log.WithError(err).Fatal("Could not connect to Elastic")
	}
	e.Run(ctx)
	defer e.Close()

	clientCalico, err := client.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("Failed to build calico client")
	}

	licenseMonitor := monitor.New(clientCalico.(backendClientAccessor).Backend())
	err = licenseMonitor.RefreshLicense(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get license from datastore; continuing without a license")
	}

	licenseChangedChan := make(chan struct{})

	// Define some of the callbacks for the license monitor. Any changes just send a signal back on the license changed channel.
	licenseMonitor.SetFeaturesChangedCallback(func() {
		licenseChangedChan <- struct{}{}
	})

	licenseMonitor.SetStatusChangedCallback(func(newLicenseStatus lclient.LicenseStatus) {
		licenseChangedChan <- struct{}{}
	})

	// Start the license monitor, which will trigger the callback above at start of day and then whenever the license
	// status changes.
	go func() {
		err := licenseMonitor.MonitorForever(context.Background())
		if err != nil {
			log.WithError(err).Warn("Error while continuously monitoring the license.")
		}
	}()

	gns := globalnetworksets.NewController(calicoClient.ProjectcalicoV3().GlobalNetworkSets())
	eip := syncElastic.NewIPSetController(e)
	edn := syncElastic.NewDomainNameSetController(e)
	ean := alertElastic.NewAlertController(e)
	sIP := events.NewSuspiciousIP(e)
	sDN := events.NewSuspiciousDomainNameSet(e)

	s := feedsWatcher.NewWatcher(
		k8sClient.CoreV1().ConfigMaps(configMapNamespace),
		rbac.RestrictedSecretsClient{k8sClient.CoreV1().Secrets(secretsNamespace)},
		calicoClient.ProjectcalicoV3().GlobalThreatFeeds(),
		gns,
		eip,
		edn,
		&http.Client{},
		e, e, sIP, sDN, e)
	a := alertWatcher.NewWatcher(
		calicoClient.ProjectcalicoV3().GlobalAlerts(),
		ean,
		e,
		&http.Client{},
	)
	valueEnableForwarding, err := strconv.ParseBool(os.Getenv("IDS_ENABLE_EVENT_FORWARDING"))
	var enableForwarding = (err == nil && valueEnableForwarding)
	var healthPingers health.Pingers
	var enableFeeds = (os.Getenv("DISABLE_FEEDS") != "yes")
	if enableFeeds {
		healthPingers = append(healthPingers, s)
	}

	var enableAlerts = (os.Getenv("DISABLE_ALERTS") != "yes")
	if enableAlerts {
		healthPingers = append(healthPingers, a)
	}

	f := forwarder.NewEventForwarder("eventforwarder-1", e)

	hs := health.NewServer(healthPingers, health.Readiers{health.AlwaysReady{}}, healthzSockPort)
	go func() {
		err := hs.Serve()
		if err != nil {
			log.WithError(err).Error("failed to start healthz server")
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var runningControllers bool
	for {
		hasLicense := licenseMonitor.GetFeatureStatus(features.ThreatDefense)
		if hasLicense && !runningControllers {
			log.Info("Starting watchers and controllers for intrusion detection.")
			if enableFeeds {
				s.Run(ctx)
				defer s.Close()
			}

			if enableAlerts {
				a.Run(ctx)
				defer a.Close()
			}

			if enableForwarding {
				f.Run(ctx)
				defer f.Close()
			}

			runningControllers = true
		} else if !hasLicense && runningControllers {
			log.Info("License is no longer active/feature is disabled.")

			if enableFeeds {
				s.Close()
			}

			if enableAlerts {
				a.Close()
			}

			if enableForwarding {
				f.Close()
			}

			runningControllers = false
		}

		select {
		case <-sig:
			log.Info("got signal; shutting down")
			err = hs.Close()
			if err != nil {
				log.WithError(err).Error("failed to stop healthz server")
			}
			return
		case <-licenseChangedChan:
			log.Info("License status has changed")
			continue
		}
	}
}
