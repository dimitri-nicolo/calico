// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	watch "k8s.io/apimachinery/pkg/watch"
)

// GlobalNetworkSetInterface is an autogenerated mock type for the GlobalNetworkSetInterface type
type GlobalNetworkSetInterface struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, globalNetworkSet, opts
func (_m *GlobalNetworkSetInterface) Create(ctx context.Context, globalNetworkSet *v3.GlobalNetworkSet, opts v1.CreateOptions) (*v3.GlobalNetworkSet, error) {
	ret := _m.Called(ctx, globalNetworkSet, opts)

	var r0 *v3.GlobalNetworkSet
	if rf, ok := ret.Get(0).(func(context.Context, *v3.GlobalNetworkSet, v1.CreateOptions) *v3.GlobalNetworkSet); ok {
		r0 = rf(ctx, globalNetworkSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.GlobalNetworkSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v3.GlobalNetworkSet, v1.CreateOptions) error); ok {
		r1 = rf(ctx, globalNetworkSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *GlobalNetworkSetInterface) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *GlobalNetworkSetInterface) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, v1.DeleteOptions, v1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *GlobalNetworkSetInterface) Get(ctx context.Context, name string, opts v1.GetOptions) (*v3.GlobalNetworkSet, error) {
	ret := _m.Called(ctx, name, opts)

	var r0 *v3.GlobalNetworkSet
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.GetOptions) *v3.GlobalNetworkSet); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.GlobalNetworkSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, v1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: ctx, opts
func (_m *GlobalNetworkSetInterface) List(ctx context.Context, opts v1.ListOptions) (*v3.GlobalNetworkSetList, error) {
	ret := _m.Called(ctx, opts)

	var r0 *v3.GlobalNetworkSetList
	if rf, ok := ret.Get(0).(func(context.Context, v1.ListOptions) *v3.GlobalNetworkSetList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.GlobalNetworkSetList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, v1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *GlobalNetworkSetInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (*v3.GlobalNetworkSet, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, data, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *v3.GlobalNetworkSet
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) *v3.GlobalNetworkSet); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.GlobalNetworkSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, globalNetworkSet, opts
func (_m *GlobalNetworkSetInterface) Update(ctx context.Context, globalNetworkSet *v3.GlobalNetworkSet, opts v1.UpdateOptions) (*v3.GlobalNetworkSet, error) {
	ret := _m.Called(ctx, globalNetworkSet, opts)

	var r0 *v3.GlobalNetworkSet
	if rf, ok := ret.Get(0).(func(context.Context, *v3.GlobalNetworkSet, v1.UpdateOptions) *v3.GlobalNetworkSet); ok {
		r0 = rf(ctx, globalNetworkSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.GlobalNetworkSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v3.GlobalNetworkSet, v1.UpdateOptions) error); ok {
		r1 = rf(ctx, globalNetworkSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *GlobalNetworkSetInterface) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

	var r0 watch.Interface
	if rf, ok := ret.Get(0).(func(context.Context, v1.ListOptions) watch.Interface); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(watch.Interface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, v1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewGlobalNetworkSetInterface interface {
	mock.TestingT
	Cleanup(func())
}

// NewGlobalNetworkSetInterface creates a new instance of GlobalNetworkSetInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGlobalNetworkSetInterface(t mockConstructorTestingTNewGlobalNetworkSetInterface) *GlobalNetworkSetInterface {
	mock := &GlobalNetworkSetInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
