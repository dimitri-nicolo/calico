// Copyright (c) 2021 Tigera, Inc. All rights reserved.

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

// GlobalAlertTemplatesGetter has a method to return a GlobalAlertTemplateInterface.
// A group's client should implement this interface.
type GlobalAlertTemplatesGetter interface {
	GlobalAlertTemplates() GlobalAlertTemplateInterface
}

// GlobalAlertTemplateInterface has methods to work with GlobalAlertTemplate resources.
type GlobalAlertTemplateInterface interface {
	Create(ctx context.Context, globalAlertTemplate *v3.GlobalAlertTemplate, opts v1.CreateOptions) (*v3.GlobalAlertTemplate, error)
	Update(ctx context.Context, globalAlertTemplate *v3.GlobalAlertTemplate, opts v1.UpdateOptions) (*v3.GlobalAlertTemplate, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.GlobalAlertTemplate, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.GlobalAlertTemplateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalAlertTemplate, err error)
	GlobalAlertTemplateExpansion
}

// globalAlertTemplates implements GlobalAlertTemplateInterface
type globalAlertTemplates struct {
	client rest.Interface
}

// newGlobalAlertTemplates returns a GlobalAlertTemplates
func newGlobalAlertTemplates(c *ProjectcalicoV3Client) *globalAlertTemplates {
	return &globalAlertTemplates{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalAlertTemplate, and returns the corresponding globalAlertTemplate object, and an error if there is any.
func (c *globalAlertTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.GlobalAlertTemplate, err error) {
	result = &v3.GlobalAlertTemplate{}
	err = c.client.Get().
		Resource("globalalerttemplates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalAlertTemplates that match those selectors.
func (c *globalAlertTemplates) List(ctx context.Context, opts v1.ListOptions) (result *v3.GlobalAlertTemplateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.GlobalAlertTemplateList{}
	err = c.client.Get().
		Resource("globalalerttemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalAlertTemplates.
func (c *globalAlertTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("globalalerttemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a globalAlertTemplate and creates it.  Returns the server's representation of the globalAlertTemplate, and an error, if there is any.
func (c *globalAlertTemplates) Create(ctx context.Context, globalAlertTemplate *v3.GlobalAlertTemplate, opts v1.CreateOptions) (result *v3.GlobalAlertTemplate, err error) {
	result = &v3.GlobalAlertTemplate{}
	err = c.client.Post().
		Resource("globalalerttemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalAlertTemplate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a globalAlertTemplate and updates it. Returns the server's representation of the globalAlertTemplate, and an error, if there is any.
func (c *globalAlertTemplates) Update(ctx context.Context, globalAlertTemplate *v3.GlobalAlertTemplate, opts v1.UpdateOptions) (result *v3.GlobalAlertTemplate, err error) {
	result = &v3.GlobalAlertTemplate{}
	err = c.client.Put().
		Resource("globalalerttemplates").
		Name(globalAlertTemplate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalAlertTemplate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the globalAlertTemplate and deletes it. Returns an error if one occurs.
func (c *globalAlertTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalalerttemplates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalAlertTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("globalalerttemplates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched globalAlertTemplate.
func (c *globalAlertTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalAlertTemplate, err error) {
	result = &v3.GlobalAlertTemplate{}
	err = c.client.Patch(pt).
		Resource("globalalerttemplates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
