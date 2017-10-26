/*
Copyright 2017 Tigera.
*/package fake

import (
	v2 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v2"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeProjectcalicoV2 struct {
	*testing.Fake
}

func (c *FakeProjectcalicoV2) GlobalNetworkPolicies() v2.GlobalNetworkPolicyInterface {
	return &FakeGlobalNetworkPolicies{c}
}

func (c *FakeProjectcalicoV2) NetworkPolicies(namespace string) v2.NetworkPolicyInterface {
	return &FakeNetworkPolicies{c, namespace}
}

func (c *FakeProjectcalicoV2) Tiers() v2.TierInterface {
	return &FakeTiers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeProjectcalicoV2) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
