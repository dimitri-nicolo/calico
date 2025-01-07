// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	"context"
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	scheme "github.com/tigera/api/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BFDConfigurationsGetter has a method to return a BFDConfigurationInterface.
// A group's client should implement this interface.
type BFDConfigurationsGetter interface {
	BFDConfigurations() BFDConfigurationInterface
}

// BFDConfigurationInterface has methods to work with BFDConfiguration resources.
type BFDConfigurationInterface interface {
	Create(ctx context.Context, bFDConfiguration *v3.BFDConfiguration, opts v1.CreateOptions) (*v3.BFDConfiguration, error)
	Update(ctx context.Context, bFDConfiguration *v3.BFDConfiguration, opts v1.UpdateOptions) (*v3.BFDConfiguration, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.BFDConfiguration, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.BFDConfigurationList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.BFDConfiguration, err error)
	BFDConfigurationExpansion
}

// bFDConfigurations implements BFDConfigurationInterface
type bFDConfigurations struct {
	client rest.Interface
}

// newBFDConfigurations returns a BFDConfigurations
func newBFDConfigurations(c *ProjectcalicoV3Client) *bFDConfigurations {
	return &bFDConfigurations{
		client: c.RESTClient(),
	}
}

// Get takes name of the bFDConfiguration, and returns the corresponding bFDConfiguration object, and an error if there is any.
func (c *bFDConfigurations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.BFDConfiguration, err error) {
	result = &v3.BFDConfiguration{}
	err = c.client.Get().
		Resource("bfdconfigurations").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of BFDConfigurations that match those selectors.
func (c *bFDConfigurations) List(ctx context.Context, opts v1.ListOptions) (result *v3.BFDConfigurationList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.BFDConfigurationList{}
	err = c.client.Get().
		Resource("bfdconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested bFDConfigurations.
func (c *bFDConfigurations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("bfdconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a bFDConfiguration and creates it.  Returns the server's representation of the bFDConfiguration, and an error, if there is any.
func (c *bFDConfigurations) Create(ctx context.Context, bFDConfiguration *v3.BFDConfiguration, opts v1.CreateOptions) (result *v3.BFDConfiguration, err error) {
	result = &v3.BFDConfiguration{}
	err = c.client.Post().
		Resource("bfdconfigurations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(bFDConfiguration).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a bFDConfiguration and updates it. Returns the server's representation of the bFDConfiguration, and an error, if there is any.
func (c *bFDConfigurations) Update(ctx context.Context, bFDConfiguration *v3.BFDConfiguration, opts v1.UpdateOptions) (result *v3.BFDConfiguration, err error) {
	result = &v3.BFDConfiguration{}
	err = c.client.Put().
		Resource("bfdconfigurations").
		Name(bFDConfiguration.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(bFDConfiguration).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the bFDConfiguration and deletes it. Returns an error if one occurs.
func (c *bFDConfigurations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("bfdconfigurations").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *bFDConfigurations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("bfdconfigurations").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched bFDConfiguration.
func (c *bFDConfigurations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.BFDConfiguration, err error) {
	result = &v3.BFDConfiguration{}
	err = c.client.Patch(pt).
		Resource("bfdconfigurations").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
