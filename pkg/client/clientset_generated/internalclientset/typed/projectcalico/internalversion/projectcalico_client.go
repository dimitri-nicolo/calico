// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	"github.com/tigera/apiserver/pkg/client/clientset_generated/internalclientset/scheme"
	rest "k8s.io/client-go/rest"
)

type ProjectcalicoInterface interface {
	RESTClient() rest.Interface
	AuthenticationReviewsGetter
	AuthorizationReviewsGetter
	BGPConfigurationsGetter
	BGPPeersGetter
	ClusterInformationsGetter
	FelixConfigurationsGetter
	GlobalAlertsGetter
	GlobalAlertTemplatesGetter
	GlobalNetworkPoliciesGetter
	GlobalNetworkSetsGetter
	GlobalReportsGetter
	GlobalReportTypesGetter
	GlobalThreatFeedsGetter
	HostEndpointsGetter
	IPPoolsGetter
	KubeControllersConfigurationsGetter
	LicenseKeysGetter
	ManagedClustersGetter
	NetworkPoliciesGetter
	NetworkSetsGetter
	ProfilesGetter
	RemoteClusterConfigurationsGetter
	StagedGlobalNetworkPoliciesGetter
	StagedKubernetesNetworkPoliciesGetter
	StagedNetworkPoliciesGetter
	TiersGetter
}

// ProjectcalicoClient is used to interact with features provided by the projectcalico.org group.
type ProjectcalicoClient struct {
	restClient rest.Interface
}

func (c *ProjectcalicoClient) AuthenticationReviews() AuthenticationReviewInterface {
	return newAuthenticationReviews(c)
}

func (c *ProjectcalicoClient) AuthorizationReviews() AuthorizationReviewInterface {
	return newAuthorizationReviews(c)
}

func (c *ProjectcalicoClient) BGPConfigurations() BGPConfigurationInterface {
	return newBGPConfigurations(c)
}

func (c *ProjectcalicoClient) BGPPeers() BGPPeerInterface {
	return newBGPPeers(c)
}

func (c *ProjectcalicoClient) ClusterInformations() ClusterInformationInterface {
	return newClusterInformations(c)
}

func (c *ProjectcalicoClient) FelixConfigurations() FelixConfigurationInterface {
	return newFelixConfigurations(c)
}

func (c *ProjectcalicoClient) GlobalAlerts() GlobalAlertInterface {
	return newGlobalAlerts(c)
}

func (c *ProjectcalicoClient) GlobalAlertTemplates() GlobalAlertTemplateInterface {
	return newGlobalAlertTemplates(c)
}

func (c *ProjectcalicoClient) GlobalNetworkPolicies() GlobalNetworkPolicyInterface {
	return newGlobalNetworkPolicies(c)
}

func (c *ProjectcalicoClient) GlobalNetworkSets() GlobalNetworkSetInterface {
	return newGlobalNetworkSets(c)
}

func (c *ProjectcalicoClient) GlobalReports() GlobalReportInterface {
	return newGlobalReports(c)
}

func (c *ProjectcalicoClient) GlobalReportTypes() GlobalReportTypeInterface {
	return newGlobalReportTypes(c)
}

func (c *ProjectcalicoClient) GlobalThreatFeeds() GlobalThreatFeedInterface {
	return newGlobalThreatFeeds(c)
}

func (c *ProjectcalicoClient) HostEndpoints() HostEndpointInterface {
	return newHostEndpoints(c)
}

func (c *ProjectcalicoClient) IPPools() IPPoolInterface {
	return newIPPools(c)
}

func (c *ProjectcalicoClient) KubeControllersConfigurations() KubeControllersConfigurationInterface {
	return newKubeControllersConfigurations(c)
}

func (c *ProjectcalicoClient) LicenseKeys() LicenseKeyInterface {
	return newLicenseKeys(c)
}

func (c *ProjectcalicoClient) ManagedClusters() ManagedClusterInterface {
	return newManagedClusters(c)
}

func (c *ProjectcalicoClient) NetworkPolicies(namespace string) NetworkPolicyInterface {
	return newNetworkPolicies(c, namespace)
}

func (c *ProjectcalicoClient) NetworkSets() NetworkSetInterface {
	return newNetworkSets(c)
}

func (c *ProjectcalicoClient) Profiles() ProfileInterface {
	return newProfiles(c)
}

func (c *ProjectcalicoClient) RemoteClusterConfigurations() RemoteClusterConfigurationInterface {
	return newRemoteClusterConfigurations(c)
}

func (c *ProjectcalicoClient) StagedGlobalNetworkPolicies() StagedGlobalNetworkPolicyInterface {
	return newStagedGlobalNetworkPolicies(c)
}

func (c *ProjectcalicoClient) StagedKubernetesNetworkPolicies(namespace string) StagedKubernetesNetworkPolicyInterface {
	return newStagedKubernetesNetworkPolicies(c, namespace)
}

func (c *ProjectcalicoClient) StagedNetworkPolicies(namespace string) StagedNetworkPolicyInterface {
	return newStagedNetworkPolicies(c, namespace)
}

func (c *ProjectcalicoClient) Tiers() TierInterface {
	return newTiers(c)
}

// NewForConfig creates a new ProjectcalicoClient for the given config.
func NewForConfig(c *rest.Config) (*ProjectcalicoClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ProjectcalicoClient{client}, nil
}

// NewForConfigOrDie creates a new ProjectcalicoClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *ProjectcalicoClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ProjectcalicoClient for the given RESTClient.
func New(c rest.Interface) *ProjectcalicoClient {
	return &ProjectcalicoClient{c}
}

func setConfigDefaults(config *rest.Config) error {
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	if config.GroupVersion == nil || config.GroupVersion.Group != scheme.Scheme.PrioritizedVersionsForGroup("projectcalico.org")[0].Group {
		gv := scheme.Scheme.PrioritizedVersionsForGroup("projectcalico.org")[0]
		config.GroupVersion = &gv
	}
	config.NegotiatedSerializer = scheme.Codecs

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *ProjectcalicoClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
