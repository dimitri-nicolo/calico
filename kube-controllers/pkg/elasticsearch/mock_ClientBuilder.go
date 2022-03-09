// Code generated by mockery v2.3.0. DO NOT EDIT.

package elasticsearch

import mock "github.com/stretchr/testify/mock"

// MockClientBuilder is an autogenerated mock type for the ClientBuilder type
type MockClientBuilder struct {
	mock.Mock
}

// Build provides a mock function with given fields:
func (_m *MockClientBuilder) Build() (Client, error) {
	ret := _m.Called()

	var r0 Client
	if rf, ok := ret.Get(0).(func() Client); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Client)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
