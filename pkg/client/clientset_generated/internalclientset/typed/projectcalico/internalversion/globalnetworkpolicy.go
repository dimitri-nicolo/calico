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

// GlobalNetworkPoliciesGetter has a method to return a GlobalNetworkPolicyInterface.
// A group's client should implement this interface.
type GlobalNetworkPoliciesGetter interface {
	GlobalNetworkPolicies() GlobalNetworkPolicyInterface
}

// GlobalNetworkPolicyInterface has methods to work with GlobalNetworkPolicy resources.
type GlobalNetworkPolicyInterface interface {
	Create(*calico.GlobalNetworkPolicy) (*calico.GlobalNetworkPolicy, error)
	Update(*calico.GlobalNetworkPolicy) (*calico.GlobalNetworkPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*calico.GlobalNetworkPolicy, error)
	List(opts v1.ListOptions) (*calico.GlobalNetworkPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.GlobalNetworkPolicy, err error)
	GlobalNetworkPolicyExpansion
}

// globalNetworkPolicies implements GlobalNetworkPolicyInterface
type globalNetworkPolicies struct {
	client rest.Interface
}

// newGlobalNetworkPolicies returns a GlobalNetworkPolicies
func newGlobalNetworkPolicies(c *ProjectcalicoClient) *globalNetworkPolicies {
	return &globalNetworkPolicies{
		client: c.RESTClient(),
	}
}

// Create takes the representation of a globalNetworkPolicy and creates it.  Returns the server's representation of the globalNetworkPolicy, and an error, if there is any.
func (c *globalNetworkPolicies) Create(globalNetworkPolicy *calico.GlobalNetworkPolicy) (result *calico.GlobalNetworkPolicy, err error) {
	result = &calico.GlobalNetworkPolicy{}
	err = c.client.Post().
		Resource("globalnetworkpolicies").
		Body(globalNetworkPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a globalNetworkPolicy and updates it. Returns the server's representation of the globalNetworkPolicy, and an error, if there is any.
func (c *globalNetworkPolicies) Update(globalNetworkPolicy *calico.GlobalNetworkPolicy) (result *calico.GlobalNetworkPolicy, err error) {
	result = &calico.GlobalNetworkPolicy{}
	err = c.client.Put().
		Resource("globalnetworkpolicies").
		Name(globalNetworkPolicy.Name).
		Body(globalNetworkPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the globalNetworkPolicy and deletes it. Returns an error if one occurs.
func (c *globalNetworkPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalnetworkpolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalNetworkPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("globalnetworkpolicies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the globalNetworkPolicy, and returns the corresponding globalNetworkPolicy object, and an error if there is any.
func (c *globalNetworkPolicies) Get(name string, options v1.GetOptions) (result *calico.GlobalNetworkPolicy, err error) {
	result = &calico.GlobalNetworkPolicy{}
	err = c.client.Get().
		Resource("globalnetworkpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalNetworkPolicies that match those selectors.
func (c *globalNetworkPolicies) List(opts v1.ListOptions) (result *calico.GlobalNetworkPolicyList, err error) {
	result = &calico.GlobalNetworkPolicyList{}
	err = c.client.Get().
		Resource("globalnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalNetworkPolicies.
func (c *globalNetworkPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("globalnetworkpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched globalNetworkPolicy.
func (c *globalNetworkPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.GlobalNetworkPolicy, err error) {
	result = &calico.GlobalNetworkPolicy{}
	err = c.client.Patch(pt).
		Resource("globalnetworkpolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
