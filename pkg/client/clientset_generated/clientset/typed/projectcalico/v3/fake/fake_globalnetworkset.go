/*
Copyright 2017 Tigera.
*/package fake

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGlobalNetworkSets implements GlobalNetworkSetInterface
type FakeGlobalNetworkSets struct {
	Fake *FakeProjectcalicoV3
}

var globalnetworksetsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "globalnetworksets"}

var globalnetworksetsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "GlobalNetworkSet"}

// Get takes name of the globalNetworkSet, and returns the corresponding globalNetworkSet object, and an error if there is any.
func (c *FakeGlobalNetworkSets) Get(name string, options v1.GetOptions) (result *v3.GlobalNetworkSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalnetworksetsResource, name), &v3.GlobalNetworkSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalNetworkSet), err
}

// List takes label and field selectors, and returns the list of GlobalNetworkSets that match those selectors.
func (c *FakeGlobalNetworkSets) List(opts v1.ListOptions) (result *v3.GlobalNetworkSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalnetworksetsResource, globalnetworksetsKind, opts), &v3.GlobalNetworkSetList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.GlobalNetworkSetList{}
	for _, item := range obj.(*v3.GlobalNetworkSetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalNetworkSets.
func (c *FakeGlobalNetworkSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalnetworksetsResource, opts))
}

// Create takes the representation of a globalNetworkSet and creates it.  Returns the server's representation of the globalNetworkSet, and an error, if there is any.
func (c *FakeGlobalNetworkSets) Create(globalNetworkSet *v3.GlobalNetworkSet) (result *v3.GlobalNetworkSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalnetworksetsResource, globalNetworkSet), &v3.GlobalNetworkSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalNetworkSet), err
}

// Update takes the representation of a globalNetworkSet and updates it. Returns the server's representation of the globalNetworkSet, and an error, if there is any.
func (c *FakeGlobalNetworkSets) Update(globalNetworkSet *v3.GlobalNetworkSet) (result *v3.GlobalNetworkSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalnetworksetsResource, globalNetworkSet), &v3.GlobalNetworkSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalNetworkSet), err
}

// Delete takes name of the globalNetworkSet and deletes it. Returns an error if one occurs.
func (c *FakeGlobalNetworkSets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalnetworksetsResource, name), &v3.GlobalNetworkSet{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalNetworkSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalnetworksetsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v3.GlobalNetworkSetList{})
	return err
}

// Patch applies the patch and returns the patched globalNetworkSet.
func (c *FakeGlobalNetworkSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalNetworkSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalnetworksetsResource, name, data, subresources...), &v3.GlobalNetworkSet{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalNetworkSet), err
}
