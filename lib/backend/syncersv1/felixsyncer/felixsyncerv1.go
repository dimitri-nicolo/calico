// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

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

package felixsyncer

import (
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/remotecluster"
	"github.com/projectcalico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/libcalico-go/lib/backend/watchersyncer"
)

// New creates a new Felix v1 Syncer.
func New(client api.Client, cfg apiconfig.CalicoAPIConfigSpec, callbacks api.SyncerCallbacks) api.Syncer {
	// Create the set of ResourceTypes required for Felix.  Since the update processors
	// also cache state, we need to create individual ones per syncer rather than create
	// a common global set.
	resourceTypes := []watchersyncer.ResourceType{
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindClusterInformation},
			UpdateProcessor: updateprocessors.NewClusterInfoUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindLicenseKey},
			UpdateProcessor: updateprocessors.NewLicenseKeyUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindFelixConfiguration},
			UpdateProcessor: updateprocessors.NewFelixConfigUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindGlobalNetworkPolicy},
			UpdateProcessor: updateprocessors.NewGlobalNetworkPolicyUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindGlobalNetworkSet},
			UpdateProcessor: updateprocessors.NewGlobalNetworkSetUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindIPPool},
			UpdateProcessor: updateprocessors.NewIPPoolUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindNode},
			UpdateProcessor: updateprocessors.NewFelixNodeUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindProfile},
			UpdateProcessor: updateprocessors.NewProfileUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindWorkloadEndpoint},
			UpdateProcessor: updateprocessors.NewWorkloadEndpointUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindNetworkPolicy},
			UpdateProcessor: updateprocessors.NewNetworkPolicyUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindNetworkSet},
			UpdateProcessor: updateprocessors.NewNetworkSetUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindTier},
			UpdateProcessor: updateprocessors.NewTierUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindHostEndpoint},
			UpdateProcessor: updateprocessors.NewHostEndpointUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindRemoteClusterConfiguration},
			UpdateProcessor: nil, // No need to process the updates so pass nil
		},
	}

	// If using Calico IPAM, include IPAM resources the felix cares about.
	if !cfg.K8sUsePodCIDR {
		resourceTypes = append(resourceTypes, watchersyncer.ResourceType{ListInterface: model.BlockListOptions{}})
	}

	// The "main" watchersyncer will spawn additional watchersyncers for any remote clusters that are found.
	// The callbacks are wrapped to allow the messages to be intercepted so that the additional watchersyncers can be spawned.
	return watchersyncer.New(
		client,
		resourceTypes,
		remotecluster.NewWrappedCallbacks(callbacks, felixRemoteClusterProcessor{}),
	)
}

// felixRemoteClusterProcessor provides the Felix syncer specific remote cluster processing.
type felixRemoteClusterProcessor struct{}

func (_ felixRemoteClusterProcessor) CreateResourceTypes() []watchersyncer.ResourceType {
	return []watchersyncer.ResourceType{
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindWorkloadEndpoint},
			UpdateProcessor: updateprocessors.NewWorkloadEndpointUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindHostEndpoint},
			UpdateProcessor: updateprocessors.NewHostEndpointUpdateProcessor(),
		},
		{
			ListInterface:   model.ResourceListOptions{Kind: apiv3.KindProfile},
			UpdateProcessor: updateprocessors.NewProfileUpdateProcessor(),
		},
	}
}

func (_ felixRemoteClusterProcessor) ConvertUpdates(clusterName string, updates []api.Update) []api.Update {
	for i, update := range updates {
		if update.UpdateType == api.UpdateTypeKVUpdated || update.UpdateType == api.UpdateTypeKVNew {
			switch t := update.Key.(type) {
			default:
				log.Warnf("unexpected type %T\n", t)
			case model.HostEndpointKey:
				t.Hostname = clusterName + "/" + t.Hostname
				updates[i].Key = t
				for profileIndex, profile := range updates[i].Value.(*model.HostEndpoint).ProfileIDs {
					updates[i].Value.(*model.HostEndpoint).ProfileIDs[profileIndex] = clusterName + "/" + profile
				}
			case model.WorkloadEndpointKey:
				t.Hostname = clusterName + "/" + t.Hostname
				updates[i].Key = t
				for profileIndex, profile := range updates[i].Value.(*model.WorkloadEndpoint).ProfileIDs {
					updates[i].Value.(*model.WorkloadEndpoint).ProfileIDs[profileIndex] = clusterName + "/" + profile
				}
			case model.ProfileRulesKey:
				t.Name = clusterName + "/" + t.Name
				updates[i].Value.(*model.ProfileRules).InboundRules = []model.Rule{}
				updates[i].Value.(*model.ProfileRules).OutboundRules = []model.Rule{}
				updates[i].Key = t
			case model.ProfileLabelsKey:
				t.Name = clusterName + "/" + t.Name
				updates[i].Key = t
			case model.ProfileTagsKey:
				t.Name = clusterName + "/" + t.Name
				updates[i].Key = t
			}
		} else if update.UpdateType == api.UpdateTypeKVDeleted {
			switch t := update.Key.(type) {
			default:
				log.Warnf("unexpected type %T\n", t)
			case model.HostEndpointKey:
				t.Hostname = clusterName + "/" + t.Hostname
				updates[i].Key = t
			case model.WorkloadEndpointKey:
				t.Hostname = clusterName + "/" + t.Hostname
				updates[i].Key = t
			case model.ProfileRulesKey:
				t.Name = clusterName + "/" + t.Name
				updates[i].Key = t
			case model.ProfileLabelsKey:
				t.Name = clusterName + "/" + t.Name
				updates[i].Key = t
			case model.ProfileTagsKey:
				t.Name = clusterName + "/" + t.Name
				updates[i].Key = t
			}
		}
	}

	return updates
}

func (_ felixRemoteClusterProcessor) GetCalicoAPIConfig(config *apiv3.RemoteClusterConfiguration) *apiconfig.CalicoAPIConfig {
	datastoreConfig := apiconfig.NewCalicoAPIConfig()
	datastoreConfig.Spec.DatastoreType = apiconfig.DatastoreType(config.Spec.DatastoreType)
	switch datastoreConfig.Spec.DatastoreType {
	case apiconfig.EtcdV3:
		datastoreConfig.Spec.EtcdEndpoints = config.Spec.EtcdEndpoints
		datastoreConfig.Spec.EtcdUsername = config.Spec.EtcdUsername
		datastoreConfig.Spec.EtcdPassword = config.Spec.EtcdPassword
		datastoreConfig.Spec.EtcdKeyFile = config.Spec.EtcdKeyFile
		datastoreConfig.Spec.EtcdCertFile = config.Spec.EtcdCertFile
		datastoreConfig.Spec.EtcdCACertFile = config.Spec.EtcdCACertFile
		return datastoreConfig
	case apiconfig.Kubernetes:
		datastoreConfig.Spec.Kubeconfig = config.Spec.Kubeconfig
		datastoreConfig.Spec.K8sAPIEndpoint = config.Spec.K8sAPIEndpoint
		datastoreConfig.Spec.K8sKeyFile = config.Spec.K8sKeyFile
		datastoreConfig.Spec.K8sCertFile = config.Spec.K8sCertFile
		datastoreConfig.Spec.K8sCAFile = config.Spec.K8sCAFile
		datastoreConfig.Spec.K8sAPIToken = config.Spec.K8sAPIToken
		datastoreConfig.Spec.K8sInsecureSkipTLSVerify = config.Spec.K8sInsecureSkipTLSVerify
		return datastoreConfig
	}
	return nil
}
