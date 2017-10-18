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

// TiersGetter has a method to return a TierInterface.
// A group's client should implement this interface.
type TiersGetter interface {
	Tiers(namespace string) TierInterface
}

// TierInterface has methods to work with Tier resources.
type TierInterface interface {
	Create(*calico.Tier) (*calico.Tier, error)
	Update(*calico.Tier) (*calico.Tier, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*calico.Tier, error)
	List(opts v1.ListOptions) (*calico.TierList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Tier, err error)
	TierExpansion
}

// tiers implements TierInterface
type tiers struct {
	client rest.Interface
	ns     string
}

// newTiers returns a Tiers
func newTiers(c *ProjectcalicoClient, namespace string) *tiers {
	return &tiers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a tier and creates it.  Returns the server's representation of the tier, and an error, if there is any.
func (c *tiers) Create(tier *calico.Tier) (result *calico.Tier, err error) {
	result = &calico.Tier{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("tiers").
		Body(tier).
		Do().
		Into(result)
	return
}

// Update takes the representation of a tier and updates it. Returns the server's representation of the tier, and an error, if there is any.
func (c *tiers) Update(tier *calico.Tier) (result *calico.Tier, err error) {
	result = &calico.Tier{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("tiers").
		Name(tier.Name).
		Body(tier).
		Do().
		Into(result)
	return
}

// Delete takes name of the tier and deletes it. Returns an error if one occurs.
func (c *tiers) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tiers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tiers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("tiers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the tier, and returns the corresponding tier object, and an error if there is any.
func (c *tiers) Get(name string, options v1.GetOptions) (result *calico.Tier, err error) {
	result = &calico.Tier{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tiers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Tiers that match those selectors.
func (c *tiers) List(opts v1.ListOptions) (result *calico.TierList, err error) {
	result = &calico.TierList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("tiers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tiers.
func (c *tiers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("tiers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched tier.
func (c *tiers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *calico.Tier, err error) {
	result = &calico.Tier{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("tiers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
