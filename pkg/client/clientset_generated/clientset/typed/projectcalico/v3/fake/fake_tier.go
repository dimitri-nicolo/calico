// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTiers implements TierInterface
type FakeTiers struct {
	Fake *FakeProjectcalicoV3
}

var tiersResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "tiers"}

var tiersKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "Tier"}

// Get takes name of the tier, and returns the corresponding tier object, and an error if there is any.
func (c *FakeTiers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(tiersResource, name), &v3.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Tier), err
}

// List takes label and field selectors, and returns the list of Tiers that match those selectors.
func (c *FakeTiers) List(ctx context.Context, opts v1.ListOptions) (result *v3.TierList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(tiersResource, tiersKind, opts), &v3.TierList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.TierList{ListMeta: obj.(*v3.TierList).ListMeta}
	for _, item := range obj.(*v3.TierList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tiers.
func (c *FakeTiers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(tiersResource, opts))
}

// Create takes the representation of a tier and creates it.  Returns the server's representation of the tier, and an error, if there is any.
func (c *FakeTiers) Create(ctx context.Context, tier *v3.Tier, opts v1.CreateOptions) (result *v3.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(tiersResource, tier), &v3.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Tier), err
}

// Update takes the representation of a tier and updates it. Returns the server's representation of the tier, and an error, if there is any.
func (c *FakeTiers) Update(ctx context.Context, tier *v3.Tier, opts v1.UpdateOptions) (result *v3.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(tiersResource, tier), &v3.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Tier), err
}

// Delete takes name of the tier and deletes it. Returns an error if one occurs.
func (c *FakeTiers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(tiersResource, name), &v3.Tier{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTiers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(tiersResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.TierList{})
	return err
}

// Patch applies the patch and returns the patched tier.
func (c *FakeTiers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(tiersResource, name, pt, data, subresources...), &v3.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Tier), err
}
