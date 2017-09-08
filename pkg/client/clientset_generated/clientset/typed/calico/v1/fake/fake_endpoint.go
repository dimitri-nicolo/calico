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
	v1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEndpoints implements EndpointInterface
type FakeEndpoints struct {
	Fake *FakeCalicoV1
}

var endpointsResource = schema.GroupVersionResource{Group: "calico.tigera.io", Version: "v1", Resource: "endpoints"}

var endpointsKind = schema.GroupVersionKind{Group: "calico.tigera.io", Version: "v1", Kind: "Endpoint"}

func (c *FakeEndpoints) Create(endpoint *v1.Endpoint) (result *v1.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(endpointsResource, endpoint), &v1.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Endpoint), err
}

func (c *FakeEndpoints) Update(endpoint *v1.Endpoint) (result *v1.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(endpointsResource, endpoint), &v1.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Endpoint), err
}

func (c *FakeEndpoints) Delete(name string, options *meta_v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(endpointsResource, name), &v1.Endpoint{})
	return err
}

func (c *FakeEndpoints) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(endpointsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1.EndpointList{})
	return err
}

func (c *FakeEndpoints) Get(name string, options meta_v1.GetOptions) (result *v1.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(endpointsResource, name), &v1.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Endpoint), err
}

func (c *FakeEndpoints) List(opts meta_v1.ListOptions) (result *v1.EndpointList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(endpointsResource, endpointsKind, opts), &v1.EndpointList{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.EndpointList), err
}

// Watch returns a watch.Interface that watches the requested endpoints.
func (c *FakeEndpoints) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(endpointsResource, opts))
}

// Patch applies the patch and returns the patched endpoint.
func (c *FakeEndpoints) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(endpointsResource, name, data, subresources...), &v1.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Endpoint), err
}
