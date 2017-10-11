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
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
	v1beta1 "k8s.io/kubernetes/pkg/client/clientset_generated/clientset/typed/extensions/v1beta1"
)

type FakeExtensionsV1beta1 struct {
	*testing.Fake
}

func (c *FakeExtensionsV1beta1) DaemonSets(namespace string) v1beta1.DaemonSetInterface {
	return &FakeDaemonSets{c, namespace}
}

func (c *FakeExtensionsV1beta1) Deployments(namespace string) v1beta1.DeploymentInterface {
	return &FakeDeployments{c, namespace}
}

func (c *FakeExtensionsV1beta1) Ingresses(namespace string) v1beta1.IngressInterface {
	return &FakeIngresses{c, namespace}
}

func (c *FakeExtensionsV1beta1) PodSecurityPolicies() v1beta1.PodSecurityPolicyInterface {
	return &FakePodSecurityPolicies{c}
}

func (c *FakeExtensionsV1beta1) ReplicaSets(namespace string) v1beta1.ReplicaSetInterface {
	return &FakeReplicaSets{c, namespace}
}

func (c *FakeExtensionsV1beta1) Scales(namespace string) v1beta1.ScaleInterface {
	return &FakeScales{c, namespace}
}

func (c *FakeExtensionsV1beta1) ThirdPartyResources() v1beta1.ThirdPartyResourceInterface {
	return &FakeThirdPartyResources{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExtensionsV1beta1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
