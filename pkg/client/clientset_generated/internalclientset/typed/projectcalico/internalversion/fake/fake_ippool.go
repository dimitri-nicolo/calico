// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeIPPools implements IPPoolInterface
type FakeIPPools struct {
	Fake *FakeProjectcalico
}

var ippoolsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "ippools"}

var ippoolsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "IPPool"}

// Get takes name of the iPPool, and returns the corresponding iPPool object, and an error if there is any.
func (c *FakeIPPools) Get(name string, options v1.GetOptions) (result *projectcalico.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(ippoolsResource, name), &projectcalico.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.IPPool), err
}

// List takes label and field selectors, and returns the list of IPPools that match those selectors.
func (c *FakeIPPools) List(opts v1.ListOptions) (result *projectcalico.IPPoolList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(ippoolsResource, ippoolsKind, opts), &projectcalico.IPPoolList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.IPPoolList{}
	for _, item := range obj.(*projectcalico.IPPoolList).Items {
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
func (c *FakeIPPools) Create(iPPool *projectcalico.IPPool) (result *projectcalico.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(ippoolsResource, iPPool), &projectcalico.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.IPPool), err
}

// Update takes the representation of a iPPool and updates it. Returns the server's representation of the iPPool, and an error, if there is any.
func (c *FakeIPPools) Update(iPPool *projectcalico.IPPool) (result *projectcalico.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(ippoolsResource, iPPool), &projectcalico.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.IPPool), err
}

// Delete takes name of the iPPool and deletes it. Returns an error if one occurs.
func (c *FakeIPPools) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(ippoolsResource, name), &projectcalico.IPPool{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeIPPools) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(ippoolsResource, listOptions)

	_, err := c.Fake.Invokes(action, &projectcalico.IPPoolList{})
	return err
}

// Patch applies the patch and returns the patched iPPool.
func (c *FakeIPPools) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.IPPool, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(ippoolsResource, name, data, subresources...), &projectcalico.IPPool{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.IPPool), err
}
