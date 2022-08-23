// Copyright (c) 2022 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeBlockAffinities implements BlockAffinityInterface
type FakeBlockAffinities struct {
	Fake *FakeProjectcalicoV3
}

var blockaffinitiesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "blockaffinities"}

var blockaffinitiesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "BlockAffinity"}

// Get takes name of the blockAffinity, and returns the corresponding blockAffinity object, and an error if there is any.
func (c *FakeBlockAffinities) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.BlockAffinity, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(blockaffinitiesResource, name), &v3.BlockAffinity{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BlockAffinity), err
}

// List takes label and field selectors, and returns the list of BlockAffinities that match those selectors.
func (c *FakeBlockAffinities) List(ctx context.Context, opts v1.ListOptions) (result *v3.BlockAffinityList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(blockaffinitiesResource, blockaffinitiesKind, opts), &v3.BlockAffinityList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.BlockAffinityList{ListMeta: obj.(*v3.BlockAffinityList).ListMeta}
	for _, item := range obj.(*v3.BlockAffinityList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested blockAffinities.
func (c *FakeBlockAffinities) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(blockaffinitiesResource, opts))
}

// Create takes the representation of a blockAffinity and creates it.  Returns the server's representation of the blockAffinity, and an error, if there is any.
func (c *FakeBlockAffinities) Create(ctx context.Context, blockAffinity *v3.BlockAffinity, opts v1.CreateOptions) (result *v3.BlockAffinity, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(blockaffinitiesResource, blockAffinity), &v3.BlockAffinity{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BlockAffinity), err
}

// Update takes the representation of a blockAffinity and updates it. Returns the server's representation of the blockAffinity, and an error, if there is any.
func (c *FakeBlockAffinities) Update(ctx context.Context, blockAffinity *v3.BlockAffinity, opts v1.UpdateOptions) (result *v3.BlockAffinity, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(blockaffinitiesResource, blockAffinity), &v3.BlockAffinity{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BlockAffinity), err
}

// Delete takes name of the blockAffinity and deletes it. Returns an error if one occurs.
func (c *FakeBlockAffinities) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(blockaffinitiesResource, name, opts), &v3.BlockAffinity{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBlockAffinities) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(blockaffinitiesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.BlockAffinityList{})
	return err
}

// Patch applies the patch and returns the patched blockAffinity.
func (c *FakeBlockAffinities) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.BlockAffinity, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(blockaffinitiesResource, name, pt, data, subresources...), &v3.BlockAffinity{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.BlockAffinity), err
}
