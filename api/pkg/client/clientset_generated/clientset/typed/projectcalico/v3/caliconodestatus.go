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

// CalicoNodeStatusesGetter has a method to return a CalicoNodeStatusInterface.
// A group's client should implement this interface.
type CalicoNodeStatusesGetter interface {
	CalicoNodeStatuses() CalicoNodeStatusInterface
}

// CalicoNodeStatusInterface has methods to work with CalicoNodeStatus resources.
type CalicoNodeStatusInterface interface {
	Create(ctx context.Context, calicoNodeStatus *v3.CalicoNodeStatus, opts v1.CreateOptions) (*v3.CalicoNodeStatus, error)
	Update(ctx context.Context, calicoNodeStatus *v3.CalicoNodeStatus, opts v1.UpdateOptions) (*v3.CalicoNodeStatus, error)
	UpdateStatus(ctx context.Context, calicoNodeStatus *v3.CalicoNodeStatus, opts v1.UpdateOptions) (*v3.CalicoNodeStatus, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.CalicoNodeStatus, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.CalicoNodeStatusList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.CalicoNodeStatus, err error)
	CalicoNodeStatusExpansion
}

// calicoNodeStatuses implements CalicoNodeStatusInterface
type calicoNodeStatuses struct {
	client rest.Interface
}

// newCalicoNodeStatuses returns a CalicoNodeStatuses
func newCalicoNodeStatuses(c *ProjectcalicoV3Client) *calicoNodeStatuses {
	return &calicoNodeStatuses{
		client: c.RESTClient(),
	}
}

// Get takes name of the calicoNodeStatus, and returns the corresponding calicoNodeStatus object, and an error if there is any.
func (c *calicoNodeStatuses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.CalicoNodeStatus, err error) {
	result = &v3.CalicoNodeStatus{}
	err = c.client.Get().
		Resource("caliconodestatuses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CalicoNodeStatuses that match those selectors.
func (c *calicoNodeStatuses) List(ctx context.Context, opts v1.ListOptions) (result *v3.CalicoNodeStatusList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.CalicoNodeStatusList{}
	err = c.client.Get().
		Resource("caliconodestatuses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested calicoNodeStatuses.
func (c *calicoNodeStatuses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("caliconodestatuses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a calicoNodeStatus and creates it.  Returns the server's representation of the calicoNodeStatus, and an error, if there is any.
func (c *calicoNodeStatuses) Create(ctx context.Context, calicoNodeStatus *v3.CalicoNodeStatus, opts v1.CreateOptions) (result *v3.CalicoNodeStatus, err error) {
	result = &v3.CalicoNodeStatus{}
	err = c.client.Post().
		Resource("caliconodestatuses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(calicoNodeStatus).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a calicoNodeStatus and updates it. Returns the server's representation of the calicoNodeStatus, and an error, if there is any.
func (c *calicoNodeStatuses) Update(ctx context.Context, calicoNodeStatus *v3.CalicoNodeStatus, opts v1.UpdateOptions) (result *v3.CalicoNodeStatus, err error) {
	result = &v3.CalicoNodeStatus{}
	err = c.client.Put().
		Resource("caliconodestatuses").
		Name(calicoNodeStatus.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(calicoNodeStatus).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *calicoNodeStatuses) UpdateStatus(ctx context.Context, calicoNodeStatus *v3.CalicoNodeStatus, opts v1.UpdateOptions) (result *v3.CalicoNodeStatus, err error) {
	result = &v3.CalicoNodeStatus{}
	err = c.client.Put().
		Resource("caliconodestatuses").
		Name(calicoNodeStatus.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(calicoNodeStatus).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the calicoNodeStatus and deletes it. Returns an error if one occurs.
func (c *calicoNodeStatuses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("caliconodestatuses").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *calicoNodeStatuses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("caliconodestatuses").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched calicoNodeStatus.
func (c *calicoNodeStatuses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.CalicoNodeStatus, err error) {
	result = &v3.CalicoNodeStatus{}
	err = c.client.Patch(pt).
		Resource("caliconodestatuses").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
