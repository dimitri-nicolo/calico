// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// AlertExceptionInterface has methods to work with AlertException resources.
type AlertExceptionInterface interface {
	Create(ctx context.Context, res *apiv3.AlertException, opts options.SetOptions) (*apiv3.AlertException, error)
	Update(ctx context.Context, res *apiv3.AlertException, opts options.SetOptions) (*apiv3.AlertException, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.AlertException, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.AlertException, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.AlertExceptionList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// alertExceptions implements AlertExceptionInterface
type alertExceptions struct {
	client client
}

// Create takes the representation of a AlertException and creates it.  Returns the stored
// representation of the AlertException, and an error, if there is any.
func (r alertExceptions) Create(ctx context.Context, res *apiv3.AlertException, opts options.SetOptions) (*apiv3.AlertException, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindAlertException, res)
	if out != nil {
		return out.(*apiv3.AlertException), err
	}
	return nil, err
}

// Update takes the representation of a AlertException and updates it. Returns the stored
// representation of the AlertException, and an error, if there is any.
func (r alertExceptions) Update(ctx context.Context, res *apiv3.AlertException, opts options.SetOptions) (*apiv3.AlertException, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindAlertException, res)
	if out != nil {
		return out.(*apiv3.AlertException), err
	}
	return nil, err
}

// Delete takes name of the AlertException and deletes it. Returns an error if one occurs.
func (r alertExceptions) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.AlertException, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindAlertException, noNamespace, name)
	if out != nil {
		return out.(*apiv3.AlertException), err
	}
	return nil, err
}

// Get takes name of the AlertException, and returns the corresponding AlertException object,
// and an error if there is any.
func (r alertExceptions) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.AlertException, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindAlertException, noNamespace, name)
	if out != nil {
		return out.(*apiv3.AlertException), err
	}
	return nil, err
}

// List returns the list of AlertException objects that match the supplied options.
func (r alertExceptions) List(ctx context.Context, opts options.ListOptions) (*apiv3.AlertExceptionList, error) {
	res := &apiv3.AlertExceptionList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindAlertException, apiv3.KindAlertExceptionList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the AlertExceptions that match the
// supplied options.
func (r alertExceptions) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindAlertException, nil)
}
