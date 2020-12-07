// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	"context"
	"time"

	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	scheme "github.com/tigera/apiserver/pkg/client/clientset_generated/internalclientset/scheme"
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
	Create(ctx context.Context, globalNetworkSet *projectcalico.GlobalNetworkSet, opts v1.CreateOptions) (*projectcalico.GlobalNetworkSet, error)
	Update(ctx context.Context, globalNetworkSet *projectcalico.GlobalNetworkSet, opts v1.UpdateOptions) (*projectcalico.GlobalNetworkSet, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*projectcalico.GlobalNetworkSet, error)
	List(ctx context.Context, opts v1.ListOptions) (*projectcalico.GlobalNetworkSetList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.GlobalNetworkSet, err error)
	GlobalNetworkSetExpansion
}

// globalNetworkSets implements GlobalNetworkSetInterface
type globalNetworkSets struct {
	client rest.Interface
}

// newGlobalNetworkSets returns a GlobalNetworkSets
func newGlobalNetworkSets(c *ProjectcalicoClient) *globalNetworkSets {
	return &globalNetworkSets{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalNetworkSet, and returns the corresponding globalNetworkSet object, and an error if there is any.
func (c *globalNetworkSets) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.GlobalNetworkSet, err error) {
	result = &projectcalico.GlobalNetworkSet{}
	err = c.client.Get().
		Resource("globalnetworksets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalNetworkSets that match those selectors.
func (c *globalNetworkSets) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.GlobalNetworkSetList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &projectcalico.GlobalNetworkSetList{}
	err = c.client.Get().
		Resource("globalnetworksets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalNetworkSets.
func (c *globalNetworkSets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("globalnetworksets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a globalNetworkSet and creates it.  Returns the server's representation of the globalNetworkSet, and an error, if there is any.
func (c *globalNetworkSets) Create(ctx context.Context, globalNetworkSet *projectcalico.GlobalNetworkSet, opts v1.CreateOptions) (result *projectcalico.GlobalNetworkSet, err error) {
	result = &projectcalico.GlobalNetworkSet{}
	err = c.client.Post().
		Resource("globalnetworksets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalNetworkSet).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a globalNetworkSet and updates it. Returns the server's representation of the globalNetworkSet, and an error, if there is any.
func (c *globalNetworkSets) Update(ctx context.Context, globalNetworkSet *projectcalico.GlobalNetworkSet, opts v1.UpdateOptions) (result *projectcalico.GlobalNetworkSet, err error) {
	result = &projectcalico.GlobalNetworkSet{}
	err = c.client.Put().
		Resource("globalnetworksets").
		Name(globalNetworkSet.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalNetworkSet).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the globalNetworkSet and deletes it. Returns an error if one occurs.
func (c *globalNetworkSets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalnetworksets").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalNetworkSets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("globalnetworksets").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched globalNetworkSet.
func (c *globalNetworkSets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.GlobalNetworkSet, err error) {
	result = &projectcalico.GlobalNetworkSet{}
	err = c.client.Patch(pt).
		Resource("globalnetworksets").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
