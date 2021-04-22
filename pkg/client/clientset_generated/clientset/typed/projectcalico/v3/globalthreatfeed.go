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

// GlobalThreatFeedsGetter has a method to return a GlobalThreatFeedInterface.
// A group's client should implement this interface.
type GlobalThreatFeedsGetter interface {
	GlobalThreatFeeds() GlobalThreatFeedInterface
}

// GlobalThreatFeedInterface has methods to work with GlobalThreatFeed resources.
type GlobalThreatFeedInterface interface {
	Create(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.CreateOptions) (*v3.GlobalThreatFeed, error)
	Update(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.UpdateOptions) (*v3.GlobalThreatFeed, error)
	UpdateStatus(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.UpdateOptions) (*v3.GlobalThreatFeed, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.GlobalThreatFeed, error)
	List(ctx context.Context, opts v1.ListOptions) (*v3.GlobalThreatFeedList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalThreatFeed, err error)
	GlobalThreatFeedExpansion
}

// globalThreatFeeds implements GlobalThreatFeedInterface
type globalThreatFeeds struct {
	client rest.Interface
}

// newGlobalThreatFeeds returns a GlobalThreatFeeds
func newGlobalThreatFeeds(c *ProjectcalicoV3Client) *globalThreatFeeds {
	return &globalThreatFeeds{
		client: c.RESTClient(),
	}
}

// Get takes name of the globalThreatFeed, and returns the corresponding globalThreatFeed object, and an error if there is any.
func (c *globalThreatFeeds) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Get().
		Resource("globalthreatfeeds").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalThreatFeeds that match those selectors.
func (c *globalThreatFeeds) List(ctx context.Context, opts v1.ListOptions) (result *v3.GlobalThreatFeedList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v3.GlobalThreatFeedList{}
	err = c.client.Get().
		Resource("globalthreatfeeds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalThreatFeeds.
func (c *globalThreatFeeds) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("globalthreatfeeds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a globalThreatFeed and creates it.  Returns the server's representation of the globalThreatFeed, and an error, if there is any.
func (c *globalThreatFeeds) Create(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.CreateOptions) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Post().
		Resource("globalthreatfeeds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalThreatFeed).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a globalThreatFeed and updates it. Returns the server's representation of the globalThreatFeed, and an error, if there is any.
func (c *globalThreatFeeds) Update(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.UpdateOptions) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Put().
		Resource("globalthreatfeeds").
		Name(globalThreatFeed.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalThreatFeed).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *globalThreatFeeds) UpdateStatus(ctx context.Context, globalThreatFeed *v3.GlobalThreatFeed, opts v1.UpdateOptions) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Put().
		Resource("globalthreatfeeds").
		Name(globalThreatFeed.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(globalThreatFeed).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the globalThreatFeed and deletes it. Returns an error if one occurs.
func (c *globalThreatFeeds) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalthreatfeeds").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalThreatFeeds) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("globalthreatfeeds").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched globalThreatFeed.
func (c *globalThreatFeeds) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Patch(pt).
		Resource("globalthreatfeeds").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
