// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package clientv3

import (
	"context"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/options"
	validator "github.com/projectcalico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/libcalico-go/lib/watch"
)

// DeepPacketInspectionInterface has methods to work with DPI resources.
type DeepPacketInspectionInterface interface {
	Create(ctx context.Context, res *apiv3.DeepPacketInspection, opts options.SetOptions) (*apiv3.DeepPacketInspection, error)
	Update(ctx context.Context, res *apiv3.DeepPacketInspection, opts options.SetOptions) (*apiv3.DeepPacketInspection, error)
	UpdateStatus(ctx context.Context, res *apiv3.DeepPacketInspection, opts options.SetOptions) (*apiv3.DeepPacketInspection, error)
	Delete(ctx context.Context, namespace, name string, opts options.DeleteOptions) (*apiv3.DeepPacketInspection, error)
	Get(ctx context.Context, namespace, name string, opts options.GetOptions) (*apiv3.DeepPacketInspection, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.DeepPacketInspectionList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// deepPacketInspections implements DeepPacketInspectionInterface
type deepPacketInspections struct {
	client client
}

// Create takes the representation of a DeepPacketInspection and creates it.  Returns the stored
// representation of the DeepPacketInspection, and an error, if there is any.
func (r deepPacketInspections) Create(ctx context.Context, res *apiv3.DeepPacketInspection, opts options.SetOptions) (*apiv3.DeepPacketInspection, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}
	out, err := r.client.resources.Create(ctx, opts, apiv3.KindDeepPacketInspection, res)
	if out != nil {
		return out.(*apiv3.DeepPacketInspection), err
	}
	return nil, err
}

// Update takes the representation of a DeepPacketInspection and updates it. Returns the stored
// representation of the DeepPacketInspection, and an error, if there is any.
func (r deepPacketInspections) Update(ctx context.Context, res *apiv3.DeepPacketInspection, opts options.SetOptions) (*apiv3.DeepPacketInspection, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}
	out, err := r.client.resources.Update(ctx, opts, apiv3.KindDeepPacketInspection, res)
	if out != nil {
		return out.(*apiv3.DeepPacketInspection), err
	}
	return nil, err
}

// UpdateStatus takes the representation of a DeepPacketInspection and updates the status section of it. Returns the stored
// representation of the DeepPacketInspection, and an error, if there is any.
func (r deepPacketInspections) UpdateStatus(ctx context.Context, res *apiv3.DeepPacketInspection, opts options.SetOptions) (*apiv3.DeepPacketInspection, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.UpdateStatus(ctx, opts, apiv3.KindDeepPacketInspection, res)
	if out != nil {
		return out.(*apiv3.DeepPacketInspection), err
	}
	return nil, err
}

// Delete takes name of the DeepPacketInspection and deletes it. Returns an error if one occurs.
func (r deepPacketInspections) Delete(ctx context.Context, namespace, name string, opts options.DeleteOptions) (*apiv3.DeepPacketInspection, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindDeepPacketInspection, namespace, name)
	if out != nil {
		return out.(*apiv3.DeepPacketInspection), err
	}
	return nil, err
}

// Get takes name of the DeepPacketInspection, and returns the corresponding DeepPacketInspection object,
// and an error if there is any.
func (r deepPacketInspections) Get(ctx context.Context, namespace, name string, opts options.GetOptions) (*apiv3.DeepPacketInspection, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindDeepPacketInspection, namespace, name)
	if out != nil {
		return out.(*apiv3.DeepPacketInspection), err
	}
	return nil, err
}

// List returns the list of DeepPacketInspection objects that match the supplied options.
func (r deepPacketInspections) List(ctx context.Context, opts options.ListOptions) (*apiv3.DeepPacketInspectionList, error) {
	res := &apiv3.DeepPacketInspectionList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindDeepPacketInspection, apiv3.KindDeepPacketInspectionList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the DeepPacketInspections that match the
// supplied options.
func (r deepPacketInspections) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindDeepPacketInspection, nil)
}
