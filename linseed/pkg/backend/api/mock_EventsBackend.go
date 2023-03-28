// Code generated by mockery v2.14.0. DO NOT EDIT.

package api

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockEventsBackend is an autogenerated mock type for the EventsBackend type
type MockEventsBackend struct {
	mock.Mock
}

// Create provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockEventsBackend) Create(_a0 context.Context, _a1 ClusterInfo, _a2 []v1.Event) (*v1.BulkResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.BulkResponse
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Event) *v1.BulkResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BulkResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, []v1.Event) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockEventsBackend) Delete(_a0 context.Context, _a1 ClusterInfo, _a2 []v1.Event) (*v1.BulkResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.BulkResponse
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Event) *v1.BulkResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BulkResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, []v1.Event) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Dismiss provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockEventsBackend) Dismiss(_a0 context.Context, _a1 ClusterInfo, _a2 []v1.Event) (*v1.BulkResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.BulkResponse
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Event) *v1.BulkResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BulkResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, []v1.Event) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockEventsBackend) List(_a0 context.Context, _a1 ClusterInfo, _a2 *v1.EventParams) (*v1.List[v1.Event], error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.List[v1.Event]
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.EventParams) *v1.List[v1.Event]); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.List[v1.Event])
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, *v1.EventParams) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockEventsBackend interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockEventsBackend creates a new instance of MockEventsBackend. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockEventsBackend(t mockConstructorTestingTNewMockEventsBackend) *MockEventsBackend {
	mock := &MockEventsBackend{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
