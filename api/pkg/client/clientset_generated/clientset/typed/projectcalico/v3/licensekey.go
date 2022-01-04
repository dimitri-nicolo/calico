// Copyright (c) 2022 Tigera, Inc. All rights reserved.

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

// LicenseKeysGetter has a method to return a LicenseKeyInterface.
// A group's client should implement this interface.
type LicenseKeysGetter interface {
	LicenseKeys() LicenseKeyInterface
}

// LicenseKeyInterface has methods to work with LicenseKey resources.
type LicenseKeyInterface interface {
	Create(ctx context.Context, licenseKey *v3.LicenseKey, opts v1.CreateOptions) (*v3.LicenseKey, error)
	Update(ctx context.Context, licenseKey *v3.LicenseKey, opts v1.UpdateOptions) (*v3.LicenseKey, error)
	UpdateStatus(ctx context.Context, licenseKey *v3.LicenseKey, opts v1.UpdateOptions) (*v3.LicenseKey, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.LicenseKey, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.LicenseKeyList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.LicenseKey, err error)
	LicenseKeyExpansion
}

// licenseKeys implements LicenseKeyInterface
type licenseKeys struct {
	client rest.Interface
}

// newLicenseKeys returns a LicenseKeys
func newLicenseKeys(c *ProjectcalicoV3Client) *licenseKeys {
	return &licenseKeys{
		client: c.RESTClient(),
	}
}

// Get takes name of the licenseKey, and returns the corresponding licenseKey object, and an error if there is any.
func (c *licenseKeys) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.LicenseKey, err error) {
	result = &v3.LicenseKey{}
	err = c.client.Get().
		Resource("licensekeys").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of LicenseKeys that match those selectors.
func (c *licenseKeys) List(ctx context.Context, opts v1.ListOptions) (result *v3.LicenseKeyList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.LicenseKeyList{}
	err = c.client.Get().
		Resource("licensekeys").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested licenseKeys.
func (c *licenseKeys) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("licensekeys").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a licenseKey and creates it.  Returns the server's representation of the licenseKey, and an error, if there is any.
func (c *licenseKeys) Create(ctx context.Context, licenseKey *v3.LicenseKey, opts v1.CreateOptions) (result *v3.LicenseKey, err error) {
	result = &v3.LicenseKey{}
	err = c.client.Post().
		Resource("licensekeys").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(licenseKey).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a licenseKey and updates it. Returns the server's representation of the licenseKey, and an error, if there is any.
func (c *licenseKeys) Update(ctx context.Context, licenseKey *v3.LicenseKey, opts v1.UpdateOptions) (result *v3.LicenseKey, err error) {
	result = &v3.LicenseKey{}
	err = c.client.Put().
		Resource("licensekeys").
		Name(licenseKey.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(licenseKey).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *licenseKeys) UpdateStatus(ctx context.Context, licenseKey *v3.LicenseKey, opts v1.UpdateOptions) (result *v3.LicenseKey, err error) {
	result = &v3.LicenseKey{}
	err = c.client.Put().
		Resource("licensekeys").
		Name(licenseKey.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(licenseKey).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the licenseKey and deletes it. Returns an error if one occurs.
func (c *licenseKeys) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("licensekeys").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *licenseKeys) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("licensekeys").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched licenseKey.
func (c *licenseKeys) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.LicenseKey, err error) {
	result = &v3.LicenseKey{}
	err = c.client.Patch(pt).
		Resource("licensekeys").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
