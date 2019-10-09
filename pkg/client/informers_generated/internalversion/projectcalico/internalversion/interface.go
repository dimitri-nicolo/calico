// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	internalinterfaces "github.com/tigera/calico-k8sapiserver/pkg/client/informers_generated/internalversion/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// BGPConfigurations returns a BGPConfigurationInformer.
	BGPConfigurations() BGPConfigurationInformer
	// BGPPeers returns a BGPPeerInformer.
	BGPPeers() BGPPeerInformer
	// ClusterInformations returns a ClusterInformationInformer.
	ClusterInformations() ClusterInformationInformer
	// FelixConfigurations returns a FelixConfigurationInformer.
	FelixConfigurations() FelixConfigurationInformer
	// GlobalAlerts returns a GlobalAlertInformer.
	GlobalAlerts() GlobalAlertInformer
	// GlobalNetworkPolicies returns a GlobalNetworkPolicyInformer.
	GlobalNetworkPolicies() GlobalNetworkPolicyInformer
	// GlobalNetworkSets returns a GlobalNetworkSetInformer.
	GlobalNetworkSets() GlobalNetworkSetInformer
	// GlobalReports returns a GlobalReportInformer.
	GlobalReports() GlobalReportInformer
	// GlobalReportTypes returns a GlobalReportTypeInformer.
	GlobalReportTypes() GlobalReportTypeInformer
	// GlobalThreatFeeds returns a GlobalThreatFeedInformer.
	GlobalThreatFeeds() GlobalThreatFeedInformer
	// HostEndpoints returns a HostEndpointInformer.
	HostEndpoints() HostEndpointInformer
	// IPPools returns a IPPoolInformer.
	IPPools() IPPoolInformer
	// LicenseKeys returns a LicenseKeyInformer.
	LicenseKeys() LicenseKeyInformer
	// ManagedClusters returns a ManagedClusterInformer.
	ManagedClusters() ManagedClusterInformer
	// NetworkPolicies returns a NetworkPolicyInformer.
	NetworkPolicies() NetworkPolicyInformer
	// NetworkSets returns a NetworkSetInformer.
	NetworkSets() NetworkSetInformer
	// Profiles returns a ProfileInformer.
	Profiles() ProfileInformer
	// RemoteClusterConfigurations returns a RemoteClusterConfigurationInformer.
	RemoteClusterConfigurations() RemoteClusterConfigurationInformer
	// Tiers returns a TierInformer.
	Tiers() TierInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// BGPConfigurations returns a BGPConfigurationInformer.
func (v *version) BGPConfigurations() BGPConfigurationInformer {
	return &bGPConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// BGPPeers returns a BGPPeerInformer.
func (v *version) BGPPeers() BGPPeerInformer {
	return &bGPPeerInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterInformations returns a ClusterInformationInformer.
func (v *version) ClusterInformations() ClusterInformationInformer {
	return &clusterInformationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// FelixConfigurations returns a FelixConfigurationInformer.
func (v *version) FelixConfigurations() FelixConfigurationInformer {
	return &felixConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalAlerts returns a GlobalAlertInformer.
func (v *version) GlobalAlerts() GlobalAlertInformer {
	return &globalAlertInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalNetworkPolicies returns a GlobalNetworkPolicyInformer.
func (v *version) GlobalNetworkPolicies() GlobalNetworkPolicyInformer {
	return &globalNetworkPolicyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalNetworkSets returns a GlobalNetworkSetInformer.
func (v *version) GlobalNetworkSets() GlobalNetworkSetInformer {
	return &globalNetworkSetInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalReports returns a GlobalReportInformer.
func (v *version) GlobalReports() GlobalReportInformer {
	return &globalReportInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalReportTypes returns a GlobalReportTypeInformer.
func (v *version) GlobalReportTypes() GlobalReportTypeInformer {
	return &globalReportTypeInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalThreatFeeds returns a GlobalThreatFeedInformer.
func (v *version) GlobalThreatFeeds() GlobalThreatFeedInformer {
	return &globalThreatFeedInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// HostEndpoints returns a HostEndpointInformer.
func (v *version) HostEndpoints() HostEndpointInformer {
	return &hostEndpointInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// IPPools returns a IPPoolInformer.
func (v *version) IPPools() IPPoolInformer {
	return &iPPoolInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// LicenseKeys returns a LicenseKeyInformer.
func (v *version) LicenseKeys() LicenseKeyInformer {
	return &licenseKeyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ManagedClusters returns a ManagedClusterInformer.
func (v *version) ManagedClusters() ManagedClusterInformer {
	return &managedClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// NetworkPolicies returns a NetworkPolicyInformer.
func (v *version) NetworkPolicies() NetworkPolicyInformer {
	return &networkPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// NetworkSets returns a NetworkSetInformer.
func (v *version) NetworkSets() NetworkSetInformer {
	return &networkSetInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Profiles returns a ProfileInformer.
func (v *version) Profiles() ProfileInformer {
	return &profileInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// RemoteClusterConfigurations returns a RemoteClusterConfigurationInformer.
func (v *version) RemoteClusterConfigurations() RemoteClusterConfigurationInformer {
	return &remoteClusterConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Tiers returns a TierInformer.
func (v *version) Tiers() TierInformer {
	return &tierInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
