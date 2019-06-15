// Copyright (c) 2017 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/srv"
	"github.com/coreos/etcd/pkg/transport"
	log "github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/logutils"

	"github.com/projectcalico/kube-controllers/pkg/config"
	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/controllers/federatedservices"
	"github.com/projectcalico/kube-controllers/pkg/controllers/namespace"
	"github.com/projectcalico/kube-controllers/pkg/controllers/networkpolicy"
	"github.com/projectcalico/kube-controllers/pkg/controllers/node"
	"github.com/projectcalico/kube-controllers/pkg/controllers/pod"
	"github.com/projectcalico/kube-controllers/pkg/controllers/serviceaccount"
	"github.com/projectcalico/kube-controllers/pkg/status"
	bapi "github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s"
	lclient "github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"
	"github.com/tigera/licensing/monitor"
)

const (
	// Same RC as the one used for a config change by Felix.
	configChangedRC = 129
)

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() bapi.Client
}

// VERSION is filled out during the build process (using git describe output)
var VERSION string
var version bool

func init() {
	// Tell glog (used by client-go) to log into STDERR. Otherwise, we risk
	// certain kinds of API errors getting logged into a directory not
	// available in a `FROM scratch` Docker container, causing glog to abort
	err := flag.Set("logtostderr", "true")
	if err != nil {
		log.WithError(err).Fatal("Failed to set logging configuration")
	}

	// Add a flag to check the version.
	flag.BoolVar(&version, "version", false, "Display version")

	// Also tell klog to log to STDERR.
	var flags flag.FlagSet
	klog.InitFlags(&flags)
	err = flags.Set("logtostderr", "true")
	if err != nil {
		log.WithError(err).Fatal("Failed to set logging configuration")
	}
}

func main() {
	flag.Parse()
	if version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	// Configure log formatting.
	log.SetFormatter(&logutils.Formatter{})

	// Install a hook that adds file/line no information.
	log.AddHook(&logutils.ContextHook{})

	// Attempt to load configuration.
	config := new(config.Config)
	if err := config.Parse(); err != nil {
		log.WithError(err).Fatal("Failed to parse config")
	}
	log.WithField("config", config).Info("Loaded configuration from environment")

	// Set the log level based on the loaded configuration.
	logLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	// Build clients to be used by the controllers.
	k8sClientset, calicoClient, err := getClients(config.Kubeconfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to start")
	}

	stop := make(chan struct{})
	defer close(stop)

	// Create the context.
	ctx := context.Background()

	if !config.DoNotInitializeCalico {
		log.Info("Ensuring Calico datastore is initialized")
		initCtx, cancelInit := context.WithTimeout(ctx, 10*time.Second)
		defer cancelInit()
		err = calicoClient.EnsureInitialized(initCtx, "", "", "k8s")
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize Calico datastore")
		}
	}

	controllerCtrl := &controllerControl{
		ctx:              ctx,
		controllerStates: make(map[string]*controllerState),
		config:           config,
		stop:             stop,
		licenseMonitor:   monitor.New(calicoClient.(backendClientAccessor).Backend()),
	}

	// Create the status file. We will only update it if we have healthchecks enabled.
	s := status.New(status.DefaultStatusFile)

	for _, controllerType := range strings.Split(config.EnabledControllers, ",") {
		switch controllerType {
		case "workloadendpoint":
			podController := pod.NewPodController(ctx, k8sClientset, calicoClient)
			controllerCtrl.controllerStates["Pod"] = &controllerState{
				controller:  podController,
				threadiness: config.WorkloadEndpointWorkers,
			}
		case "profile", "namespace":
			namespaceController := namespace.NewNamespaceController(ctx, k8sClientset, calicoClient)
			controllerCtrl.controllerStates["Namespace"] = &controllerState{
				controller:  namespaceController,
				threadiness: config.ProfileWorkers,
			}
		case "policy":
			policyController := networkpolicy.NewPolicyController(ctx, k8sClientset, calicoClient)
			controllerCtrl.controllerStates["NetworkPolicy"] = &controllerState{
				controller:  policyController,
				threadiness: config.PolicyWorkers,
			}
		case "node":
			nodeController := node.NewNodeController(ctx, k8sClientset, calicoClient, config)
			controllerCtrl.controllerStates["Node"] = &controllerState{
				controller:  nodeController,
				threadiness: config.NodeWorkers,
			}
		case "serviceaccount":
			serviceAccountController := serviceaccount.NewServiceAccountController(ctx, k8sClientset, calicoClient)
			controllerCtrl.controllerStates["ServiceAccount"] = &controllerState{
				controller:  serviceAccountController,
				threadiness: config.ProfileWorkers,
			}
		case "federatedservices":
			federatedEndpointsController := federatedservices.NewFederatedServicesController(ctx, k8sClientset, calicoClient)
			controllerCtrl.controllerStates["FederatedServices"] = &controllerState{
				controller:     federatedEndpointsController,
				licenseFeature: features.FederatedServices,
				threadiness:    config.FederatedServicesWorkers,
			}
			controllerCtrl.needLicenseMonitoring = true
		default:
			log.Fatalf("Invalid controller '%s' provided.", controllerType)
		}
	}

	if config.DatastoreType == "etcdv3" {
		// If configured to do so, start an etcdv3 compaction.
		go startCompactor(ctx, config)
	}

	// Run the health checks on a separate goroutine.
	if config.HealthEnabled {
		log.Info("Starting status report routine")
		go runHealthChecks(ctx, s, k8sClientset, calicoClient)
	}

	// Run the controllers. This runs indefinitely.
	controllerCtrl.RunControllers()
}

