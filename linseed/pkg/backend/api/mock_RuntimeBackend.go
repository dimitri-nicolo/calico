// Code generated by mockery v2.27.1. DO NOT EDIT.

package api

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockRuntimeBackend is an autogenerated mock type for the RuntimeBackend type
type MockRuntimeBackend struct {
	mock.Mock
}

// Create provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockRuntimeBackend) Create(_a0 context.Context, _a1 ClusterInfo, _a2 []v1.Report) (*v1.BulkResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.BulkResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Report) (*v1.BulkResponse, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Report) *v1.BulkResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BulkResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, []v1.Report) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockRuntimeBackend) List(_a0 context.Context, _a1 ClusterInfo, _a2 *v1.RuntimeReportParams) (*v1.List[v1.RuntimeReport], error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.List[v1.RuntimeReport]
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.RuntimeReportParams) (*v1.List[v1.RuntimeReport], error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.RuntimeReportParams) *v1.List[v1.RuntimeReport]); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.List[v1.RuntimeReport])
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, *v1.RuntimeReportParams) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockRuntimeBackend interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockRuntimeBackend creates a new instance of MockRuntimeBackend. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockRuntimeBackend(t mockConstructorTestingTNewMockRuntimeBackend) *MockRuntimeBackend {
	mock := &MockRuntimeBackend{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
