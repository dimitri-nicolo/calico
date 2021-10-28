// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
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
package config

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/projectcalico/kube-controllers/pkg/config/configfactory"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/libcalico-go/lib/watch"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/errors"
	"github.com/projectcalico/libcalico-go/lib/options"
)

const datastoreBackoff = time.Second

// RunConfig represents the configuration for all controllers and includes
// merged information from environment variables (Config) and the Calico
// resource KubeControllersConfiguration
type RunConfig struct {
	LogLevelScreen         log.Level
	Controllers            ControllersConfig
	EtcdV3CompactionPeriod time.Duration
	HealthEnabled          bool
	PrometheusPort         int

	ShortLicensePolling bool
}

type ControllersConfig struct {
	Node                       *NodeControllerConfig
	Policy                     *GenericControllerConfig
	WorkloadEndpoint           *GenericControllerConfig
	ServiceAccount             *GenericControllerConfig
	Namespace                  *GenericControllerConfig
	Service                    *GenericControllerConfig
	FederatedServices          *GenericControllerConfig
	ElasticsearchConfiguration *ElasticsearchCfgControllerCfg
	AuthorizationConfiguration *AuthorizationControllerCfg
	ManagedCluster             *ManagedClusterControllerConfig
}

type GenericControllerConfig struct {
	ReconcilerPeriod time.Duration
	NumberOfWorkers  int
}

type NodeControllerConfig struct {
	SyncLabels        bool
	AutoHostEndpoints bool

	// Should the Node controller delete Calico nodes?  Generally, this is
	// true for etcdv3 datastores.
	DeleteNodes bool

	// The grace period used by the controller to determine if an IP address is leaked.
	// Set to 0 to disable IP address garbage collection.
	LeakGracePeriod *v1.Duration

	RESTConfig *restclient.Config
}

type ElasticsearchCfgControllerCfg struct {
	NumberOfWorkers int
	RESTConfig      *restclient.Config
}

type AuthorizationControllerCfg struct {
	NumberOfWorkers                   int
	ReconcilerPeriod                  time.Duration
	OIDCAuthUsernamePrefix            string
	OIDCAuthGroupPrefix               string
	EnableElasticsearchOIDCWorkaround bool
}

type ManagedClusterControllerConfig struct {
	GenericControllerConfig
	RESTConfig                     *restclient.Config
	MultiClusterForwardingEndpoint string
	MultiClusterForwardingCA       string
	ElasticConfig                  ElasticsearchCfgControllerCfg
	LicenseConfig                  LicenseControllerCfg
}

type RunConfigController struct {
	out chan RunConfig
}

type LicenseControllerCfg struct {
	NumberOfWorkers int
	RESTConfig      *restclient.Config
}

// ConfigChan returns a channel that sends an initial config snapshot at start
// of day, and updates whenever the config changes.
func (r *RunConfigController) ConfigChan() <-chan RunConfig {
	return r.out
}

// NewRunConfigController creates the RunConfigController.  The controller connects
// to the datastore to get the KubeControllersConfiguration resource, merges it with
// the config from environment variables, and emits RunConfig objects over a channel
// to push config out to the rest of the controllers.  It also handles setting the
// KubeControllersConfiguration.Status with the current running configuration
func NewRunConfigController(ctx context.Context, cfg Config, client clientv3.KubeControllersConfigurationInterface) *RunConfigController {
	ctrl := &RunConfigController{out: make(chan RunConfig)}
	go syncDatastore(ctx, cfg, client, ctrl.out)
	return ctrl
}

