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

// FakeGlobalReportTypes implements GlobalReportTypeInterface
type FakeGlobalReportTypes struct {
	Fake *FakeProjectcalicoV3
}

var globalreporttypesResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "globalreporttypes"}

var globalreporttypesKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "GlobalReportType"}

// Get takes name of the globalReportType, and returns the corresponding globalReportType object, and an error if there is any.
func (c *FakeGlobalReportTypes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.GlobalReportType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(globalreporttypesResource, name), &v3.GlobalReportType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReportType), err
}

// List takes label and field selectors, and returns the list of GlobalReportTypes that match those selectors.
func (c *FakeGlobalReportTypes) List(ctx context.Context, opts v1.ListOptions) (result *v3.GlobalReportTypeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(globalreporttypesResource, globalreporttypesKind, opts), &v3.GlobalReportTypeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.GlobalReportTypeList{ListMeta: obj.(*v3.GlobalReportTypeList).ListMeta}
	for _, item := range obj.(*v3.GlobalReportTypeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested globalReportTypes.
func (c *FakeGlobalReportTypes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(globalreporttypesResource, opts))
}

// Create takes the representation of a globalReportType and creates it.  Returns the server's representation of the globalReportType, and an error, if there is any.
func (c *FakeGlobalReportTypes) Create(ctx context.Context, globalReportType *v3.GlobalReportType, opts v1.CreateOptions) (result *v3.GlobalReportType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(globalreporttypesResource, globalReportType), &v3.GlobalReportType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReportType), err
}

// Update takes the representation of a globalReportType and updates it. Returns the server's representation of the globalReportType, and an error, if there is any.
func (c *FakeGlobalReportTypes) Update(ctx context.Context, globalReportType *v3.GlobalReportType, opts v1.UpdateOptions) (result *v3.GlobalReportType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(globalreporttypesResource, globalReportType), &v3.GlobalReportType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReportType), err
}

// Delete takes name of the globalReportType and deletes it. Returns an error if one occurs.
func (c *FakeGlobalReportTypes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(globalreporttypesResource, name), &v3.GlobalReportType{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGlobalReportTypes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(globalreporttypesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.GlobalReportTypeList{})
	return err
}

// Patch applies the patch and returns the patched globalReportType.
func (c *FakeGlobalReportTypes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.GlobalReportType, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(globalreporttypesResource, name, pt, data, subresources...), &v3.GlobalReportType{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.GlobalReportType), err
}
