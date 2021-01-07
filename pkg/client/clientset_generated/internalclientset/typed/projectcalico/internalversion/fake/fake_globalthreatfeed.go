// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGlobalThreatFeeds implements GlobalThreatFeedInterface
type FakeGlobalThreatFeeds struct {
	Fake *FakeProjectcalico
}

var globalthreatfeedsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "globalthreatfeeds"}

var globalthreatfeedsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "GlobalThreatFeed"}

// Get takes name of the globalThreatFeed, and returns the corresponding globalThreatFeed object, and an error if there is any.
func (c *FakeGlobalThreatFeeds) Get(ctx context.Context, name string, options v1.GetOptions) (result *projectcalico.GlobalThreatFeed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalthreatfeedsResource, name), &projectcalico.GlobalThreatFeed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalThreatFeed), err
}

// List takes label and field selectors, and returns the list of GlobalThreatFeeds that match those selectors.
func (c *FakeGlobalThreatFeeds) List(ctx context.Context, opts v1.ListOptions) (result *projectcalico.GlobalThreatFeedList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalthreatfeedsResource, globalthreatfeedsKind, opts), &projectcalico.GlobalThreatFeedList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.GlobalThreatFeedList{ListMeta: obj.(*projectcalico.GlobalThreatFeedList).ListMeta}
	for _, item := range obj.(*projectcalico.GlobalThreatFeedList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalThreatFeeds.
func (c *FakeGlobalThreatFeeds) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalthreatfeedsResource, opts))
}

// Create takes the representation of a globalThreatFeed and creates it.  Returns the server's representation of the globalThreatFeed, and an error, if there is any.
func (c *FakeGlobalThreatFeeds) Create(ctx context.Context, globalThreatFeed *projectcalico.GlobalThreatFeed, opts v1.CreateOptions) (result *projectcalico.GlobalThreatFeed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalthreatfeedsResource, globalThreatFeed), &projectcalico.GlobalThreatFeed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalThreatFeed), err
}

// Update takes the representation of a globalThreatFeed and updates it. Returns the server's representation of the globalThreatFeed, and an error, if there is any.
func (c *FakeGlobalThreatFeeds) Update(ctx context.Context, globalThreatFeed *projectcalico.GlobalThreatFeed, opts v1.UpdateOptions) (result *projectcalico.GlobalThreatFeed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalthreatfeedsResource, globalThreatFeed), &projectcalico.GlobalThreatFeed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalThreatFeed), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGlobalThreatFeeds) UpdateStatus(ctx context.Context, globalThreatFeed *projectcalico.GlobalThreatFeed, opts v1.UpdateOptions) (*projectcalico.GlobalThreatFeed, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(globalthreatfeedsResource, "status", globalThreatFeed), &projectcalico.GlobalThreatFeed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalThreatFeed), err
}

// Delete takes name of the globalThreatFeed and deletes it. Returns an error if one occurs.
func (c *FakeGlobalThreatFeeds) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalthreatfeedsResource, name), &projectcalico.GlobalThreatFeed{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalThreatFeeds) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalthreatfeedsResource, listOpts)

	_, err := c.Fake.Invokes(action, &projectcalico.GlobalThreatFeedList{})
	return err
}

// Patch applies the patch and returns the patched globalThreatFeed.
func (c *FakeGlobalThreatFeeds) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *projectcalico.GlobalThreatFeed, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalthreatfeedsResource, name, pt, data, subresources...), &projectcalico.GlobalThreatFeed{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalThreatFeed), err
}