func syncDatastore(ctx context.Context, cfg Config, client clientv3.KubeControllersConfigurationInterface, out chan<- RunConfig) {
	var snapshot *v3.KubeControllersConfiguration
	var err error
	var current RunConfig
	// currentSet tracks whether we've explicitly set `current` to distinguish
	// so we can tell the difference between its initial state and begin explicitly
	// set to the empty state.
	var currentSet bool
	var w watch.Interface

	env := make(map[string]string)
	for _, k := range AllEnvs {
		v, ok := os.LookupEnv(k)
		if ok {
			env[k] = v
		}
	}

MAINLOOP:
	for {
		// Check if our context is expired
		select {
		case <-ctx.Done():
			return
		default:
			// no-op
		}

		// if we don't have a snapshot, then try to get one
		if snapshot == nil {
			snapshot, err = getOrCreateSnapshot(ctx, client, cfg.KubeControllersConfigName)
			if err != nil {
				log.WithError(err).Warn(fmt.Sprintf("unable to get KubeControllersConfiguration(%s)", cfg.KubeControllersConfigName))
				snapshot = nil
				time.Sleep(datastoreBackoff)
				continue MAINLOOP
			}
		}

		// Ok, we should now have a snapshot.  Combine it with the environment variable
		// config to get the running config.
		new, status := mergeConfig(env, cfg, snapshot.Spec)

		// Write the status back to the API datastore, so that end users can inspect the current
		// running config.
		snapshot.Status = status
		snapshot, err = client.Update(ctx, snapshot, options.SetOptions{})
		if err != nil {
			log.WithError(err).Warn(fmt.Sprintf("unable to perform status update on KubeControllersConfiguration(%s)", cfg.KubeControllersConfigName))
			snapshot = nil
			time.Sleep(datastoreBackoff)
			continue MAINLOOP
		}

		// With the snapshot updated, get a list of
		// kubecontrollersconfigurations so we can watch on its resource
		// version.
		kccList, err := client.List(ctx, options.ListOptions{Name: cfg.KubeControllersConfigName})
		if err != nil {
			log.WithError(err).Warn(fmt.Sprintf("unable to list KubeControllersConfiguration(%s)", cfg.KubeControllersConfigName))
			snapshot = nil
			time.Sleep(datastoreBackoff)
			continue MAINLOOP
		}

		// Is this new running config different than our current?
		if !currentSet || !reflect.DeepEqual(new, current) {
			out <- new
			currentSet = true
			current = new
		}

		// Watch for changes
		if w != nil {
			w.Stop()
		}
		w, err = client.Watch(ctx, options.ListOptions{ResourceVersion: kccList.ResourceVersion, Name: cfg.KubeControllersConfigName})
		if err != nil {
			// Watch failed
			log.WithError(err).Warn("unable to watch KubeControllersConfigurations")
			snapshot = nil
			time.Sleep(datastoreBackoff)
			continue MAINLOOP
		}
		defer w.Stop()
		for e := range w.ResultChan() {
			switch e.Type {
			case watch.Error:
				// Watch error; restart from beginning. Note that k8s watches terminate periodically but these
				// terminate without error - in this case we'll just attempt to watch from the latest snapshot rev.
				log.WithError(err).Error("error watching KubeControllersConfiguration")
				snapshot = nil
				time.Sleep(datastoreBackoff)
				continue MAINLOOP
			case watch.Added, watch.Modified:
				// New snapshot
				newKCC := e.Object.(*v3.KubeControllersConfiguration)
				if newKCC.Name != cfg.KubeControllersConfigName {
					// Some non-default object got into the datastore --- calicoctl should
					// prevent this, but an admin with datastore access might not know better.
					// Ignore it
					log.WithField("name", newKCC.Name).Warning("unexpected KubeControllersConfiguration object")
					continue
				}
				snapshot = newKCC
				new, status = mergeConfig(env, cfg, snapshot.Spec)

				// Update the status, but only if it's different, otherwise
				// our update will trigger a watch update in an infinite loop
				if !reflect.DeepEqual(snapshot.Status, status) {
					snapshot.Status = status
					snapshot, err = client.Update(ctx, snapshot, options.SetOptions{})
					if err != nil {
						// this probably means someone else is trying to write to the resource,
						// so best to just take a breath and start over
						log.WithError(err).Warn(fmt.Sprintf("unable to perform status update on KubeControllersConfiguration(%s)", cfg.KubeControllersConfigName))
						snapshot = nil
						time.Sleep(datastoreBackoff)
						continue MAINLOOP
					}
				}

				// Do we need to push an update?
				if !reflect.DeepEqual(new, current) {
					out <- new
					currentSet = true
					current = new
				}
			case watch.Deleted:
				// I think in some oddball cases the watcher can set this to nil
				// so guard against it.
				if e.Previous != nil {
					oldKCC := e.Previous.(*v3.KubeControllersConfiguration)
					// Ignore any object whose name doesn't match
					if oldKCC.Name == cfg.KubeControllersConfigName {
						// do a full resync, which will recreate an object
						// if one doesn't exist
						snapshot = nil
						// not an error per-se, so don't bother with sleeping
						// to backoff
						continue MAINLOOP
					}
				}
			}
		}
	}
}

