/*
Copyright 2017 Tigera.
*/package fake

import (
	v2 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTiers implements TierInterface
type FakeTiers struct {
	Fake *FakeProjectcalicoV2
}

var tiersResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v2", Resource: "tiers"}

var tiersKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v2", Kind: "Tier"}

// Get takes name of the tier, and returns the corresponding tier object, and an error if there is any.
func (c *FakeTiers) Get(name string, options v1.GetOptions) (result *v2.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(tiersResource, name), &v2.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.Tier), err
}

// List takes label and field selectors, and returns the list of Tiers that match those selectors.
func (c *FakeTiers) List(opts v1.ListOptions) (result *v2.TierList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(tiersResource, tiersKind, opts), &v2.TierList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v2.TierList{}
	for _, item := range obj.(*v2.TierList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tiers.
func (c *FakeTiers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(tiersResource, opts))
}

// Create takes the representation of a tier and creates it.  Returns the server's representation of the tier, and an error, if there is any.
func (c *FakeTiers) Create(tier *v2.Tier) (result *v2.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(tiersResource, tier), &v2.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.Tier), err
}

// Update takes the representation of a tier and updates it. Returns the server's representation of the tier, and an error, if there is any.
func (c *FakeTiers) Update(tier *v2.Tier) (result *v2.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(tiersResource, tier), &v2.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.Tier), err
}

// Delete takes name of the tier and deletes it. Returns an error if one occurs.
func (c *FakeTiers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(tiersResource, name), &v2.Tier{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTiers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(tiersResource, listOptions)

	_, err := c.Fake.Invokes(action, &v2.TierList{})
	return err
}

// Patch applies the patch and returns the patched tier.
func (c *FakeTiers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v2.Tier, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(tiersResource, name, data, subresources...), &v2.Tier{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2.Tier), err
}
