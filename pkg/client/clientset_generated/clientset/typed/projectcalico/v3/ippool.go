// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package v3

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
	scheme "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset/scheme"
)

// IPPoolsGetter has a method to return a IPPoolInterface.
// A group's client should implement this interface.
type IPPoolsGetter interface {
	IPPools() IPPoolInterface
}

// IPPoolInterface has methods to work with IPPool resources.
type IPPoolInterface interface {
	Create(ctx context.Context, iPPool *v3.IPPool, opts v1.CreateOptions) (*v3.IPPool, error)
	Update(ctx context.Context, iPPool *v3.IPPool, opts v1.UpdateOptions) (*v3.IPPool, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.IPPool, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.IPPoolList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.IPPool, err error)
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
func (c *iPPools) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Get().
		Resource("ippools").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of IPPools that match those selectors.
func (c *iPPools) List(ctx context.Context, opts v1.ListOptions) (result *v3.IPPoolList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.IPPoolList{}
	err = c.client.Get().
		Resource("ippools").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested iPPools.
func (c *iPPools) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("ippools").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a iPPool and creates it.  Returns the server's representation of the iPPool, and an error, if there is any.
func (c *iPPools) Create(ctx context.Context, iPPool *v3.IPPool, opts v1.CreateOptions) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Post().
		Resource("ippools").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(iPPool).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a iPPool and updates it. Returns the server's representation of the iPPool, and an error, if there is any.
func (c *iPPools) Update(ctx context.Context, iPPool *v3.IPPool, opts v1.UpdateOptions) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Put().
		Resource("ippools").
		Name(iPPool.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(iPPool).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the iPPool and deletes it. Returns an error if one occurs.
func (c *iPPools) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("ippools").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *iPPools) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("ippools").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched iPPool.
func (c *iPPools) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.IPPool, err error) {
	result = &v3.IPPool{}
	err = c.client.Patch(pt).
		Resource("ippools").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
