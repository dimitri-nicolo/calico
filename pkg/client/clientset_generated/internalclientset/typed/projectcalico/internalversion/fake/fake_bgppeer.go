// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	projectcalico "github.com/tigera/apiserver/pkg/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeBGPPeers implements BGPPeerInterface
type FakeBGPPeers struct {
	Fake *FakeProjectcalico
}

var bgppeersResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "bgppeers"}

var bgppeersKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "BGPPeer"}

// Get takes name of the bGPPeer, and returns the corresponding bGPPeer object, and an error if there is any.
func (c *FakeBGPPeers) Get(name string, options v1.GetOptions) (result *projectcalico.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(bgppeersResource, name), &projectcalico.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.BGPPeer), err
}

// List takes label and field selectors, and returns the list of BGPPeers that match those selectors.
func (c *FakeBGPPeers) List(opts v1.ListOptions) (result *projectcalico.BGPPeerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(bgppeersResource, bgppeersKind, opts), &projectcalico.BGPPeerList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.BGPPeerList{ListMeta: obj.(*projectcalico.BGPPeerList).ListMeta}
	for _, item := range obj.(*projectcalico.BGPPeerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested bGPPeers.
func (c *FakeBGPPeers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(bgppeersResource, opts))
}

// Create takes the representation of a bGPPeer and creates it.  Returns the server's representation of the bGPPeer, and an error, if there is any.
func (c *FakeBGPPeers) Create(bGPPeer *projectcalico.BGPPeer) (result *projectcalico.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(bgppeersResource, bGPPeer), &projectcalico.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.BGPPeer), err
}

// Update takes the representation of a bGPPeer and updates it. Returns the server's representation of the bGPPeer, and an error, if there is any.
func (c *FakeBGPPeers) Update(bGPPeer *projectcalico.BGPPeer) (result *projectcalico.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(bgppeersResource, bGPPeer), &projectcalico.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.BGPPeer), err
}

// Delete takes name of the bGPPeer and deletes it. Returns an error if one occurs.
func (c *FakeBGPPeers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(bgppeersResource, name), &projectcalico.BGPPeer{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBGPPeers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(bgppeersResource, listOptions)

	_, err := c.Fake.Invokes(action, &projectcalico.BGPPeerList{})
	return err
}

// Patch applies the patch and returns the patched bGPPeer.
func (c *FakeBGPPeers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.BGPPeer, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(bgppeersResource, name, pt, data, subresources...), &projectcalico.BGPPeer{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.BGPPeer), err
}
