// Code generated by mockery v2.14.0. DO NOT EDIT.

package elastic

import mock "github.com/stretchr/testify/mock"

// MockFlowFilter is an autogenerated mock type for the FlowFilter type
type MockFlowFilter struct {
	mock.Mock
}

// IncludeFlow provides a mock function with given fields: flow
func (_m *MockFlowFilter) IncludeFlow(flow *CompositeAggregationBucket) (bool, error) {
	ret := _m.Called(flow)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*CompositeAggregationBucket) bool); ok {
		r0 = rf(flow)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*CompositeAggregationBucket) error); ok {
		r1 = rf(flow)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ModifyFlow provides a mock function with given fields: flow
func (_m *MockFlowFilter) ModifyFlow(flow *CompositeAggregationBucket) error {
	ret := _m.Called(flow)

	var r0 error
	if rf, ok := ret.Get(0).(func(*CompositeAggregationBucket) error); ok {
		r0 = rf(flow)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewMockFlowFilter interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockFlowFilter creates a new instance of MockFlowFilter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockFlowFilter(t mockConstructorTestingTNewMockFlowFilter) *MockFlowFilter {
	mock := &MockFlowFilter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