// getOrCreateSnapshot gets the current KubeControllersConfig from the datastore,
// or creates and returns one with KubeControllersConfig.Name if it doesn't exist
func getOrCreateSnapshot(ctx context.Context, kcc clientv3.KubeControllersConfigurationInterface, configName string) (*v3.KubeControllersConfiguration, error) {
	snapshot, err := kcc.Get(ctx, configName, options.GetOptions{})
	// If the KubeControllersConfig with given name doesn't exist, we'll create it.
	if _, ok := err.(errors.ErrorResourceDoesNotExist); ok {
		initialConfig, getConfigErr := configfactory.NewDefaultKubeControllersConfig(configName)
		if getConfigErr != nil {
			return nil, getConfigErr
		}
		toBeCreated := initialConfig.DeepCopy()
		var createSnapshotErr error
		snapshot, createSnapshotErr = kcc.Create(ctx, toBeCreated, options.SetOptions{})
		if createSnapshotErr != nil {
			// Besides datastore connection errors, we might get a race with
			// something else creating the resource but this can get handled
			// in the main retry loop just fine
			return nil, createSnapshotErr
		}
	} else if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// mergeConfig takes the environment variables, and resulting config
func mergeConfig(envVars map[string]string, envCfg Config, apiCfg v3.KubeControllersConfigurationSpec) (RunConfig, v3.KubeControllersConfigurationStatus) {
	var rCfg RunConfig
	status := v3.KubeControllersConfigurationStatus{EnvironmentVars: map[string]string{}}
	rc := &rCfg.Controllers

	mergeLogLevel(envVars, &status, &rCfg, apiCfg)

	mergeEnabledControllers(envVars, &status, &rCfg, apiCfg)

	mergeReconcilerPeriod(envVars, &status, &rCfg)

	mergeCompactionPeriod(envVars, &status, &rCfg, apiCfg)

	mergeHealthEnabled(envVars, &status, &rCfg, apiCfg)

	// Merge prometheus information.
	if apiCfg.PrometheusMetricsPort != nil {
		rCfg.PrometheusPort = *apiCfg.PrometheusMetricsPort
	}

	// Don't bother looking at this unless the node controller is enabled.
	if rc.Node != nil {
		mergeSyncNodeLabels(envVars, &status, &rCfg, apiCfg, envCfg)

		mergeAutoHostEndpoints(envVars, &status, &rCfg, apiCfg)

		// There is no env var config for this, so always merge from the API config.
		if apiCfg.Controllers.Node != nil {
			rc.Node.LeakGracePeriod = apiCfg.Controllers.Node.LeakGracePeriod
			status.RunningConfig.Controllers.Node.LeakGracePeriod = apiCfg.Controllers.Node.LeakGracePeriod
		}

		if envCfg.DatastoreType != "kubernetes" {
			rc.Node.DeleteNodes = true
			// This field doesn't have an equivalent in the status
		}

		restCfg, err := clientcmd.BuildConfigFromFlags("", envCfg.Kubeconfig)
		if err != nil {
			log.WithError(err).Fatal("failed to build kubernetes client config")
		}
		rc.Node.RESTConfig = restCfg
	}

	// Number of workers is not exposed on the API, so just use the envCfg for it
	// NOTE: NodeController doesn't actually use number of workers config, so don't
	//       bother setting it.
	if rc.Policy != nil {
		rc.Policy.NumberOfWorkers = envCfg.PolicyWorkers
	}
	if rc.WorkloadEndpoint != nil {
		rc.WorkloadEndpoint.NumberOfWorkers = envCfg.WorkloadEndpointWorkers
	}
	if rc.ServiceAccount != nil {
		rc.ServiceAccount.NumberOfWorkers = envCfg.ProfileWorkers
	}
	if rc.Namespace != nil {
		rc.Namespace.NumberOfWorkers = envCfg.ProfileWorkers
	}
	if rc.FederatedServices != nil {
		rc.FederatedServices.NumberOfWorkers = envCfg.FederatedServicesWorkers
	}

	// These controllers aren't supported on the API yet, so no need to merge
	// configuration. So, we just apply the fields from the environment variables
	if rc.Service != nil {
		rc.Service.NumberOfWorkers = envCfg.ServiceWorkers
	}
	if rc.ElasticsearchConfiguration != nil {
		rc.ElasticsearchConfiguration.NumberOfWorkers = envCfg.ManagedClusterWorkers
		restCfg, err := clientcmd.BuildConfigFromFlags("", envCfg.Kubeconfig)
		if err != nil {
			log.WithError(err).Fatal("failed to build kubernetes client config")
		}
		rc.ElasticsearchConfiguration.RESTConfig = restCfg
	}
	if rc.ManagedCluster != nil {
		rc.ManagedCluster.NumberOfWorkers = envCfg.ManagedClusterWorkers
		rc.ManagedCluster.ElasticConfig.NumberOfWorkers = envCfg.ManagedClusterElasticsearchConfigurationWorkers
		rc.ManagedCluster.LicenseConfig.NumberOfWorkers = envCfg.ManagedClusterLicenseConfigurationWorkers
		rc.ManagedCluster.MultiClusterForwardingEndpoint = envCfg.MultiClusterForwardingEndpoint
		rc.ManagedCluster.MultiClusterForwardingCA = envCfg.MultiClusterForwardingCA
		restCfg, err := clientcmd.BuildConfigFromFlags("", envCfg.Kubeconfig)
		if err != nil {
			log.WithError(err).Fatal("failed to build kubernetes client config")
		}
		rc.ManagedCluster.RESTConfig = restCfg
	}
	if rc.AuthorizationConfiguration != nil {
		rc.AuthorizationConfiguration.NumberOfWorkers = envCfg.AuthorizationWorkers
		rc.AuthorizationConfiguration.OIDCAuthUsernamePrefix = envCfg.OIDCAuthUsernamePrefix
		rc.AuthorizationConfiguration.OIDCAuthGroupPrefix = envCfg.OIDCAuthGroupPrefix
		rc.AuthorizationConfiguration.EnableElasticsearchOIDCWorkaround = envCfg.EnableElasticsearchOIDCWorkaround
	}

	rCfg.ShortLicensePolling = envCfg.DebugUseShortPollIntervals

	return rCfg, status
}

func mergeAutoHostEndpoints(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig, apiCfg v3.KubeControllersConfigurationSpec) {
	// make these names shorter
	rc := &rCfg.Controllers
	ac := &apiCfg.Controllers
	sc := &status.RunningConfig.Controllers

	v, p := envVars[EnvAutoHostEndpoints]
	if p {
		status.EnvironmentVars[EnvAutoHostEndpoints] = v
		if strings.ToLower(v) == "enabled" {
			rc.Node.AutoHostEndpoints = true
		} else if strings.ToLower(v) != "disabled" {
			log.WithField(EnvAutoHostEndpoints, v).Fatal("invalid environment variable value")
		}
	} else {
		if ac.Node != nil && ac.Node.HostEndpoint != nil && ac.Node.HostEndpoint.AutoCreate == v3.Enabled {
			rc.Node.AutoHostEndpoints = true
		}
	}
	if rc.Node.AutoHostEndpoints {
		sc.Node.HostEndpoint = &v3.AutoHostEndpointConfig{AutoCreate: v3.Enabled}
	} else {
		sc.Node.HostEndpoint = &v3.AutoHostEndpointConfig{AutoCreate: v3.Disabled}
	}
}

func mergeSyncNodeLabels(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig, apiCfg v3.KubeControllersConfigurationSpec, cfg Config) {
	// make these names shorter
	rc := &rCfg.Controllers
	ac := &apiCfg.Controllers
	sc := &status.RunningConfig.Controllers

	// Don't sync labels in Kubernetes, since the labels are already there
	if cfg.DatastoreType == "kubernetes" {
		status.EnvironmentVars["DATASTORE_TYPE"] = "kubernetes"
		rc.Node.SyncLabels = false
	} else {
		// Etcd datastore, are we configured to sync labels?
		v, p := envVars[EnvSyncNodeLabels]
		if p {
			status.EnvironmentVars[EnvSyncNodeLabels] = v
			snl, err := strconv.ParseBool(v)
			if err != nil {
				log.WithField(EnvSyncNodeLabels, v).Fatal("invalid environment variable value")
			}
			rc.Node.SyncLabels = snl
		} else {
			// No environment variable
			if ac.Node != nil && ac.Node.SyncLabels == v3.Disabled {
				rc.Node.SyncLabels = false
			} else {
				// includes default case of not included as well
				rc.Node.SyncLabels = true
			}
		}
	}
	if rc.Node.SyncLabels {
		sc.Node.SyncLabels = v3.Enabled
	} else {
		sc.Node.SyncLabels = v3.Disabled
	}
}

func mergeHealthEnabled(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig, apiCfg v3.KubeControllersConfigurationSpec) {
	v, p := envVars[EnvHealthEnabled]
	if p {
		status.EnvironmentVars[EnvHealthEnabled] = v
		he, err := strconv.ParseBool(v)
		if err != nil {
			log.WithField(EnvHealthEnabled, v).Fatal("invalid environment variable value")
		}
		rCfg.HealthEnabled = he
	} else {
		// Not set on env, use API
		if apiCfg.HealthChecks != v3.Disabled {
			// Covers "" and "Enabled", as well as an invalid data, since Enabled is the default
			rCfg.HealthEnabled = true
		}
	}
	if rCfg.HealthEnabled {
		status.RunningConfig.HealthChecks = v3.Enabled
	} else {
		status.RunningConfig.HealthChecks = v3.Disabled
	}
}

func mergeCompactionPeriod(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig, apiCfg v3.KubeControllersConfigurationSpec) {
	v, p := envVars[EnvCompactionPeriod]
	if p {
		status.EnvironmentVars[EnvCompactionPeriod] = v
		d, err := time.ParseDuration(v)
		if err != nil {
			log.WithField(EnvCompactionPeriod, v).Fatal("invalid environment variable value")
		}
		rCfg.EtcdV3CompactionPeriod = d
	} else {
		// Not set on environment variable
		if apiCfg.EtcdV3CompactionPeriod != nil {
			rCfg.EtcdV3CompactionPeriod = apiCfg.EtcdV3CompactionPeriod.Duration
		} else {
			// Not set on API, use default
			rCfg.EtcdV3CompactionPeriod = time.Minute * 10
		}
	}
	status.RunningConfig.EtcdV3CompactionPeriod = &v1.Duration{Duration: rCfg.EtcdV3CompactionPeriod}
}

func mergeReconcilerPeriod(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig) {
	// make these names shorter
	rc := &rCfg.Controllers
	sc := &status.RunningConfig.Controllers

	v, p := envVars[EnvReconcilerPeriod]
	if p {
		status.EnvironmentVars[EnvReconcilerPeriod] = v
		d, err := time.ParseDuration(v)
		if err != nil {
			log.WithField(EnvReconcilerPeriod, v).Fatal("invalid environment variable value")
		}
		// Valid env value, set on every enabled controller
		// NOTE: Node controller doesn't use a cache, so ignores reconciler period
		if rc.Policy != nil {
			rc.Policy.ReconcilerPeriod = d
			sc.Policy.ReconcilerPeriod = &v1.Duration{Duration: d}
		}
		if rc.WorkloadEndpoint != nil {
			rc.WorkloadEndpoint.ReconcilerPeriod = d
			sc.WorkloadEndpoint.ReconcilerPeriod = &v1.Duration{Duration: d}
		}
		if rc.ServiceAccount != nil {
			rc.ServiceAccount.ReconcilerPeriod = d
			sc.ServiceAccount.ReconcilerPeriod = &v1.Duration{Duration: d}
		}
		if rc.Namespace != nil {
			rc.Namespace.ReconcilerPeriod = d
			sc.Namespace.ReconcilerPeriod = &v1.Duration{Duration: d}
		}
		if rc.Service != nil {
			rc.Service.ReconcilerPeriod = d
			// not supported on KubeControllersConfiguration
		}
		if rc.FederatedServices != nil {
			rc.FederatedServices.ReconcilerPeriod = d
			sc.FederatedServices.ReconcilerPeriod = &v1.Duration{Duration: d}
		}
		if rc.ManagedCluster != nil {
			rc.ManagedCluster.ReconcilerPeriod = d
			// not supported on KubeControllersConfiguration
		}
		if rc.AuthorizationConfiguration != nil {
			rc.AuthorizationConfiguration.ReconcilerPeriod = d
			// not supported on KubeControllersConfiguration
		}
	}
}

func mergeEnabledControllers(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig, apiCfg v3.KubeControllersConfigurationSpec) {
	// make these names shorter
	rc := &rCfg.Controllers
	ac := apiCfg.Controllers
	sc := &status.RunningConfig.Controllers
	n := ac.Node
	pol := ac.Policy
	w := ac.WorkloadEndpoint
	s := ac.ServiceAccount
	ns := ac.Namespace
	f := ac.FederatedServices

	v, p := envVars[EnvEnabledControllers]
	if p {
		status.EnvironmentVars[EnvEnabledControllers] = v
		log.WithField(EnvEnabledControllers, v).Debug("applying env config")
		for _, controllerType := range strings.Split(v, ",") {
			switch controllerType {
			case "workloadendpoint":
				rc.WorkloadEndpoint = &GenericControllerConfig{}
				sc.WorkloadEndpoint = &v3.WorkloadEndpointControllerConfig{}
			case "profile", "namespace":
				rc.Namespace = &GenericControllerConfig{}
				sc.Namespace = &v3.NamespaceControllerConfig{}
			case "policy":
				rc.Policy = &GenericControllerConfig{}
				sc.Policy = &v3.PolicyControllerConfig{}
			case "node":
				rc.Node = &NodeControllerConfig{}
				sc.Node = &v3.NodeControllerConfig{}
			case "serviceaccount":
				rc.ServiceAccount = &GenericControllerConfig{}
				sc.ServiceAccount = &v3.ServiceAccountControllerConfig{}
			case "flannelmigration":
				log.WithField(EnvEnabledControllers, v).Fatal("cannot run flannelmigration with other controllers")
			// Calico Enterprise controllers
			case "service":
				rc.Service = &GenericControllerConfig{}
				// service not supported on KubeControllersConfiguration yet
			case "federatedservices":
				rc.FederatedServices = &GenericControllerConfig{}
				sc.FederatedServices = &v3.FederatedServicesControllerConfig{}
			case "elasticsearchconfiguration":
				rc.ElasticsearchConfiguration = &ElasticsearchCfgControllerCfg{}
				// elasticsearchconfiguration not supported on KubeControllersConfiguration yet
			case "managedcluster":
				rc.ManagedCluster = &ManagedClusterControllerConfig{}
				// managed cluster not supported on KubeControllersConfiguration yet
			case "authorization":
				rc.AuthorizationConfiguration = &AuthorizationControllerCfg{}
				// authorization not supported on KubeControllersConfiguration yet
			default:
				log.Fatalf("Invalid controller '%s' provided.", controllerType)
			}
		}
	} else {
		// No environment variable, use API
		if n != nil {
			rc.Node = &NodeControllerConfig{}
			sc.Node = &v3.NodeControllerConfig{}

			// NOTE: Node controller doesn't use a cache, so doesn't use reconciler period
			sc.Node.ReconcilerPeriod = nil

			// SyncLabels and AutoHostEndpoint are handled later with their
			// corresponding environment variables
		}

		if pol != nil {
			rc.Policy = &GenericControllerConfig{}
			sc.Policy = &v3.PolicyControllerConfig{}
		}

		if w != nil {
			rc.WorkloadEndpoint = &GenericControllerConfig{}
			sc.WorkloadEndpoint = &v3.WorkloadEndpointControllerConfig{}
		}

		if s != nil {
			rc.ServiceAccount = &GenericControllerConfig{}
			sc.ServiceAccount = &v3.ServiceAccountControllerConfig{}
		}

		if ns != nil {
			rc.Namespace = &GenericControllerConfig{}
			sc.Namespace = &v3.NamespaceControllerConfig{}
		}

		if f != nil {
			rc.FederatedServices = &GenericControllerConfig{}
			sc.FederatedServices = &v3.FederatedServicesControllerConfig{}
		}
	}

	// Set reconciler periods, if enabled
	if rc.Policy != nil && pol != nil {
		if pol.ReconcilerPeriod == nil {
			rc.Policy.ReconcilerPeriod = time.Minute * 5
		} else {
			rc.Policy.ReconcilerPeriod = pol.ReconcilerPeriod.Duration
		}
		sc.Policy.ReconcilerPeriod = pol.ReconcilerPeriod
	}
	if rc.WorkloadEndpoint != nil && w != nil {
		if w.ReconcilerPeriod == nil {
			rc.WorkloadEndpoint.ReconcilerPeriod = time.Minute * 5
		} else {
			rc.WorkloadEndpoint.ReconcilerPeriod = w.ReconcilerPeriod.Duration
		}
		sc.WorkloadEndpoint.ReconcilerPeriod = w.ReconcilerPeriod
	}
	if rc.Namespace != nil && ns != nil {
		if ns.ReconcilerPeriod == nil {
			rc.Namespace.ReconcilerPeriod = time.Minute * 5
		} else {
			rc.Namespace.ReconcilerPeriod = ns.ReconcilerPeriod.Duration
		}
		sc.Namespace.ReconcilerPeriod = ns.ReconcilerPeriod
	}
	if rc.ServiceAccount != nil && s != nil {
		if s.ReconcilerPeriod == nil {
			rc.ServiceAccount.ReconcilerPeriod = time.Minute * 5
		} else {
			rc.ServiceAccount.ReconcilerPeriod = s.ReconcilerPeriod.Duration
		}
		sc.ServiceAccount.ReconcilerPeriod = s.ReconcilerPeriod
	}
	if rc.FederatedServices != nil && f != nil {
		if f.ReconcilerPeriod == nil {
			rc.FederatedServices.ReconcilerPeriod = time.Minute * 5
		} else {
			rc.FederatedServices.ReconcilerPeriod = f.ReconcilerPeriod.Duration
		}
		sc.FederatedServices.ReconcilerPeriod = f.ReconcilerPeriod
	}
}

func mergeLogLevel(envVars map[string]string, status *v3.KubeControllersConfigurationStatus, rCfg *RunConfig, apiCfg v3.KubeControllersConfigurationSpec) {
	v, p := envVars[EnvLogLevel]
	if p {
		status.EnvironmentVars[EnvLogLevel] = v
		l, err := log.ParseLevel(v)
		if err != nil {
			log.WithField(EnvLogLevel, v).Fatal("invalid environment variable value")
		}
		rCfg.LogLevelScreen = l
	} else {
		// No environment variable, check API
		l, err := log.ParseLevel(apiCfg.LogSeverityScreen)
		if err == nil {
			// API valid
			rCfg.LogLevelScreen = l
		} else {
			// API invalid, use default
			log.WithField("LOG_LEVEL", apiCfg.LogSeverityScreen).Warn("unknown log level, using Info")
			rCfg.LogLevelScreen = log.InfoLevel
		}
	}
	status.RunningConfig.LogSeverityScreen = strings.Title(rCfg.LogLevelScreen.String())
}
