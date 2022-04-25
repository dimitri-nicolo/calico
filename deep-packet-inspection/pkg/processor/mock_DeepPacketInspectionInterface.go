// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package processor

import (
	"context"

	"github.com/stretchr/testify/mock"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
	watch "github.com/projectcalico/calico/libcalico-go/lib/watch"
)

type MockDeepPacketInspectionInterface struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, res, opts
func (_m *MockDeepPacketInspectionInterface) Create(ctx context.Context, res *v3.DeepPacketInspection, opts options.SetOptions) (*v3.DeepPacketInspection, error) {
	return nil, nil
}

// Delete provides a mock function with given fields: ctx, namespace, name, opts
func (_m *MockDeepPacketInspectionInterface) Delete(ctx context.Context, namespace string, name string, opts options.DeleteOptions) (*v3.DeepPacketInspection, error) {
	return nil, nil
}

// Get provides a mock function with given fields: ctx, namespace, name, opts
func (_m *MockDeepPacketInspectionInterface) Get(ctx context.Context, namespace string, name string, opts options.GetOptions) (*v3.DeepPacketInspection, error) {
	ret := _m.Called(ctx, namespace, name, opts)

	var r0 *v3.DeepPacketInspection
	if rf, ok := ret.Get(0).(func(context.Context, string, string, options.GetOptions) *v3.DeepPacketInspection); ok {
		r0 = rf(ctx, namespace, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.DeepPacketInspection)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, options.GetOptions) error); ok {
		r1 = rf(ctx, namespace, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: ctx, opts
func (_m *MockDeepPacketInspectionInterface) List(ctx context.Context, opts options.ListOptions) (*v3.DeepPacketInspectionList, error) {
	return nil, nil
}

// Update provides a mock function with given fields: ctx, res, opts
func (_m *MockDeepPacketInspectionInterface) Update(ctx context.Context, res *v3.DeepPacketInspection, opts options.SetOptions) (*v3.DeepPacketInspection, error) {
	return nil, nil
}

// UpdateStatus provides a mock function with given fields: ctx, res, opts
func (_m *MockDeepPacketInspectionInterface) UpdateStatus(ctx context.Context, res *v3.DeepPacketInspection, opts options.SetOptions) (*v3.DeepPacketInspection, error) {
	ret := _m.Called(ctx, res, opts)

	var r0 *v3.DeepPacketInspection
	if rf, ok := ret.Get(0).(func(context.Context, *v3.DeepPacketInspection, options.SetOptions) *v3.DeepPacketInspection); ok {
		r0 = rf(ctx, res, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.DeepPacketInspection)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v3.DeepPacketInspection, options.SetOptions) error); ok {
		r1 = rf(ctx, res, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *MockDeepPacketInspectionInterface) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return nil, nil
}
