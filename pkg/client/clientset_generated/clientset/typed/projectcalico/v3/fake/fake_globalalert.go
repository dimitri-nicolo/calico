// Copyright (c) 2020 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v3 "github.com/tigera/apiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGlobalAlerts implements GlobalAlertInterface
type FakeGlobalAlerts struct {
	Fake *FakeProjectcalicoV3
}

var globalalertsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "globalalerts"}

var globalalertsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "GlobalAlert"}

// Get takes name of the globalAlert, and returns the corresponding globalAlert object, and an error if there is any.
func (c *FakeGlobalAlerts) Get(name string, options v1.GetOptions) (result *v3.GlobalAlert, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalalertsResource, name), &v3.GlobalAlert{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalAlert), err
}

// List takes label and field selectors, and returns the list of GlobalAlerts that match those selectors.
func (c *FakeGlobalAlerts) List(opts v1.ListOptions) (result *v3.GlobalAlertList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalalertsResource, globalalertsKind, opts), &v3.GlobalAlertList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.GlobalAlertList{ListMeta: obj.(*v3.GlobalAlertList).ListMeta}
	for _, item := range obj.(*v3.GlobalAlertList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalAlerts.
func (c *FakeGlobalAlerts) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalalertsResource, opts))
}

// Create takes the representation of a globalAlert and creates it.  Returns the server's representation of the globalAlert, and an error, if there is any.
func (c *FakeGlobalAlerts) Create(globalAlert *v3.GlobalAlert) (result *v3.GlobalAlert, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalalertsResource, globalAlert), &v3.GlobalAlert{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalAlert), err
}

// Update takes the representation of a globalAlert and updates it. Returns the server's representation of the globalAlert, and an error, if there is any.
func (c *FakeGlobalAlerts) Update(globalAlert *v3.GlobalAlert) (result *v3.GlobalAlert, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalalertsResource, globalAlert), &v3.GlobalAlert{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalAlert), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGlobalAlerts) UpdateStatus(globalAlert *v3.GlobalAlert) (*v3.GlobalAlert, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(globalalertsResource, "status", globalAlert), &v3.GlobalAlert{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalAlert), err
}

// Delete takes name of the globalAlert and deletes it. Returns an error if one occurs.
func (c *FakeGlobalAlerts) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalalertsResource, name), &v3.GlobalAlert{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalAlerts) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalalertsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v3.GlobalAlertList{})
	return err
}

// Patch applies the patch and returns the patched globalAlert.
func (c *FakeGlobalAlerts) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalAlert, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalalertsResource, name, pt, data, subresources...), &v3.GlobalAlert{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalAlert), err
}
