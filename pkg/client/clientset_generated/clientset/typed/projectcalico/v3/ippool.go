// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// IPPoolsGetter has a method to return a IPPoolInterface.
// A group's client should implement this interface.
type IPPoolsGetter interface {
	IPPools() IPPoolInterface
}

// IPPoolInterface has methods to work with IPPool resources.
type IPPoolInterface interface {
	Create(*v3.IPPool) (*v3.IPPool, error)
	Update(*v3.IPPool) (*v3.IPPool, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v3.IPPool, error)
	List(opts v1.ListOptions) (*v3.IPPoolList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.IPPool, err error)
	IPPoolExpansion
}

// iPPools implements IPPoolInterface
type iPPools struct {
	client rest.Interface
}

// newIPPools returns a IPPools
func newIPPools(c *ProjectcalicoV3Client) *iPPools {
	return &iPPools{
		client: c.RESTClient(),
	}
}

// Get takes name of the iPPool, and returns the corresponding iPPool object, and an error if there is any.
func (c *iPPools) Get(name string, options v1.GetOptions) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Get().
		Resource("ippools").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of IPPools that match those selectors.
func (c *iPPools) List(opts v1.ListOptions) (result *v3.IPPoolList, err error) {
	result = &v3.IPPoolList{}
	err = c.client.Get().
		Resource("ippools").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested iPPools.
func (c *iPPools) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("ippools").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a iPPool and creates it.  Returns the server's representation of the iPPool, and an error, if there is any.
func (c *iPPools) Create(iPPool *v3.IPPool) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Post().
		Resource("ippools").
		Body(iPPool).
		Do().
		Into(result)
	return
}

// Update takes the representation of a iPPool and updates it. Returns the server's representation of the iPPool, and an error, if there is any.
func (c *iPPools) Update(iPPool *v3.IPPool) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Put().
		Resource("ippools").
		Name(iPPool.Name).
		Body(iPPool).
		Do().
		Into(result)
	return
}

// Delete takes name of the iPPool and deletes it. Returns an error if one occurs.
func (c *iPPools) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("ippools").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *iPPools) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("ippools").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched iPPool.
func (c *iPPools) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Patch(pt).
		Resource("ippools").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
