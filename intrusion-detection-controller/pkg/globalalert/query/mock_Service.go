// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

// Code generated by mockery v2.14.0. DO NOT EDIT.

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

// DeleteElasticWatchers provides a mock function with given fields: _a0
func (_m *MockService) DeleteElasticWatchers(_a0 context.Context) {
	_m.Called(_a0)
}

// ExecuteAlert provides a mock function with given fields: _a0, _a1
func (_m *MockService) ExecuteAlert(_a1 context.Context, _a0 *v3.GlobalAlert) v3.GlobalAlertStatus {
	ret := _m.Called(_a0, _a1)

	var r0 v3.GlobalAlertStatus
	if rf, ok := ret.Get(0).(func(*v3.GlobalAlert, context.Context) v3.GlobalAlertStatus); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(v3.GlobalAlertStatus)
	}

	return r0
}

type mockConstructorTestingTNewMockService interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockService creates a new instance of MockService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockService(t mockConstructorTestingTNewMockService) *MockService {
	mock := &MockService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
