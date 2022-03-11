// Code generated by mockery 2.9.0. DO NOT EDIT.

package controller

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// ADJobController is an autogenerated mock type for the ADJobController type
type MockADJobController struct {
	mock.Mock
}

// AddToManagedJobs provides a mock function with given fields: resource
func (_m *MockADJobController) AddToManagedJobs(resource interface{}) error {
	ret := _m.Called(resource)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(resource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *MockADJobController) Close() {
	_m.Called()
}

// RemoveManagedJob provides a mock function with given fields: resourceName
func (_m *MockADJobController) RemoveManagedJob(resourceName string) {
	_m.Called(resourceName)
}

// Run provides a mock function with given fields: ctx
func (_m *MockADJobController) Run(ctx context.Context) {
	_m.Called(ctx)
}
