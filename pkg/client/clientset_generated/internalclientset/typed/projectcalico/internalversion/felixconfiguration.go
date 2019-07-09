// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	scheme "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FelixConfigurationsGetter has a method to return a FelixConfigurationInterface.
// A group's client should implement this interface.
type FelixConfigurationsGetter interface {
	FelixConfigurations() FelixConfigurationInterface
}

// FelixConfigurationInterface has methods to work with FelixConfiguration resources.
type FelixConfigurationInterface interface {
	Create(*projectcalico.FelixConfiguration) (*projectcalico.FelixConfiguration, error)
	Update(*projectcalico.FelixConfiguration) (*projectcalico.FelixConfiguration, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*projectcalico.FelixConfiguration, error)
	List(opts v1.ListOptions) (*projectcalico.FelixConfigurationList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.FelixConfiguration, err error)
	FelixConfigurationExpansion
}

// felixConfigurations implements FelixConfigurationInterface
type felixConfigurations struct {
	client rest.Interface
}

// newFelixConfigurations returns a FelixConfigurations
func newFelixConfigurations(c *ProjectcalicoClient) *felixConfigurations {
	return &felixConfigurations{
		client: c.RESTClient(),
	}
}

// Get takes name of the felixConfiguration, and returns the corresponding felixConfiguration object, and an error if there is any.
func (c *felixConfigurations) Get(name string, options v1.GetOptions) (result *projectcalico.FelixConfiguration, err error) {
	result = &projectcalico.FelixConfiguration{}
	err = c.client.Get().
		Resource("felixconfigurations").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FelixConfigurations that match those selectors.
func (c *felixConfigurations) List(opts v1.ListOptions) (result *projectcalico.FelixConfigurationList, err error) {
	result = &projectcalico.FelixConfigurationList{}
	err = c.client.Get().
		Resource("felixconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested felixConfigurations.
func (c *felixConfigurations) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("felixconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a felixConfiguration and creates it.  Returns the server's representation of the felixConfiguration, and an error, if there is any.
func (c *felixConfigurations) Create(felixConfiguration *projectcalico.FelixConfiguration) (result *projectcalico.FelixConfiguration, err error) {
	result = &projectcalico.FelixConfiguration{}
	err = c.client.Post().
		Resource("felixconfigurations").
		Body(felixConfiguration).
		Do().
		Into(result)
	return
}

// Update takes the representation of a felixConfiguration and updates it. Returns the server's representation of the felixConfiguration, and an error, if there is any.
func (c *felixConfigurations) Update(felixConfiguration *projectcalico.FelixConfiguration) (result *projectcalico.FelixConfiguration, err error) {
	result = &projectcalico.FelixConfiguration{}
	err = c.client.Put().
		Resource("felixconfigurations").
		Name(felixConfiguration.Name).
		Body(felixConfiguration).
		Do().
		Into(result)
	return
}

// Delete takes name of the felixConfiguration and deletes it. Returns an error if one occurs.
func (c *felixConfigurations) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("felixconfigurations").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *felixConfigurations) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("felixconfigurations").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched felixConfiguration.
func (c *felixConfigurations) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.FelixConfiguration, err error) {
	result = &projectcalico.FelixConfiguration{}
	err = c.client.Patch(pt).
		Resource("felixconfigurations").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
