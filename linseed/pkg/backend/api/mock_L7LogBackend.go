// Code generated by mockery v2.14.0. DO NOT EDIT.

package api

import (
	context "context"

	elastic "github.com/olivere/elastic/v7"
	mock "github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockL7LogBackend is an autogenerated mock type for the L7LogBackend type
type MockL7LogBackend struct {
	mock.Mock
}

// Aggregations provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockL7LogBackend) Aggregations(_a0 context.Context, _a1 ClusterInfo, _a2 *v1.L7AggregationParams) (*elastic.Aggregations, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *elastic.Aggregations
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.L7AggregationParams) *elastic.Aggregations); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*elastic.Aggregations)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, *v1.L7AggregationParams) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Create provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockL7LogBackend) Create(_a0 context.Context, _a1 ClusterInfo, _a2 []v1.L7Log) (*v1.BulkResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.BulkResponse
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.L7Log) *v1.BulkResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BulkResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, []v1.L7Log) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockL7LogBackend) List(_a0 context.Context, _a1 ClusterInfo, _a2 *v1.L7LogParams) (*v1.List[v1.L7Log], error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.List[v1.L7Log]
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.L7LogParams) *v1.List[v1.L7Log]); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.List[v1.L7Log])
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, *v1.L7LogParams) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockL7LogBackend interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockL7LogBackend creates a new instance of MockL7LogBackend. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockL7LogBackend(t mockConstructorTestingTNewMockL7LogBackend) *MockL7LogBackend {
	mock := &MockL7LogBackend{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
