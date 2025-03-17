// Copyright (c) 2018,2020 Tigera, Inc. All rights reserved.

package federationsyncer

import (
	"reflect"

	log "github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/k8s"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/remotecluster"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/watchersyncer"
)

const (
	calicoClientID = "calico"
	k8sClientID    = "ks"
)

var emptyDatastoreConfig = apiconfig.NewCalicoAPIConfig()

// New creates a new federation syncer. This particular syncer requires both Calico datastore access and Kubernetes

func New(calicoClient api.Client, k8sClientset *kubernetes.Clientset, callbacks api.SyncerCallbacks) api.Syncer {
	k8sServicesClient := k8s.NewK8sResourceWrapperClient(k8sClientset)
	// The resources in this syncer are backed by two different clients, so we specify which client for each
	// resource type.
	clients := map[string]api.Client{
		calicoClientID: calicoClient,
		k8sClientID:    k8sServicesClient,
	}
	resourceTypes := []watchersyncer.ResourceType{
		{
			ListInterface:   model.ResourceListOptions{Kind: model.KindKubernetesService},
			UpdateProcessor: nil,         // No need to process the updates so pass nil
			ClientID:        k8sClientID, // This is backed by the kubernetes wrapped client
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindK8sEndpoints},
			UpdateProcessor: nil,         // No need to process the updates so pass nil
			ClientID:        k8sClientID, // This is backed by the kubernetes wrapped client
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindRemoteClusterConfiguration},
			UpdateProcessor: nil,            // No need to process the updates so pass nil
			ClientID:        calicoClientID, // This is backed by the calico client
		},
	}

	// The "main" watchersyncer will spawn additional watchersyncers for any remote clusters that are found.
	// The callbacks are wrapped to allow the messages to be intercepted so that the additional watchersyncers can be spawned.
	return watchersyncer.NewMultiClient(
		clients,
		resourceTypes,
		remotecluster.NewWrappedCallbacks(callbacks, k8sClientset, federationRemoteClusterProcessor{}),
	)
}

// federationRemoteClusterProcessor provides the service syncer specific remote cluster processing.
type federationRemoteClusterProcessor struct{}

func (_ federationRemoteClusterProcessor) CreateResourceTypes(overlayRoutingMode apiv3.OverlayRoutingMode, usePodCIDR bool) []watchersyncer.ResourceType {
	return []watchersyncer.ResourceType{
		{
			ListInterface:         model.ResourceListOptions{Kind: model.KindKubernetesService},
			UpdateProcessor:       nil,  // No need to process the updates so pass nil
			SendDeletesOnConnFail: true, // If the connection fails, treat as if the services are not there.
		},
		{
			ListInterface:         model.ResourceListOptions{Kind: apiv3.KindK8sEndpoints},
			UpdateProcessor:       nil,  // No need to process the updates so pass nil
			SendDeletesOnConnFail: true, // If the connection fails, treat as if the endpoints are not there.
		},
	}
}

// ConvertUpdates converts each ResourceKey in the updates to be a RemoteClusterResourceKey, so that the
// cluster name is available to the syncer consumer.
func (_ federationRemoteClusterProcessor) ConvertUpdates(clusterName string, updates []api.Update) []api.Update {
	for i := range updates {
		rk := updates[i].Key.(model.ResourceKey)
		rcrk := model.RemoteClusterResourceKey{
			Cluster:     clusterName,
			ResourceKey: rk,
		}
		updates[i].Key = rcrk
	}

	return updates
}

// GetCalicoAPIConfig returns only a kubernetes backed config, or nil if is no Kubernetes configuration
// specified at all.
func (_ federationRemoteClusterProcessor) GetCalicoAPIConfig(config *apiv3.RemoteClusterConfiguration) *apiconfig.CalicoAPIConfig {
	// The remote services syncer requires the kubernetes access details, even if the Calico datastore
	// is etcd. Extract the kubernetes data from the config.
	datastoreConfig := apiconfig.NewCalicoAPIConfig()
	datastoreConfig.Spec.Kubeconfig = config.Spec.Kubeconfig
	datastoreConfig.Spec.K8sAPIEndpoint = config.Spec.K8sAPIEndpoint
	datastoreConfig.Spec.K8sKeyFile = config.Spec.K8sKeyFile
	datastoreConfig.Spec.K8sCertFile = config.Spec.K8sCertFile
	datastoreConfig.Spec.K8sCAFile = config.Spec.K8sCAFile
	datastoreConfig.Spec.K8sAPIToken = config.Spec.K8sAPIToken
	datastoreConfig.Spec.K8sInsecureSkipTLSVerify = config.Spec.K8sInsecureSkipTLSVerify
	datastoreConfig.Spec.KubeconfigInline = config.Spec.KubeconfigInline

	// If the datastore config is still empty then the RCC did not contain any kubernetes data,
	// we cannot use this RCC in the syncer.
	if reflect.DeepEqual(datastoreConfig, emptyDatastoreConfig) {
		log.Warningf("RemoteClusterConfiguration(%s) does not contain valid Kubernetes API configuration. "+
			"Service data will not be included from this cluster.", config.Name)
		return nil
	}

	datastoreConfig.Spec.DatastoreType = apiconfig.Kubernetes
	return datastoreConfig
}

// CreateClient creates the private backend client which is used to sync Kubernetes Services and Endpoints.
func (_ federationRemoteClusterProcessor) CreateClient(config apiconfig.CalicoAPIConfig) (api.Client, error) {
	_, cs, err := k8s.CreateKubernetesClientset(&config.Spec)
	if err != nil {
		return nil, err
	}
	return k8s.NewK8sResourceWrapperClient(cs), nil
}
