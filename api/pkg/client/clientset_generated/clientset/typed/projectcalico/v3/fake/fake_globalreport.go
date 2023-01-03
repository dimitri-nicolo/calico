// Copyright (c) 2023 Tigera, Inc. All rights reserved.

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

// FakeGlobalReports implements GlobalReportInterface
type FakeGlobalReports struct {
	Fake *FakeProjectcalicoV3
}

var globalreportsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "globalreports"}

var globalreportsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "GlobalReport"}

// Get takes name of the globalReport, and returns the corresponding globalReport object, and an error if there is any.
func (c *FakeGlobalReports) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalreportsResource, name), &v3.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReport), err
}

// List takes label and field selectors, and returns the list of GlobalReports that match those selectors.
func (c *FakeGlobalReports) List(ctx context.Context, opts v1.ListOptions) (result *v3.GlobalReportList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalreportsResource, globalreportsKind, opts), &v3.GlobalReportList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.GlobalReportList{ListMeta: obj.(*v3.GlobalReportList).ListMeta}
	for _, item := range obj.(*v3.GlobalReportList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalReports.
func (c *FakeGlobalReports) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalreportsResource, opts))
}

// Create takes the representation of a globalReport and creates it.  Returns the server's representation of the globalReport, and an error, if there is any.
func (c *FakeGlobalReports) Create(ctx context.Context, globalReport *v3.GlobalReport, opts v1.CreateOptions) (result *v3.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalreportsResource, globalReport), &v3.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReport), err
}

// Update takes the representation of a globalReport and updates it. Returns the server's representation of the globalReport, and an error, if there is any.
func (c *FakeGlobalReports) Update(ctx context.Context, globalReport *v3.GlobalReport, opts v1.UpdateOptions) (result *v3.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalreportsResource, globalReport), &v3.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReport), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGlobalReports) UpdateStatus(ctx context.Context, globalReport *v3.GlobalReport, opts v1.UpdateOptions) (*v3.GlobalReport, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(globalreportsResource, "status", globalReport), &v3.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReport), err
}

// Delete takes name of the globalReport and deletes it. Returns an error if one occurs.
func (c *FakeGlobalReports) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(globalreportsResource, name, opts), &v3.GlobalReport{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalReports) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalreportsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.GlobalReportList{})
	return err
}

// Patch applies the patch and returns the patched globalReport.
func (c *FakeGlobalReports) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalReport, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalreportsResource, name, pt, data, subresources...), &v3.GlobalReport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReport), err
}
