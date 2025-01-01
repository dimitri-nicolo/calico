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

// UISettingsGetter has a method to return a UISettingsInterface.
// A group's client should implement this interface.
type UISettingsGetter interface {
	UISettings() UISettingsInterface
}

// UISettingsInterface has methods to work with UISettings resources.
type UISettingsInterface interface {
	Create(ctx context.Context, uISettings *v3.UISettings, opts v1.CreateOptions) (*v3.UISettings, error)
	Update(ctx context.Context, uISettings *v3.UISettings, opts v1.UpdateOptions) (*v3.UISettings, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.UISettings, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.UISettingsList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.UISettings, err error)
	UISettingsExpansion
}

// uISettings implements UISettingsInterface
type uISettings struct {
	client rest.Interface
}

// newUISettings returns a UISettings
func newUISettings(c *ProjectcalicoV3Client) *uISettings {
	return &uISettings{
		client: c.RESTClient(),
	}
}

// Get takes name of the uISettings, and returns the corresponding uISettings object, and an error if there is any.
func (c *uISettings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.UISettings, err error) {
	result = &v3.UISettings{}
	err = c.client.Get().
		Resource("uisettings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of UISettings that match those selectors.
func (c *uISettings) List(ctx context.Context, opts v1.ListOptions) (result *v3.UISettingsList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.UISettingsList{}
	err = c.client.Get().
		Resource("uisettings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested uISettings.
func (c *uISettings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("uisettings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a uISettings and creates it.  Returns the server's representation of the uISettings, and an error, if there is any.
func (c *uISettings) Create(ctx context.Context, uISettings *v3.UISettings, opts v1.CreateOptions) (result *v3.UISettings, err error) {
	result = &v3.UISettings{}
	err = c.client.Post().
		Resource("uisettings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(uISettings).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a uISettings and updates it. Returns the server's representation of the uISettings, and an error, if there is any.
func (c *uISettings) Update(ctx context.Context, uISettings *v3.UISettings, opts v1.UpdateOptions) (result *v3.UISettings, err error) {
	result = &v3.UISettings{}
	err = c.client.Put().
		Resource("uisettings").
		Name(uISettings.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(uISettings).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the uISettings and deletes it. Returns an error if one occurs.
func (c *uISettings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("uisettings").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *uISettings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("uisettings").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched uISettings.
func (c *uISettings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.UISettings, err error) {
	result = &v3.UISettings{}
	err = c.client.Patch(pt).
		Resource("uisettings").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
