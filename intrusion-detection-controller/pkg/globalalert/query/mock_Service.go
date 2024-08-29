// Code generated by mockery v2.42.2. DO NOT EDIT.

package query

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// ExecuteAlert provides a mock function with given fields: _a0, _a1
func (_m *MockService) ExecuteAlert(_a0 context.Context, _a1 *v3.GlobalAlert) v3.GlobalAlertStatus {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for ExecuteAlert")
	}

	var r0 v3.GlobalAlertStatus
	if rf, ok := ret.Get(0).(func(context.Context, *v3.GlobalAlert) v3.GlobalAlertStatus); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(v3.GlobalAlertStatus)
	}

	return r0
}

// NewMockService creates a new instance of MockService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockService {
	mock := &MockService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