// Run the controller health checks.
func runHealthChecks(ctx context.Context, s *status.Status, k8sClientset *kubernetes.Clientset, calicoClient client.Interface) {
	s.SetReady("CalicoDatastore", false, "initialized to false")
	s.SetReady("KubeAPIServer", false, "initialized to false")

	// Loop forever and perform healthchecks.
	for {
		// skip healthchecks if configured
		// Datastore HealthCheck
		healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := calicoClient.EnsureInitialized(healthCtx, "", "", "k8s")
		if err != nil {
			log.WithError(err).Errorf("Failed to verify datastore")
			s.SetReady(
				"CalicoDatastore",
				false,
				fmt.Sprintf("Error verifying datastore: %v", err),
			)
		} else {
			s.SetReady("CalicoDatastore", true, "")
		}
		cancel()

		// Kube-apiserver HealthCheck
		healthStatus := 0
		k8sCheckDone := make(chan interface{}, 1)
		go func(k8sCheckDone <-chan interface{}) {
			time.Sleep(2 * time.Second)
			select {
			case <-k8sCheckDone:
				// The check has completed.
			default:
				// Check is still running, so report not ready.
				s.SetReady(
					"KubeAPIServer",
					false,
					fmt.Sprintf("Error reaching apiserver: taking a long time to check apiserver"),
				)
			}
		}(k8sCheckDone)
		k8sClientset.Discovery().RESTClient().Get().AbsPath("/healthz").Do().StatusCode(&healthStatus)
		k8sCheckDone <- nil
		if healthStatus != http.StatusOK {
			log.WithError(err).Errorf("Failed to reach apiserver")
			s.SetReady(
				"KubeAPIServer",
				false,
				fmt.Sprintf("Error reaching apiserver: %v with http status code: %d", err, healthStatus),
			)
		} else {
			s.SetReady("KubeAPIServer", true, "")
		}

		time.Sleep(10 * time.Second)
	}
}

// Starts an etcdv3 compaction goroutine with the given config.
func startCompactor(ctx context.Context, config *config.Config) {
	interval, err := time.ParseDuration(config.CompactionPeriod)
	if err != nil {
		log.WithError(err).Fatal("Invalid compact interval")
	}

	if interval.Nanoseconds() == 0 {
		log.Info("Disabling periodic etcdv3 compaction")
		return
	}

	// Kick off a periodic compaction of etcd, retry until success.
	for {
		etcdClient, err := newEtcdV3Client()
		if err != nil {
			log.WithError(err).Error("Failed to start etcd compaction routine, retry in 1m")
			time.Sleep(1 * time.Minute)
			continue
		}

		log.WithField("period", interval).Info("Starting periodic etcdv3 compaction")
		etcd3.StartCompactor(ctx, etcdClient, interval)
		break
	}
}

// getClients builds and returns Kubernetes and Calico clients.
func getClients(kubeconfig string) (*kubernetes.Clientset, client.Interface, error) {
	// Get Calico client
	calicoClient, err := client.NewFromEnv()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build Calico client: %s", err)
	}

	// If the calico client is actually a kubernetes backed client, just return the kubernetes
	// clientset that is in that client. This is minor finesse that is only useful for controllers
	// that may be run on clusters using Kubernetes API for the Calico datastore.
	beca := calicoClient.(backendClientAccessor)
	bec := beca.Backend()
	if kc, ok := bec.(*k8s.KubeClient); ok {
		return kc.ClientSet, calicoClient, err
	}

	// Now build the Kubernetes client, we support in-cluster config and kubeconfig
	// as means of configuring the client.
	k8sconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build kubernetes client config: %s", err)
	}

	// Get Kubernetes clientset
	k8sClientset, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	return k8sClientset, calicoClient, nil
}

