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

	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/alert"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/globalalert/controllers/managedcluster"
	"github.com/tigera/intrusion-detection/controller/pkg/util"

	log "github.com/sirupsen/logrus"
	lclient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"
	"github.com/tigera/licensing/monitor"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/events"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/rbac"
	syncElastic "github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/feeds/sync/globalnetworksets"
	feedsWatcher "github.com/tigera/intrusion-detection/controller/pkg/feeds/watcher"
	"github.com/tigera/intrusion-detection/controller/pkg/forwarder"
	"github.com/tigera/intrusion-detection/controller/pkg/health"

	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
)

const (
	DefaultElasticScheme                  = "https"
	DefaultElasticHost                    = "elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"
	DefaultElasticPort                    = "9200"
	DefaultElasticUser                    = "elastic"
	DefaultConfigMapNamespace             = "tigera-intrusion-detection"
	DefaultSecretsNamespace               = "tigera-intrusion-detection"
	DefaultMultiClusterForwardingEndpoint = "https://tigera-manager.tigera-manager.svc:9443"
	DefaultMultiClusterForwardingCA       = "/manager-tls/cert"
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
	configMapNamespace := getStrEnvOrDefault("CONFIG_MAP_NAMESPACE", DefaultConfigMapNamespace)
	secretsNamespace := getStrEnvOrDefault("SECRETS_NAMESPACE", DefaultSecretsNamespace)

	var u *url.URL
	uri := os.Getenv("ELASTIC_URI")
	if uri != "" {
		var err error
		u, err = url.Parse(uri)
		if err != nil {
			panic(err)
		}
	} else {
		scheme := getStrEnvOrDefault("ELASTIC_SCHEME", DefaultElasticScheme)
		host := getStrEnvOrDefault("ELASTIC_HOST", DefaultElasticHost)

		portStr := getStrEnvOrDefault("ELASTIC_PORT", DefaultElasticPort)
		port, err := strconv.ParseInt(portStr, 10, 16)
		if err != nil {
			panic(err)
		}

		u = &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", host, port),
		}
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}
	user := getStrEnvOrDefault("ELASTIC_USER", DefaultElasticUser)
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

	esCLI, err := elastic.NewClient(h, u, user, pass, debugElastic)
	if err != nil {
		log.WithError(err).Fatal("Could not connect to Elastic")
		panic(err)
	}
	defer esCLI.Stop()
	e := elastic.NewService(esCLI, u, indexSettings)
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
	valueEnableForwarding, err := strconv.ParseBool(os.Getenv("IDS_ENABLE_EVENT_FORWARDING"))
	var enableForwarding = (err == nil && valueEnableForwarding)
	var healthPingers health.Pingers
	var enableFeeds = (os.Getenv("DISABLE_FEEDS") != "yes")
	if enableFeeds {
		healthPingers = append(healthPingers, s)
	}

	var managementAlertController, managedClusterController controller.Controller
	var alertHealthPinger, managedClusterHealthPinger health.Pingers
	enableAlerts := os.Getenv("DISABLE_ALERTS") != "yes"
	if enableAlerts {
		managementAlertController, alertHealthPinger = alert.NewGlobalAlertController(calicoClient, esCLI, getStrEnvOrDefault("CLUSTER_NAME", "cluster"))
		healthPingers = append(healthPingers, &alertHealthPinger)

		multiClusterForwardingEndpoint := getStrEnvOrDefault("MULTI_CLUSTER_FORWARDING_ENDPOINT", DefaultMultiClusterForwardingEndpoint)
		multiClusterForwardingCA := getStrEnvOrDefault("MULTI_CLUSTER_FORWARDING_CA", DefaultMultiClusterForwardingCA)
		managedClusterController, managedClusterHealthPinger = managedcluster.NewManagedClusterController(calicoClient, esCLI, indexSettings,
			util.ManagedClusterClient(config, multiClusterForwardingEndpoint, multiClusterForwardingCA))
		healthPingers = append(healthPingers, &managedClusterHealthPinger)
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
				managementAlertController.Run(ctx)
				defer managementAlertController.Close()
				managedClusterController.Run(ctx)
				defer managedClusterController.Close()
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
				managedClusterController.Close()
				managementAlertController.Close()
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

// getStrEnvOrDefault returns the environment variable named by the key if it is not empty, else returns the defaultValue
func getStrEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}
