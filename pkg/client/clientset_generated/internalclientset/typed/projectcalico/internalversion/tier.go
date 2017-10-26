/*
Copyright 2017 Tigera.
*/package internalversion

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// TiersGetter has a method to return a TierInterface.
// A group's client should implement this interface.
type TiersGetter interface {
	Tiers() TierInterface
}

// TierInterface has methods to work with Tier resources.
type TierInterface interface {
	Create(*projectcalico.Tier) (*projectcalico.Tier, error)
	Update(*projectcalico.Tier) (*projectcalico.Tier, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*projectcalico.Tier, error)
	List(opts v1.ListOptions) (*projectcalico.TierList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.Tier, err error)
	TierExpansion
}

// tiers implements TierInterface
type tiers struct {
	client rest.Interface
}

// newTiers returns a Tiers
func newTiers(c *ProjectcalicoClient) *tiers {
	return &tiers{
		client: c.RESTClient(),
	}
}

// Get takes name of the tier, and returns the corresponding tier object, and an error if there is any.
func (c *tiers) Get(name string, options v1.GetOptions) (result *projectcalico.Tier, err error) {
	result = &projectcalico.Tier{}
	err = c.client.Get().
		Resource("tiers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Tiers that match those selectors.
func (c *tiers) List(opts v1.ListOptions) (result *projectcalico.TierList, err error) {
	result = &projectcalico.TierList{}
	err = c.client.Get().
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
		Resource("tiers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a tier and creates it.  Returns the server's representation of the tier, and an error, if there is any.
func (c *tiers) Create(tier *projectcalico.Tier) (result *projectcalico.Tier, err error) {
	result = &projectcalico.Tier{}
	err = c.client.Post().
		Resource("tiers").
		Body(tier).
		Do().
		Into(result)
	return
}

// Update takes the representation of a tier and updates it. Returns the server's representation of the tier, and an error, if there is any.
func (c *tiers) Update(tier *projectcalico.Tier) (result *projectcalico.Tier, err error) {
	result = &projectcalico.Tier{}
	err = c.client.Put().
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
		Resource("tiers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tiers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("tiers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched tier.
func (c *tiers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.Tier, err error) {
	result = &projectcalico.Tier{}
	err = c.client.Patch(pt).
		Resource("tiers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
