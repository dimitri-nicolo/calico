// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeProjectcalicoV3 struct {
	*testing.Fake
}

func (c *FakeProjectcalicoV3) BGPConfigurations() v3.BGPConfigurationInterface {
	return &FakeBGPConfigurations{c}
}

func (c *FakeProjectcalicoV3) BGPPeers() v3.BGPPeerInterface {
	return &FakeBGPPeers{c}
}

func (c *FakeProjectcalicoV3) GlobalNetworkPolicies() v3.GlobalNetworkPolicyInterface {
	return &FakeGlobalNetworkPolicies{c}
}

func (c *FakeProjectcalicoV3) GlobalNetworkSets() v3.GlobalNetworkSetInterface {
	return &FakeGlobalNetworkSets{c}
}

func (c *FakeProjectcalicoV3) GlobalReports() v3.GlobalReportInterface {
	return &FakeGlobalReports{c}
}

func (c *FakeProjectcalicoV3) GlobalReportTypes() v3.GlobalReportTypeInterface {
	return &FakeGlobalReportTypes{c}
}

func (c *FakeProjectcalicoV3) GlobalThreatFeeds() v3.GlobalThreatFeedInterface {
	return &FakeGlobalThreatFeeds{c}
}

func (c *FakeProjectcalicoV3) HostEndpoints() v3.HostEndpointInterface {
	return &FakeHostEndpoints{c}
}

func (c *FakeProjectcalicoV3) IPPools() v3.IPPoolInterface {
	return &FakeIPPools{c}
}

func (c *FakeProjectcalicoV3) LicenseKeys() v3.LicenseKeyInterface {
	return &FakeLicenseKeys{c}
}

func (c *FakeProjectcalicoV3) NetworkPolicies(namespace string) v3.NetworkPolicyInterface {
	return &FakeNetworkPolicies{c, namespace}
}

func (c *FakeProjectcalicoV3) Tiers() v3.TierInterface {
	return &FakeTiers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeProjectcalicoV3) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
