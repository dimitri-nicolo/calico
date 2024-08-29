// Code generated by mockery v2.42.2. DO NOT EDIT.

package client

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockQueryInterface is an autogenerated mock type for the QueryInterface type
type MockQueryInterface struct {
	mock.Mock
}

// RunQuery provides a mock function with given fields: ctx, req
func (_m *MockQueryInterface) RunQuery(ctx context.Context, req interface{}) (interface{}, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for RunQuery")
	}

	var r0 interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, interface{}) (interface{}, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, interface{}) interface{}); ok {
		r0 = rf(ctx, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, interface{}) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockQueryInterface creates a new instance of MockQueryInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockQueryInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockQueryInterface {
	mock := &MockQueryInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
