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

package internalversion

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// EndpointsGetter has a method to return a EndpointInterface.
// A group's client should implement this interface.
type EndpointsGetter interface {
	Endpoints() EndpointInterface
}

// EndpointInterface has methods to work with Endpoint resources.
type EndpointInterface interface {
	Create(*calico.Endpoint) (*calico.Endpoint, error)
	Update(*calico.Endpoint) (*calico.Endpoint, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*calico.Endpoint, error)
	List(opts v1.ListOptions) (*calico.EndpointList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Endpoint, err error)
	EndpointExpansion
}

// endpoints implements EndpointInterface
type endpoints struct {
	client rest.Interface
}

// newEndpoints returns a Endpoints
func newEndpoints(c *CalicoClient) *endpoints {
	return &endpoints{
		client: c.RESTClient(),
	}
}

// Create takes the representation of a endpoint and creates it.  Returns the server's representation of the endpoint, and an error, if there is any.
func (c *endpoints) Create(endpoint *calico.Endpoint) (result *calico.Endpoint, err error) {
	result = &calico.Endpoint{}
	err = c.client.Post().
		Resource("endpoints").
		Body(endpoint).
		Do().
		Into(result)
	return
}

// Update takes the representation of a endpoint and updates it. Returns the server's representation of the endpoint, and an error, if there is any.
func (c *endpoints) Update(endpoint *calico.Endpoint) (result *calico.Endpoint, err error) {
	result = &calico.Endpoint{}
	err = c.client.Put().
		Resource("endpoints").
		Name(endpoint.Name).
		Body(endpoint).
		Do().
		Into(result)
	return
}

// Delete takes name of the endpoint and deletes it. Returns an error if one occurs.
func (c *endpoints) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("endpoints").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *endpoints) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("endpoints").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the endpoint, and returns the corresponding endpoint object, and an error if there is any.
func (c *endpoints) Get(name string, options v1.GetOptions) (result *calico.Endpoint, err error) {
	result = &calico.Endpoint{}
	err = c.client.Get().
		Resource("endpoints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Endpoints that match those selectors.
func (c *endpoints) List(opts v1.ListOptions) (result *calico.EndpointList, err error) {
	result = &calico.EndpointList{}
	err = c.client.Get().
		Resource("endpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested endpoints.
func (c *endpoints) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("endpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched endpoint.
func (c *endpoints) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Endpoint, err error) {
	result = &calico.Endpoint{}
	err = c.client.Patch(pt).
		Resource("endpoints").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