// Returns an etcdv3 client based on the environment. The client will be configured to
// match that in use by the libcalico-go client.
func newEtcdV3Client() (*clientv3.Client, error) {
	config, err := apiconfig.LoadClientConfigFromEnvironment()
	if err != nil {
		return nil, err
	}

	if config.Spec.EtcdEndpoints != "" && config.Spec.EtcdDiscoverySrv != "" {
		log.Warning("Multiple etcd endpoint discovery methods specified in etcdv3 API config")
		return nil, fmt.Errorf("multiple discovery or bootstrap options specified, use either \"etcdEndpoints\" or \"etcdDiscoverySrv\"")
	}

	// Split the endpoints into a location slice.
	etcdLocation := []string{}
	if config.Spec.EtcdEndpoints != "" {
		etcdLocation = strings.Split(config.Spec.EtcdEndpoints, ",")
	}

	if config.Spec.EtcdDiscoverySrv != "" {
		srvs, srvErr := srv.GetClient("etcd-client", config.Spec.EtcdDiscoverySrv)
		if srvErr != nil {
			return nil, fmt.Errorf("failed to discover etcd endpoints through SRV discovery: %v", srvErr)
		}
		etcdLocation = srvs.Endpoints
	}

	if len(etcdLocation) == 0 {
		log.Warning("No etcd endpoints specified in etcdv3 API config")
		return nil, fmt.Errorf("no etcd endpoints specified")
	}

	// Create the etcd client
	tlsInfo := &transport.TLSInfo{
		CAFile:   config.Spec.EtcdCACertFile,
		CertFile: config.Spec.EtcdCertFile,
		KeyFile:  config.Spec.EtcdKeyFile,
	}
	tls, _ := tlsInfo.ClientConfig()

	cfg := clientv3.Config{
		Endpoints:   etcdLocation,
		TLS:         tls,
		DialTimeout: 10 * time.Second,
	}

	// Plumb through the username and password if both are configured.
	if config.Spec.EtcdUsername != "" && config.Spec.EtcdPassword != "" {
		cfg.Username = config.Spec.EtcdUsername
		cfg.Password = config.Spec.EtcdPassword
	}

	return clientv3.New(cfg)
}

// Object for keeping track of controller states and statuses.
type controllerControl struct {
	ctx                   context.Context
	controllerStates      map[string]*controllerState
	config                *config.Config
	stop                  chan struct{}
	licenseMonitor        monitor.LicenseMonitor
	needLicenseMonitoring bool
}

// Runs all the controllers. Calls a hard exit to restart the controller process if
// a license change invalidates a running controller.
func (cc *controllerControl) RunControllers() {
	// Instantiate the license monitor values
	lCtx, cancel := context.WithTimeout(cc.ctx, 10*time.Second)
	err := cc.licenseMonitor.RefreshLicense(lCtx)
	cancel()
	if err != nil {
		log.WithError(err).Error("Failed to get license from datastore; continuing without a license")
	}

	if cc.config.DebugUseShortPollIntervals {
		log.Info("Using short license poll interval for FV")
		cc.licenseMonitor.SetPollInterval(1 * time.Second)
	}

	licenseChangedChan := make(chan struct{})

	// Define some of the callbacks for the license monitor. Any changes just send a signal back on the license changed channel.
	cc.licenseMonitor.SetFeaturesChangedCallback(func() {
		licenseChangedChan <- struct{}{}
	})

	cc.licenseMonitor.SetStatusChangedCallback(func(newLicenseStatus lclient.LicenseStatus) {
		licenseChangedChan <- struct{}{}
	})

	if cc.needLicenseMonitoring {
		// Start the license monitor, which will trigger the callback above at start of day and then whenever the license
		// status changes.
		// Need to wrap the call to MonitorForever in a function to pass static-checks.
		go func() {
			err := cc.licenseMonitor.MonitorForever(context.Background())
			if err != nil {
				log.WithError(err).Warn("Error while continuously monitoring the license.")
			}
		}()
	}

	// Start the controllers then wait indefinitely for license changes to come through to update the controllers as needed.
	for {
		for controllerType, cs := range cc.controllerStates {
			missingLicense := cs.licenseFeature != "" && !cc.licenseMonitor.GetFeatureStatus(cs.licenseFeature)
			if !cs.running && !missingLicense {
				// Run the controller
				log.Infof("Started the %s controller", controllerType)
				go cs.controller.Run(cs.threadiness, cc.config.ReconcilerPeriod, cc.stop)
				cs.running = true
			} else if cs.running && missingLicense {
				// Restart the controller since the updated license has less functionality than before.
				log.Warn("License was changed, shutting down the controllers")
				os.Exit(configChangedRC)
			}
		}

		// Wait until an update is made to see if we need to make changes to the running sets of controllers.
		<-licenseChangedChan
	}
}

// Object for keeping track of Controller information.
type controllerState struct {
	controller     controller.Controller
	running        bool
	licenseFeature string
	threadiness    int
}
