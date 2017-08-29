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

// PoliciesGetter has a method to return a PolicyInterface.
// A group's client should implement this interface.
type PoliciesGetter interface {
	Policies(namespace string) PolicyInterface
}

// PolicyInterface has methods to work with Policy resources.
type PolicyInterface interface {
	Create(*calico.Policy) (*calico.Policy, error)
	Update(*calico.Policy) (*calico.Policy, error)
	UpdateStatus(*calico.Policy) (*calico.Policy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*calico.Policy, error)
	List(opts v1.ListOptions) (*calico.PolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Policy, err error)
	PolicyExpansion
}

// policies implements PolicyInterface
type policies struct {
	client rest.Interface
	ns     string
}

// newPolicies returns a Policies
func newPolicies(c *CalicoClient, namespace string) *policies {
	return &policies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a policy and creates it.  Returns the server's representation of the policy, and an error, if there is any.
func (c *policies) Create(policy *calico.Policy) (result *calico.Policy, err error) {
	result = &calico.Policy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("policies").
		Body(policy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a policy and updates it. Returns the server's representation of the policy, and an error, if there is any.
func (c *policies) Update(policy *calico.Policy) (result *calico.Policy, err error) {
	result = &calico.Policy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("policies").
		Name(policy.Name).
		Body(policy).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *policies) UpdateStatus(policy *calico.Policy) (result *calico.Policy, err error) {
	result = &calico.Policy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("policies").
		Name(policy.Name).
		SubResource("status").
		Body(policy).
		Do().
		Into(result)
	return
}

// Delete takes name of the policy and deletes it. Returns an error if one occurs.
func (c *policies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("policies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *policies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("policies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the policy, and returns the corresponding policy object, and an error if there is any.
func (c *policies) Get(name string, options v1.GetOptions) (result *calico.Policy, err error) {
	result = &calico.Policy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("policies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Policies that match those selectors.
func (c *policies) List(opts v1.ListOptions) (result *calico.PolicyList, err error) {
	result = &calico.PolicyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("policies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested policies.
func (c *policies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("policies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched policy.
func (c *policies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Policy, err error) {
	result = &calico.Policy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("policies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
