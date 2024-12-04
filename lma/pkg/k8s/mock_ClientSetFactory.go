// Code generated by mockery v2.46.3. DO NOT EDIT.

package k8s

import (
	mock "github.com/stretchr/testify/mock"
	user "k8s.io/apiserver/pkg/authentication/user"
	rest "k8s.io/client-go/rest"
)

// MockClientSetFactory is an autogenerated mock type for the ClientSetFactory type
type MockClientSetFactory struct {
	mock.Mock
}

// Impersonate provides a mock function with given fields: _a0
func (_m *MockClientSetFactory) Impersonate(_a0 *user.DefaultInfo) ClientSetFactory {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Impersonate")
	}

	var r0 ClientSetFactory
	if rf, ok := ret.Get(0).(func(*user.DefaultInfo) ClientSetFactory); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ClientSetFactory)
		}
	}

	return r0
}

// NewClientSetForApplication provides a mock function with given fields: cluster
func (_m *MockClientSetFactory) NewClientSetForApplication(cluster string) (ClientSet, error) {
	ret := _m.Called(cluster)

	if len(ret) == 0 {
		panic("no return value specified for NewClientSetForApplication")
	}

	var r0 ClientSet
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (ClientSet, error)); ok {
		return rf(cluster)
	}
	if rf, ok := ret.Get(0).(func(string) ClientSet); ok {
		r0 = rf(cluster)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ClientSet)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(cluster)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewClientSetForUser provides a mock function with given fields: _a0, clusterID
func (_m *MockClientSetFactory) NewClientSetForUser(_a0 user.Info, clusterID string) (ClientSet, error) {
	ret := _m.Called(_a0, clusterID)

	if len(ret) == 0 {
		panic("no return value specified for NewClientSetForUser")
	}

	var r0 ClientSet
	var r1 error
	if rf, ok := ret.Get(0).(func(user.Info, string) (ClientSet, error)); ok {
		return rf(_a0, clusterID)
	}
	if rf, ok := ret.Get(0).(func(user.Info, string) ClientSet); ok {
		r0 = rf(_a0, clusterID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ClientSet)
		}
	}

	if rf, ok := ret.Get(1).(func(user.Info, string) error); ok {
		r1 = rf(_a0, clusterID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewRestConfigForApplication provides a mock function with given fields: clusterID
func (_m *MockClientSetFactory) NewRestConfigForApplication(clusterID string) *rest.Config {
	ret := _m.Called(clusterID)

	if len(ret) == 0 {
		panic("no return value specified for NewRestConfigForApplication")
	}

	var r0 *rest.Config
	if rf, ok := ret.Get(0).(func(string) *rest.Config); ok {
		r0 = rf(clusterID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rest.Config)
		}
	}

	return r0
}

// NewMockClientSetFactory creates a new instance of MockClientSetFactory. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClientSetFactory(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClientSetFactory {
	mock := &MockClientSetFactory{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
