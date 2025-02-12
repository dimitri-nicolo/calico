// Copyright (c) 2025 Tigera, Inc. All rights reserved.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeUISettingsGroups implements UISettingsGroupInterface
type FakeUISettingsGroups struct {
	Fake *FakeProjectcalicoV3
}

var uisettingsgroupsResource = v3.SchemeGroupVersion.WithResource("uisettingsgroups")

var uisettingsgroupsKind = v3.SchemeGroupVersion.WithKind("UISettingsGroup")

// Get takes name of the uISettingsGroup, and returns the corresponding uISettingsGroup object, and an error if there is any.
func (c *FakeUISettingsGroups) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.UISettingsGroup, err error) {
	emptyResult := &v3.UISettingsGroup{}
	obj, err := c.Fake.
		Invokes(testing.NewRootGetActionWithOptions(uisettingsgroupsResource, name, options), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.UISettingsGroup), err
}

// List takes label and field selectors, and returns the list of UISettingsGroups that match those selectors.
func (c *FakeUISettingsGroups) List(ctx context.Context, opts v1.ListOptions) (result *v3.UISettingsGroupList, err error) {
	emptyResult := &v3.UISettingsGroupList{}
	obj, err := c.Fake.
		Invokes(testing.NewRootListActionWithOptions(uisettingsgroupsResource, uisettingsgroupsKind, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.UISettingsGroupList{ListMeta: obj.(*v3.UISettingsGroupList).ListMeta}
	for _, item := range obj.(*v3.UISettingsGroupList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested uISettingsGroups.
func (c *FakeUISettingsGroups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchActionWithOptions(uisettingsgroupsResource, opts))
}

// Create takes the representation of a uISettingsGroup and creates it.  Returns the server's representation of the uISettingsGroup, and an error, if there is any.
func (c *FakeUISettingsGroups) Create(ctx context.Context, uISettingsGroup *v3.UISettingsGroup, opts v1.CreateOptions) (result *v3.UISettingsGroup, err error) {
	emptyResult := &v3.UISettingsGroup{}
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateActionWithOptions(uisettingsgroupsResource, uISettingsGroup, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.UISettingsGroup), err
}

// Update takes the representation of a uISettingsGroup and updates it. Returns the server's representation of the uISettingsGroup, and an error, if there is any.
func (c *FakeUISettingsGroups) Update(ctx context.Context, uISettingsGroup *v3.UISettingsGroup, opts v1.UpdateOptions) (result *v3.UISettingsGroup, err error) {
	emptyResult := &v3.UISettingsGroup{}
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateActionWithOptions(uisettingsgroupsResource, uISettingsGroup, opts), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.UISettingsGroup), err
}

// Delete takes name of the uISettingsGroup and deletes it. Returns an error if one occurs.
func (c *FakeUISettingsGroups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(uisettingsgroupsResource, name, opts), &v3.UISettingsGroup{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUISettingsGroups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionActionWithOptions(uisettingsgroupsResource, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v3.UISettingsGroupList{})
	return err
}

// Patch applies the patch and returns the patched uISettingsGroup.
func (c *FakeUISettingsGroups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.UISettingsGroup, err error) {
	emptyResult := &v3.UISettingsGroup{}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceActionWithOptions(uisettingsgroupsResource, name, pt, data, opts, subresources...), emptyResult)
	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v3.UISettingsGroup), err
}
