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
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeCoreV1 struct {
	*testing.Fake
}

func (c *FakeCoreV1) ComponentStatuses() v1.ComponentStatusInterface {
	return &FakeComponentStatuses{c}
}

func (c *FakeCoreV1) ConfigMaps(namespace string) v1.ConfigMapInterface {
	return &FakeConfigMaps{c, namespace}
}

func (c *FakeCoreV1) Endpoints(namespace string) v1.EndpointsInterface {
	return &FakeEndpoints{c, namespace}
}

func (c *FakeCoreV1) Events(namespace string) v1.EventInterface {
	return &FakeEvents{c, namespace}
}

func (c *FakeCoreV1) LimitRanges(namespace string) v1.LimitRangeInterface {
	return &FakeLimitRanges{c, namespace}
}

func (c *FakeCoreV1) Namespaces() v1.NamespaceInterface {
	return &FakeNamespaces{c}
}

func (c *FakeCoreV1) Nodes() v1.NodeInterface {
	return &FakeNodes{c}
}

func (c *FakeCoreV1) PersistentVolumes() v1.PersistentVolumeInterface {
	return &FakePersistentVolumes{c}
}

func (c *FakeCoreV1) PersistentVolumeClaims(namespace string) v1.PersistentVolumeClaimInterface {
	return &FakePersistentVolumeClaims{c, namespace}
}

func (c *FakeCoreV1) Pods(namespace string) v1.PodInterface {
	return &FakePods{c, namespace}
}

func (c *FakeCoreV1) PodTemplates(namespace string) v1.PodTemplateInterface {
	return &FakePodTemplates{c, namespace}
}

func (c *FakeCoreV1) ReplicationControllers(namespace string) v1.ReplicationControllerInterface {
	return &FakeReplicationControllers{c, namespace}
}

func (c *FakeCoreV1) ResourceQuotas(namespace string) v1.ResourceQuotaInterface {
	return &FakeResourceQuotas{c, namespace}
}

func (c *FakeCoreV1) Secrets(namespace string) v1.SecretInterface {
	return &FakeSecrets{c, namespace}
}

func (c *FakeCoreV1) Services(namespace string) v1.ServiceInterface {
	return &FakeServices{c, namespace}
}

func (c *FakeCoreV1) ServiceAccounts(namespace string) v1.ServiceAccountInterface {
	return &FakeServiceAccounts{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCoreV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
