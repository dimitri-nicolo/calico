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
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEndpoints implements EndpointInterface
type FakeEndpoints struct {
	Fake *FakeCalico
}

var endpointsResource = schema.GroupVersionResource{Group: "calico.tigera.io", Version: "", Resource: "endpoints"}

func (c *FakeEndpoints) Create(endpoint *calico.Endpoint) (result *calico.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(endpointsResource, endpoint), &calico.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Endpoint), err
}

func (c *FakeEndpoints) Update(endpoint *calico.Endpoint) (result *calico.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(endpointsResource, endpoint), &calico.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Endpoint), err
}

func (c *FakeEndpoints) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(endpointsResource, name), &calico.Endpoint{})
	return err
}

func (c *FakeEndpoints) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(endpointsResource, listOptions)

	_, err := c.Fake.Invokes(action, &calico.EndpointList{})
	return err
}

func (c *FakeEndpoints) Get(name string, options v1.GetOptions) (result *calico.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(endpointsResource, name), &calico.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Endpoint), err
}

func (c *FakeEndpoints) List(opts v1.ListOptions) (result *calico.EndpointList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(endpointsResource, opts), &calico.EndpointList{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.EndpointList), err
}

// Watch returns a watch.Interface that watches the requested endpoints.
func (c *FakeEndpoints) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(endpointsResource, opts))
}

// Patch applies the patch and returns the patched endpoint.
func (c *FakeEndpoints) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Endpoint, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(endpointsResource, name, data, subresources...), &calico.Endpoint{})
	if obj == nil {
		return nil, err
	}
	return obj.(*calico.Endpoint), err
}
