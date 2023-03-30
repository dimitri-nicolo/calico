// Code generated by mockery v2.14.0. DO NOT EDIT.

package alert

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockForwarder is an autogenerated mock type for the Forwarder type
type MockForwarder struct {
	mock.Mock
}

// Forward provides a mock function with given fields: item
func (_m *MockForwarder) Forward(item v1.Event) {
	_m.Called(item)
}

// Run provides a mock function with given fields: ctx
func (_m *MockForwarder) Run(ctx context.Context) {
	_m.Called(ctx)
}

type mockConstructorTestingTNewMockForwarder interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockForwarder creates a new instance of MockForwarder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockForwarder(t mockConstructorTestingTNewMockForwarder) *MockForwarder {
	mock := &MockForwarder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
