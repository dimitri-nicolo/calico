// Code generated by mockery v2.14.0. DO NOT EDIT.

package api

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockFlowBackend is an autogenerated mock type for the FlowBackend type
type MockFlowBackend struct {
	mock.Mock
}

// List provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockFlowBackend) List(_a0 context.Context, _a1 ClusterInfo, _a2 v1.L3FlowParams) (*v1.L3FlowResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.L3FlowResponse
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, v1.L3FlowParams) *v1.L3FlowResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.L3FlowResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, v1.L3FlowParams) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockFlowBackend interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockFlowBackend creates a new instance of MockFlowBackend. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockFlowBackend(t mockConstructorTestingTNewMockFlowBackend) *MockFlowBackend {
	mock := &MockFlowBackend{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
