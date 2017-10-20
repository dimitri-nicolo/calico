/*
Copyright 2017 Tigera.
*/package fake

import (
	internalversion "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset/typed/projectcalico/internalversion"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeProjectcalico struct {
	*testing.Fake
}

func (c *FakeProjectcalico) GlobalNetworkPolicies() internalversion.GlobalNetworkPolicyInterface {
	return &FakeGlobalNetworkPolicies{c}
}

func (c *FakeProjectcalico) NetworkPolicies(namespace string) internalversion.NetworkPolicyInterface {
	return &FakeNetworkPolicies{c, namespace}
}

func (c *FakeProjectcalico) Tiers() internalversion.TierInterface {
	return &FakeTiers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeProjectcalico) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
