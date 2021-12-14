// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// UISettingsGroupInterface has methods to work with UISettingsGroup resources.
type UISettingsGroupInterface interface {
	Create(ctx context.Context, res *apiv3.UISettingsGroup, opts options.SetOptions) (*apiv3.UISettingsGroup, error)
	Update(ctx context.Context, res *apiv3.UISettingsGroup, opts options.SetOptions) (*apiv3.UISettingsGroup, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.UISettingsGroup, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.UISettingsGroup, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.UISettingsGroupList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// uisettingsgroups implements UISettingsGroupInterface
type uisettingsgroups struct {
	client client
}

// Create takes the representation of a UISettingsGroup and creates it.  Returns the stored
// representation of the UISettingsGroup, and an error, if there is any.
func (r uisettingsgroups) Create(
	ctx context.Context, res *apiv3.UISettingsGroup, opts options.SetOptions,
) (*apiv3.UISettingsGroup, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}
	out, err := r.client.resources.Create(ctx, opts, apiv3.KindUISettingsGroup, res)
	if out != nil {
		return out.(*apiv3.UISettingsGroup), err
	}
	return nil, err
}

// Update takes the representation of a UISettingsGroup and updates it. Returns the stored
// representation of the UISettingsGroup, and an error, if there is any.
func (r uisettingsgroups) Update(
	ctx context.Context, res *apiv3.UISettingsGroup, opts options.SetOptions,
) (*apiv3.UISettingsGroup, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}
	out, err := r.client.resources.Update(ctx, opts, apiv3.KindUISettingsGroup, res)
	if out != nil {
		return out.(*apiv3.UISettingsGroup), err
	}
	return nil, err
}

// Delete takes name of the UISettingsGroup and deletes it. Returns an error if one occurs.
func (r uisettingsgroups) Delete(
	ctx context.Context, name string, opts options.DeleteOptions,
) (*apiv3.UISettingsGroup, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindUISettingsGroup, noNamespace, name)
	if out != nil {
		return out.(*apiv3.UISettingsGroup), err
	}
	return nil, err
}

// Get takes name of the UISettingsGroup, and returns the corresponding UISettingsGroup object,
// and an error if there is any.
func (r uisettingsgroups) Get(
	ctx context.Context, name string, opts options.GetOptions,
) (*apiv3.UISettingsGroup, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindUISettingsGroup, noNamespace, name)
	if out != nil {
		return out.(*apiv3.UISettingsGroup), err
	}
	return nil, err
}

// List returns the list of UISettingsGroup objects that match the supplied options.
func (r uisettingsgroups) List(
	ctx context.Context, opts options.ListOptions,
) (*apiv3.UISettingsGroupList, error) {
	res := &apiv3.UISettingsGroupList{}
	if err := r.client.resources.List(
		ctx, opts, apiv3.KindUISettingsGroup, apiv3.KindUISettingsGroupList, res,
	); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the uisettingsgroup that match the
// supplied options.
func (r uisettingsgroups) Watch(
	ctx context.Context, opts options.ListOptions,
) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindUISettingsGroup, nil)
}
