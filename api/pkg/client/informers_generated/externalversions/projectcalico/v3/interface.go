// Copyright (c) 2022 Tigera, Inc. All rights reserved.

// Code generated by informer-gen. DO NOT EDIT.

package v3

import (
	internalinterfaces "github.com/tigera/api/pkg/client/informers_generated/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// AlertExceptions returns a AlertExceptionInformer.
	AlertExceptions() AlertExceptionInformer
	// AuthenticationReviews returns a AuthenticationReviewInformer.
	AuthenticationReviews() AuthenticationReviewInformer
	// AuthorizationReviews returns a AuthorizationReviewInformer.
	AuthorizationReviews() AuthorizationReviewInformer
	// BGPConfigurations returns a BGPConfigurationInformer.
	BGPConfigurations() BGPConfigurationInformer
	// BGPPeers returns a BGPPeerInformer.
	BGPPeers() BGPPeerInformer
	// CalicoNodeStatuses returns a CalicoNodeStatusInformer.
	CalicoNodeStatuses() CalicoNodeStatusInformer
	// ClusterInformations returns a ClusterInformationInformer.
	ClusterInformations() ClusterInformationInformer
	// DeepPacketInspections returns a DeepPacketInspectionInformer.
	DeepPacketInspections() DeepPacketInspectionInformer
	// FelixConfigurations returns a FelixConfigurationInformer.
	FelixConfigurations() FelixConfigurationInformer
	// GlobalAlerts returns a GlobalAlertInformer.
	GlobalAlerts() GlobalAlertInformer
	// GlobalAlertTemplates returns a GlobalAlertTemplateInformer.
	GlobalAlertTemplates() GlobalAlertTemplateInformer
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
	// IPReservations returns a IPReservationInformer.
	IPReservations() IPReservationInformer
	// KubeControllersConfigurations returns a KubeControllersConfigurationInformer.
	KubeControllersConfigurations() KubeControllersConfigurationInformer
	// LicenseKeys returns a LicenseKeyInformer.
	LicenseKeys() LicenseKeyInformer
	// ManagedClusters returns a ManagedClusterInformer.
	ManagedClusters() ManagedClusterInformer
	// NetworkPolicies returns a NetworkPolicyInformer.
	NetworkPolicies() NetworkPolicyInformer
	// NetworkSets returns a NetworkSetInformer.
	NetworkSets() NetworkSetInformer
	// PacketCaptures returns a PacketCaptureInformer.
	PacketCaptures() PacketCaptureInformer
	// Profiles returns a ProfileInformer.
	Profiles() ProfileInformer
	// RemoteClusterConfigurations returns a RemoteClusterConfigurationInformer.
	RemoteClusterConfigurations() RemoteClusterConfigurationInformer
	// StagedGlobalNetworkPolicies returns a StagedGlobalNetworkPolicyInformer.
	StagedGlobalNetworkPolicies() StagedGlobalNetworkPolicyInformer
	// StagedKubernetesNetworkPolicies returns a StagedKubernetesNetworkPolicyInformer.
	StagedKubernetesNetworkPolicies() StagedKubernetesNetworkPolicyInformer
	// StagedNetworkPolicies returns a StagedNetworkPolicyInformer.
	StagedNetworkPolicies() StagedNetworkPolicyInformer
	// Tiers returns a TierInformer.
	Tiers() TierInformer
	// UISettings returns a UISettingsInformer.
	UISettings() UISettingsInformer
	// UISettingsGroups returns a UISettingsGroupInformer.
	UISettingsGroups() UISettingsGroupInformer
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

// AlertExceptions returns a AlertExceptionInformer.
func (v *version) AlertExceptions() AlertExceptionInformer {
	return &alertExceptionInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// AuthenticationReviews returns a AuthenticationReviewInformer.
func (v *version) AuthenticationReviews() AuthenticationReviewInformer {
	return &authenticationReviewInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// AuthorizationReviews returns a AuthorizationReviewInformer.
func (v *version) AuthorizationReviews() AuthorizationReviewInformer {
	return &authorizationReviewInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// BGPConfigurations returns a BGPConfigurationInformer.
func (v *version) BGPConfigurations() BGPConfigurationInformer {
	return &bGPConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// BGPPeers returns a BGPPeerInformer.
func (v *version) BGPPeers() BGPPeerInformer {
	return &bGPPeerInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// CalicoNodeStatuses returns a CalicoNodeStatusInformer.
func (v *version) CalicoNodeStatuses() CalicoNodeStatusInformer {
	return &calicoNodeStatusInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterInformations returns a ClusterInformationInformer.
func (v *version) ClusterInformations() ClusterInformationInformer {
	return &clusterInformationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// DeepPacketInspections returns a DeepPacketInspectionInformer.
func (v *version) DeepPacketInspections() DeepPacketInspectionInformer {
	return &deepPacketInspectionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// FelixConfigurations returns a FelixConfigurationInformer.
func (v *version) FelixConfigurations() FelixConfigurationInformer {
	return &felixConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalAlerts returns a GlobalAlertInformer.
func (v *version) GlobalAlerts() GlobalAlertInformer {
	return &globalAlertInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// GlobalAlertTemplates returns a GlobalAlertTemplateInformer.
func (v *version) GlobalAlertTemplates() GlobalAlertTemplateInformer {
	return &globalAlertTemplateInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
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

// IPReservations returns a IPReservationInformer.
func (v *version) IPReservations() IPReservationInformer {
	return &iPReservationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// KubeControllersConfigurations returns a KubeControllersConfigurationInformer.
func (v *version) KubeControllersConfigurations() KubeControllersConfigurationInformer {
	return &kubeControllersConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
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
	return &networkSetInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// PacketCaptures returns a PacketCaptureInformer.
func (v *version) PacketCaptures() PacketCaptureInformer {
	return &packetCaptureInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Profiles returns a ProfileInformer.
func (v *version) Profiles() ProfileInformer {
	return &profileInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// RemoteClusterConfigurations returns a RemoteClusterConfigurationInformer.
func (v *version) RemoteClusterConfigurations() RemoteClusterConfigurationInformer {
	return &remoteClusterConfigurationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// StagedGlobalNetworkPolicies returns a StagedGlobalNetworkPolicyInformer.
func (v *version) StagedGlobalNetworkPolicies() StagedGlobalNetworkPolicyInformer {
	return &stagedGlobalNetworkPolicyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// StagedKubernetesNetworkPolicies returns a StagedKubernetesNetworkPolicyInformer.
func (v *version) StagedKubernetesNetworkPolicies() StagedKubernetesNetworkPolicyInformer {
	return &stagedKubernetesNetworkPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// StagedNetworkPolicies returns a StagedNetworkPolicyInformer.
func (v *version) StagedNetworkPolicies() StagedNetworkPolicyInformer {
	return &stagedNetworkPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Tiers returns a TierInformer.
func (v *version) Tiers() TierInformer {
	return &tierInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// UISettings returns a UISettingsInformer.
func (v *version) UISettings() UISettingsInformer {
	return &uISettingsInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// UISettingsGroups returns a UISettingsGroupInformer.
func (v *version) UISettingsGroups() UISettingsGroupInformer {
	return &uISettingsGroupInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
