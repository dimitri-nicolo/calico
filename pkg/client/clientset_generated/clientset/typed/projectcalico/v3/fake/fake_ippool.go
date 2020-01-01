// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeIPPools implements IPPoolInterface
type FakeIPPools struct {
	Fake *FakeProjectcalicoV3
}

var ippoolsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "ippools"}

var ippoolsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "IPPool"}

// Get takes name of the iPPool, and returns the corresponding iPPool object, and an error if there is any.
func (c *FakeIPPools) Get(name string, options v1.GetOptions) (result *v3.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(ippoolsResource, name), &v3.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.IPPool), err
}

// List takes label and field selectors, and returns the list of IPPools that match those selectors.
func (c *FakeIPPools) List(opts v1.ListOptions) (result *v3.IPPoolList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(ippoolsResource, ippoolsKind, opts), &v3.IPPoolList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.IPPoolList{ListMeta: obj.(*v3.IPPoolList).ListMeta}
	for _, item := range obj.(*v3.IPPoolList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested iPPools.
func (c *FakeIPPools) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(ippoolsResource, opts))
}

// Create takes the representation of a iPPool and creates it.  Returns the server's representation of the iPPool, and an error, if there is any.
func (c *FakeIPPools) Create(iPPool *v3.IPPool) (result *v3.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(ippoolsResource, iPPool), &v3.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.IPPool), err
}

// Update takes the representation of a iPPool and updates it. Returns the server's representation of the iPPool, and an error, if there is any.
func (c *FakeIPPools) Update(iPPool *v3.IPPool) (result *v3.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(ippoolsResource, iPPool), &v3.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.IPPool), err
}

// Delete takes name of the iPPool and deletes it. Returns an error if one occurs.
func (c *FakeIPPools) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(ippoolsResource, name), &v3.IPPool{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeIPPools) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(ippoolsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v3.IPPoolList{})
	return err
}

// Patch applies the patch and returns the patched iPPool.
func (c *FakeIPPools) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(ippoolsResource, name, pt, data, subresources...), &v3.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.IPPool), err
}
