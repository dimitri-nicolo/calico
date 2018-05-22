/*
Copyright 2017 Tigera.
*/package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GlobalNetworkSetsGetter has a method to return a GlobalNetworkSetInterface.
// A group's client should implement this interface.
type GlobalNetworkSetsGetter interface {
	GlobalNetworkSets() GlobalNetworkSetInterface
}

// GlobalNetworkSetInterface has methods to work with GlobalNetworkSet resources.
type GlobalNetworkSetInterface interface {
	Create(*v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error)
	Update(*v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v3.GlobalNetworkSet, error)
	List(opts v1.ListOptions) (*v3.GlobalNetworkSetList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalNetworkSet, err error)
	GlobalNetworkSetExpansion
}

// globalNetworkSets implements GlobalNetworkSetInterface
type globalNetworkSets struct {
	client rest.Interface
}

// newGlobalNetworkSets returns a GlobalNetworkSets
func newGlobalNetworkSets(c *ProjectcalicoV3Client) *globalNetworkSets {
	return &globalNetworkSets{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalNetworkSet, and returns the corresponding globalNetworkSet object, and an error if there is any.
func (c *globalNetworkSets) Get(name string, options v1.GetOptions) (result *v3.GlobalNetworkSet, err error) {
	result = &v3.GlobalNetworkSet{}
	err = c.client.Get().
		Resource("globalnetworksets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalNetworkSets that match those selectors.
func (c *globalNetworkSets) List(opts v1.ListOptions) (result *v3.GlobalNetworkSetList, err error) {
	result = &v3.GlobalNetworkSetList{}
	err = c.client.Get().
		Resource("globalnetworksets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalNetworkSets.
func (c *globalNetworkSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("globalnetworksets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a globalNetworkSet and creates it.  Returns the server's representation of the globalNetworkSet, and an error, if there is any.
func (c *globalNetworkSets) Create(globalNetworkSet *v3.GlobalNetworkSet) (result *v3.GlobalNetworkSet, err error) {
	result = &v3.GlobalNetworkSet{}
	err = c.client.Post().
		Resource("globalnetworksets").
		Body(globalNetworkSet).
		Do().
		Into(result)
	return
}

// Update takes the representation of a globalNetworkSet and updates it. Returns the server's representation of the globalNetworkSet, and an error, if there is any.
func (c *globalNetworkSets) Update(globalNetworkSet *v3.GlobalNetworkSet) (result *v3.GlobalNetworkSet, err error) {
	result = &v3.GlobalNetworkSet{}
	err = c.client.Put().
		Resource("globalnetworksets").
		Name(globalNetworkSet.Name).
		Body(globalNetworkSet).
		Do().
		Into(result)
	return
}

// Delete takes name of the globalNetworkSet and deletes it. Returns an error if one occurs.
func (c *globalNetworkSets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalnetworksets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalNetworkSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("globalnetworksets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched globalNetworkSet.
func (c *globalNetworkSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalNetworkSet, err error) {
	result = &v3.GlobalNetworkSet{}
	err = c.client.Patch(pt).
		Resource("globalnetworksets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
