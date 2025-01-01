// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeBGPPeers implements BGPPeerInterface
type FakeBGPPeers struct {
	Fake *FakeProjectcalicoV3
}

var bgppeersResource = v3.SchemeGroupVersion.WithResource("bgppeers")

var bgppeersKind = v3.SchemeGroupVersion.WithKind("BGPPeer")

// Get takes name of the bGPPeer, and returns the corresponding bGPPeer object, and an error if there is any.
func (c *FakeBGPPeers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(bgppeersResource, name), &v3.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BGPPeer), err
}

// List takes label and field selectors, and returns the list of BGPPeers that match those selectors.
func (c *FakeBGPPeers) List(ctx context.Context, opts v1.ListOptions) (result *v3.BGPPeerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(bgppeersResource, bgppeersKind, opts), &v3.BGPPeerList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.BGPPeerList{ListMeta: obj.(*v3.BGPPeerList).ListMeta}
	for _, item := range obj.(*v3.BGPPeerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested bGPPeers.
func (c *FakeBGPPeers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(bgppeersResource, opts))
}

// Create takes the representation of a bGPPeer and creates it.  Returns the server's representation of the bGPPeer, and an error, if there is any.
func (c *FakeBGPPeers) Create(ctx context.Context, bGPPeer *v3.BGPPeer, opts v1.CreateOptions) (result *v3.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(bgppeersResource, bGPPeer), &v3.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BGPPeer), err
}

// Update takes the representation of a bGPPeer and updates it. Returns the server's representation of the bGPPeer, and an error, if there is any.
func (c *FakeBGPPeers) Update(ctx context.Context, bGPPeer *v3.BGPPeer, opts v1.UpdateOptions) (result *v3.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(bgppeersResource, bGPPeer), &v3.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BGPPeer), err
}

// Delete takes name of the bGPPeer and deletes it. Returns an error if one occurs.
func (c *FakeBGPPeers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(bgppeersResource, name, opts), &v3.BGPPeer{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBGPPeers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(bgppeersResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.BGPPeerList{})
	return err
}

// Patch applies the patch and returns the patched bGPPeer.
func (c *FakeBGPPeers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(bgppeersResource, name, pt, data, subresources...), &v3.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BGPPeer), err
}
