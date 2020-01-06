// Copyright (c) 2020 Tigera, Inc. All rights reserved.

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

// FakeGlobalReports implements GlobalReportInterface
type FakeGlobalReports struct {
	Fake *FakeProjectcalico
}

var globalreportsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "", Resource: "globalreports"}

var globalreportsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "", Kind: "GlobalReport"}

// Get takes name of the globalReport, and returns the corresponding globalReport object, and an error if there is any.
func (c *FakeGlobalReports) Get(name string, options v1.GetOptions) (result *projectcalico.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalreportsResource, name), &projectcalico.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalReport), err
}

// List takes label and field selectors, and returns the list of GlobalReports that match those selectors.
func (c *FakeGlobalReports) List(opts v1.ListOptions) (result *projectcalico.GlobalReportList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalreportsResource, globalreportsKind, opts), &projectcalico.GlobalReportList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &projectcalico.GlobalReportList{ListMeta: obj.(*projectcalico.GlobalReportList).ListMeta}
	for _, item := range obj.(*projectcalico.GlobalReportList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalReports.
func (c *FakeGlobalReports) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalreportsResource, opts))
}

// Create takes the representation of a globalReport and creates it.  Returns the server's representation of the globalReport, and an error, if there is any.
func (c *FakeGlobalReports) Create(globalReport *projectcalico.GlobalReport) (result *projectcalico.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalreportsResource, globalReport), &projectcalico.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalReport), err
}

// Update takes the representation of a globalReport and updates it. Returns the server's representation of the globalReport, and an error, if there is any.
func (c *FakeGlobalReports) Update(globalReport *projectcalico.GlobalReport) (result *projectcalico.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalreportsResource, globalReport), &projectcalico.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalReport), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGlobalReports) UpdateStatus(globalReport *projectcalico.GlobalReport) (*projectcalico.GlobalReport, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(globalreportsResource, "status", globalReport), &projectcalico.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalReport), err
}

// Delete takes name of the globalReport and deletes it. Returns an error if one occurs.
func (c *FakeGlobalReports) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalreportsResource, name), &projectcalico.GlobalReport{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalReports) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalreportsResource, listOptions)

	_, err := c.Fake.Invokes(action, &projectcalico.GlobalReportList{})
	return err
}

// Patch applies the patch and returns the patched globalReport.
func (c *FakeGlobalReports) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *projectcalico.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalreportsResource, name, pt, data, subresources...), &projectcalico.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*projectcalico.GlobalReport), err
}
