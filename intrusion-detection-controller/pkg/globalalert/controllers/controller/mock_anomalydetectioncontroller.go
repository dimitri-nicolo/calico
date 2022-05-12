// Code generated by mockery 2.9.0. DO NOT EDIT.

package controller

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// AnomalyDetectionController is an autogenerated mock type for the AnomalyDetectionController type
type MockAnomalyDetectionController struct {
	mock.Mock
}

// AddDetector provides a mock function with given fields: resource
func (_m *MockAnomalyDetectionController) AddDetector(resource interface{}) error {
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
func (_m *MockAnomalyDetectionController) Close() {
	_m.Called()
}

// RemoveDetector provides a mock function with given fields: resource
func (_m *MockAnomalyDetectionController) RemoveDetector(resource interface{}) error {
	ret := _m.Called(resource)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(resource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Run provides a mock function with given fields: ctx
func (_m *MockAnomalyDetectionController) Run(ctx context.Context) {
	_m.Called(ctx)
}
