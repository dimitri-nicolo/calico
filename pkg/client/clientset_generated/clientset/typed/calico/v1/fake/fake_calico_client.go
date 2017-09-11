/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	v1 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/calico/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeCalicoV1 struct {
	*testing.Fake
}

func (c *FakeCalicoV1) Endpoints() v1.EndpointInterface {
	return &FakeEndpoints{c}
}

func (c *FakeCalicoV1) Nodes() v1.NodeInterface {
	return &FakeNodes{c}
}

func (c *FakeCalicoV1) Policies(namespace string) v1.PolicyInterface {
	return &FakePolicies{c, namespace}
}

func (c *FakeCalicoV1) Tiers() v1.TierInterface {
	return &FakeTiers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCalicoV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
