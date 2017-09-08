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

package v1

import (
	v1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Create(*v1.Endpoint) (*v1.Endpoint, error)
	Update(*v1.Endpoint) (*v1.Endpoint, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.Endpoint, error)
	List(opts meta_v1.ListOptions) (*v1.EndpointList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Endpoint, err error)
	EndpointExpansion
}

// endpoints implements EndpointInterface
type endpoints struct {
	client rest.Interface
}

// newEndpoints returns a Endpoints
func newEndpoints(c *CalicoV1Client) *endpoints {
	return &endpoints{
		client: c.RESTClient(),
	}
}

// Create takes the representation of a endpoint and creates it.  Returns the server's representation of the endpoint, and an error, if there is any.
func (c *endpoints) Create(endpoint *v1.Endpoint) (result *v1.Endpoint, err error) {
	result = &v1.Endpoint{}
	err = c.client.Post().
		Resource("endpoints").
		Body(endpoint).
		Do().
		Into(result)
	return
}

// Update takes the representation of a endpoint and updates it. Returns the server's representation of the endpoint, and an error, if there is any.
func (c *endpoints) Update(endpoint *v1.Endpoint) (result *v1.Endpoint, err error) {
	result = &v1.Endpoint{}
	err = c.client.Put().
		Resource("endpoints").
		Name(endpoint.Name).
		Body(endpoint).
		Do().
		Into(result)
	return
}

// Delete takes name of the endpoint and deletes it. Returns an error if one occurs.
func (c *endpoints) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("endpoints").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *endpoints) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("endpoints").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the endpoint, and returns the corresponding endpoint object, and an error if there is any.
func (c *endpoints) Get(name string, options meta_v1.GetOptions) (result *v1.Endpoint, err error) {
	result = &v1.Endpoint{}
	err = c.client.Get().
		Resource("endpoints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Endpoints that match those selectors.
func (c *endpoints) List(opts meta_v1.ListOptions) (result *v1.EndpointList, err error) {
	result = &v1.EndpointList{}
	err = c.client.Get().
		Resource("endpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested endpoints.
func (c *endpoints) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("endpoints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched endpoint.
func (c *endpoints) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Endpoint, err error) {
	result = &v1.Endpoint{}
	err = c.client.Patch(pt).
		Resource("endpoints").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
