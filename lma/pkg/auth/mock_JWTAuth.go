// Code generated by mockery v2.27.1. DO NOT EDIT.

package auth

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"
	user "k8s.io/apiserver/pkg/authentication/user"

	v1 "k8s.io/api/authorization/v1"
)

// MockJWTAuth is an autogenerated mock type for the JWTAuth type
type MockJWTAuth struct {
	mock.Mock
}

// Authenticate provides a mock function with given fields: r
func (_m *MockJWTAuth) Authenticate(r *http.Request) (user.Info, int, error) {
	ret := _m.Called(r)

	var r0 user.Info
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(*http.Request) (user.Info, int, error)); ok {
		return rf(r)
	}
	if rf, ok := ret.Get(0).(func(*http.Request) user.Info); ok {
		r0 = rf(r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(user.Info)
		}
	}

	if rf, ok := ret.Get(1).(func(*http.Request) int); ok {
		r1 = rf(r)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(*http.Request) error); ok {
		r2 = rf(r)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Authorize provides a mock function with given fields: usr, resources, nonResources
func (_m *MockJWTAuth) Authorize(usr user.Info, resources *v1.ResourceAttributes, nonResources *v1.NonResourceAttributes) (bool, error) {
	ret := _m.Called(usr, resources, nonResources)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(user.Info, *v1.ResourceAttributes, *v1.NonResourceAttributes) (bool, error)); ok {
		return rf(usr, resources, nonResources)
	}
	if rf, ok := ret.Get(0).(func(user.Info, *v1.ResourceAttributes, *v1.NonResourceAttributes) bool); ok {
		r0 = rf(usr, resources, nonResources)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(user.Info, *v1.ResourceAttributes, *v1.NonResourceAttributes) error); ok {
		r1 = rf(usr, resources, nonResources)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockJWTAuth interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockJWTAuth creates a new instance of MockJWTAuth. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockJWTAuth(t mockConstructorTestingTNewMockJWTAuth) *MockJWTAuth {
	mock := &MockJWTAuth{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
