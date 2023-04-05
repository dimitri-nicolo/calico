// Code generated by mockery v2.14.0. DO NOT EDIT.

package processor

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	model "github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

// MockProcessor is an autogenerated mock type for the Processor type
type MockProcessor struct {
	mock.Mock
}

// Add provides a mock function with given fields: ctx, wepKey, iface
func (_m *MockProcessor) Add(ctx context.Context, wepKey model.WorkloadEndpointKey, iface string) {
	_m.Called(ctx, wepKey, iface)
}

// Close provides a mock function with given fields:
func (_m *MockProcessor) Close() {
	_m.Called()
}

// Remove provides a mock function with given fields: wepKey
func (_m *MockProcessor) Remove(wepKey model.WorkloadEndpointKey) {
	_m.Called(wepKey)
}

// WEPInterfaceCount provides a mock function with given fields:
func (_m *MockProcessor) WEPInterfaceCount() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

type mockConstructorTestingTNewMockProcessor interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockProcessor creates a new instance of MockProcessor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockProcessor(t mockConstructorTestingTNewMockProcessor) *MockProcessor {
	mock := &MockProcessor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
