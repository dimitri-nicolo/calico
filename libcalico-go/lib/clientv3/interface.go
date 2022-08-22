// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.

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

package clientv3

import (
	"context"

	"github.com/projectcalico/calico/libcalico-go/lib/ipam"
)

type Interface interface {
	// Nodes returns an interface for managing node resources.
	Nodes() NodeInterface
	// GlobalNetworkPolicies returns an interface for managing global network policy resources.
	GlobalNetworkPolicies() GlobalNetworkPolicyInterface
	// NetworkPolicies returns an interface for managing namespaced network policy resources.
	NetworkPolicies() NetworkPolicyInterface
	// StagedGlobalNetworkPolicies returns an interface for managing staged global network policy resources.
	StagedGlobalNetworkPolicies() StagedGlobalNetworkPolicyInterface
	// StagedNetworkPolicies returns an interface for managing staged namespaced network policy resources.
	StagedNetworkPolicies() StagedNetworkPolicyInterface
	// StagedKubernetesNetworkPolicies returns an interface for managing staged kubernetes network policy resources.
	StagedKubernetesNetworkPolicies() StagedKubernetesNetworkPolicyInterface
	// IPPools returns an interface for managing IP pool resources.
	IPPools() IPPoolInterface
	// IPReservations returns an interface for managing IP reservation resources.
	IPReservations() IPReservationInterface
	// Profiles returns an interface for managing profile resources.
	Profiles() ProfileInterface
	// GlobalNetworkSets returns an interface for managing global network sets resources.
	GlobalNetworkSets() GlobalNetworkSetInterface
	// NetworkSets returns an interface for managing network sets resources.
	NetworkSets() NetworkSetInterface
	// HostEndpoints returns an interface for managing host endpoint resources.
	HostEndpoints() HostEndpointInterface
	// WorkloadEndpoints returns an interface for managing workload endpoint resources.
	WorkloadEndpoints() WorkloadEndpointInterface
	// BGPPeers returns an interface for managing BGP peer resources.
	BGPPeers() BGPPeerInterface
	// Tiers returns an interface for managing tier resources.
	Tiers() TierInterface
	// UISettingsGroups returns an interface for managing uisettingsgroup resources.
	UISettingsGroups() UISettingsGroupInterface
	// UISettings returns an interface for managing uisettings resources.
	UISettings() UISettingsInterface
	// IPAM returns an interface for managing IP address assignment and releasing.
	IPAM() ipam.Interface
	// BGPConfigurations returns an interface for managing the BGP configuration resources.
	BGPConfigurations() BGPConfigurationInterface
	// FelixConfigurations returns an interface for managing the Felix configuration resources.
	FelixConfigurations() FelixConfigurationInterface
	// ClusterInformation returns an interface for managing the cluster information resource.
	ClusterInformation() ClusterInformationInterface
	// KubeControllersConfiguration returns an interface for managing the
	// KubeControllersConfiguration resource.
	KubeControllersConfiguration() KubeControllersConfigurationInterface
	// RemoteClusterConfiguration returns an interface for managing the remote cluster configuration resources.
	RemoteClusterConfigurations() RemoteClusterConfigurationInterface
	// LicenseKey returns an interface for managing the license key resource.
	LicenseKey() LicenseKeyInterface
	// AlertExceptions returns an interface for managing the alert exception resource.
	AlertExceptions() AlertExceptionInterface
	// GlobalAlerts returns an interface for managing the alert resources.
	GlobalAlerts() GlobalAlertInterface
	// GlobalAlertTemplates returns an interface for managing the alert template resources.
	GlobalAlertTemplates() GlobalAlertTemplateInterface
	// GlobalThreatFeeds returns an interface for managing the Threat feed resources.
	GlobalThreatFeeds() GlobalThreatFeedInterface
	// GlobalReportTypes returns an interface for managing the report types resources.
	GlobalReportTypes() GlobalReportTypeInterface
	// GlobalReports returns an interface for managing the report resources.
	GlobalReports() GlobalReportInterface
	// ManagedClusters returns an interface for managing the managed cluster resources.
	ManagedClusters() ManagedClusterInterface
	// PacketCaptures returns an interface for managing the packet capture resources.
	PacketCaptures() PacketCaptureInterface
	// DeepPacketInspections returns an interface for managing the DPI resources.
	DeepPacketInspections() DeepPacketInspectionInterface
	// CalicoNodeStatus returns an interface for managing CalicoNodeStatus resources.
	CalicoNodeStatus() CalicoNodeStatusInterface
	// IPAMConfig returns an interface for managing IPAMConfig resources.
	IPAMConfig() IPAMConfigInterface
	// BlockAffinities returns an interface for viewing IPAM block affinity resources.
	BlockAffinities() BlockAffinityInterface

	// EnsureInitialized is used to ensure the backend datastore is correctly
	// initialized for use by Calico.  This method may be called multiple times, and
	// will have no effect if the datastore is already correctly initialized.
	// Most Calico deployment scenarios will automatically implicitly invoke this
	// method and so a general consumer of this API can assume that the datastore
	// is already initialized.
	EnsureInitialized(ctx context.Context, calicoVersion, cnxVersion, clusterType string) error
}

// Compile-time assertion that our client implements its interface.
var _ Interface = (*client)(nil)
