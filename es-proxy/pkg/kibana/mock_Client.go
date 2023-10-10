// Code generated by mockery v2.27.1. DO NOT EDIT.

package kibana

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// Login provides a mock function with given fields: currentURL, username, password
func (_m *MockClient) Login(currentURL string, username string, password string) (*http.Response, error) {
	ret := _m.Called(currentURL, username, password)

	var r0 *http.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string, string) (*http.Response, error)); ok {
		return rf(currentURL, username, password)
	}
	if rf, ok := ret.Get(0).(func(string, string, string) *http.Response); ok {
		r0 = rf(currentURL, username, password)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*http.Response)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(currentURL, username, password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockClient creates a new instance of MockClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockClient(t mockConstructorTestingTNewMockClient) *MockClient {
	mock := &MockClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
