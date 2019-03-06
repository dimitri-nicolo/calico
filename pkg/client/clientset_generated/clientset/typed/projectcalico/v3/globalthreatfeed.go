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

// GlobalThreatFeedsGetter has a method to return a GlobalThreatFeedInterface.
// A group's client should implement this interface.
type GlobalThreatFeedsGetter interface {
	GlobalThreatFeeds() GlobalThreatFeedInterface
}

// GlobalThreatFeedInterface has methods to work with GlobalThreatFeed resources.
type GlobalThreatFeedInterface interface {
	Create(*v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error)
	Update(*v3.GlobalThreatFeed) (*v3.GlobalThreatFeed, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v3.GlobalThreatFeed, error)
	List(opts v1.ListOptions) (*v3.GlobalThreatFeedList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalThreatFeed, err error)
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
func (c *globalThreatFeeds) Get(name string, options v1.GetOptions) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Get().
		Resource("globalthreatfeeds").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GlobalThreatFeeds that match those selectors.
func (c *globalThreatFeeds) List(opts v1.ListOptions) (result *v3.GlobalThreatFeedList, err error) {
	result = &v3.GlobalThreatFeedList{}
	err = c.client.Get().
		Resource("globalthreatfeeds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested globalThreatFeeds.
func (c *globalThreatFeeds) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("globalthreatfeeds").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a globalThreatFeed and creates it.  Returns the server's representation of the globalThreatFeed, and an error, if there is any.
func (c *globalThreatFeeds) Create(globalThreatFeed *v3.GlobalThreatFeed) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Post().
		Resource("globalthreatfeeds").
		Body(globalThreatFeed).
		Do().
		Into(result)
	return
}

// Update takes the representation of a globalThreatFeed and updates it. Returns the server's representation of the globalThreatFeed, and an error, if there is any.
func (c *globalThreatFeeds) Update(globalThreatFeed *v3.GlobalThreatFeed) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Put().
		Resource("globalthreatfeeds").
		Name(globalThreatFeed.Name).
		Body(globalThreatFeed).
		Do().
		Into(result)
	return
}

// Delete takes name of the globalThreatFeed and deletes it. Returns an error if one occurs.
func (c *globalThreatFeeds) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("globalthreatfeeds").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *globalThreatFeeds) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("globalthreatfeeds").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched globalThreatFeed.
func (c *globalThreatFeeds) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalThreatFeed, err error) {
	result = &v3.GlobalThreatFeed{}
	err = c.client.Patch(pt).
		Resource("globalthreatfeeds").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
