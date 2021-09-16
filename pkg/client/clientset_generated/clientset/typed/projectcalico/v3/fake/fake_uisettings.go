// Copyright (c) 2021 Tigera, Inc. All rights reserved.

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

// FakeUISettings implements UISettingsInterface
type FakeUISettings struct {
	Fake *FakeProjectcalicoV3
}

var uisettingsResource = schema.GroupVersionResource{Group: "projectcalico.org", Version: "v3", Resource: "uisettings"}

var uisettingsKind = schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "UISettings"}

// Get takes name of the uISettings, and returns the corresponding uISettings object, and an error if there is any.
func (c *FakeUISettings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v3.UISettings, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(uisettingsResource, name), &v3.UISettings{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.UISettings), err
}

// List takes label and field selectors, and returns the list of UISettings that match those selectors.
func (c *FakeUISettings) List(ctx context.Context, opts v1.ListOptions) (result *v3.UISettingsList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(uisettingsResource, uisettingsKind, opts), &v3.UISettingsList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.UISettingsList{ListMeta: obj.(*v3.UISettingsList).ListMeta}
	for _, item := range obj.(*v3.UISettingsList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested uISettings.
func (c *FakeUISettings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(uisettingsResource, opts))
}

// Create takes the representation of a uISettings and creates it.  Returns the server's representation of the uISettings, and an error, if there is any.
func (c *FakeUISettings) Create(ctx context.Context, uISettings *v3.UISettings, opts v1.CreateOptions) (result *v3.UISettings, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(uisettingsResource, uISettings), &v3.UISettings{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.UISettings), err
}

// Update takes the representation of a uISettings and updates it. Returns the server's representation of the uISettings, and an error, if there is any.
func (c *FakeUISettings) Update(ctx context.Context, uISettings *v3.UISettings, opts v1.UpdateOptions) (result *v3.UISettings, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(uisettingsResource, uISettings), &v3.UISettings{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.UISettings), err
}

// Delete takes name of the uISettings and deletes it. Returns an error if one occurs.
func (c *FakeUISettings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(uisettingsResource, name), &v3.UISettings{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUISettings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(uisettingsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v3.UISettingsList{})
	return err
}

// Patch applies the patch and returns the patched uISettings.
func (c *FakeUISettings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v3.UISettings, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(uisettingsResource, name, pt, data, subresources...), &v3.UISettings{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v3.UISettings), err
}
