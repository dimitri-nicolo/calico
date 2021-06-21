// Code generated by mockery v2.3.0. DO NOT EDIT.

package capture

import (
	mock "github.com/stretchr/testify/mock"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
)

// MockLocator is an autogenerated mock type for the Locator type
type MockLocator struct {
	mock.Mock
}

// GetEntryPod provides a mock function with given fields: clusterID, node
func (_m *MockLocator) GetEntryPod(clusterID string, node string) (string, string, error) {
	ret := _m.Called(clusterID, node)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(clusterID, node)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(string, string) string); ok {
		r1 = rf(clusterID, node)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string) error); ok {
		r2 = rf(clusterID, node)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetPacketCapture provides a mock function with given fields: clusterID, name, namespace
func (_m *MockLocator) GetPacketCapture(clusterID string, name string, namespace string) (*v3.PacketCapture, error) {
	ret := _m.Called(clusterID, name, namespace)

	var r0 *v3.PacketCapture
	if rf, ok := ret.Get(0).(func(string, string, string) *v3.PacketCapture); ok {
		r0 = rf(clusterID, name, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v3.PacketCapture)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string) error); ok {
		r1 = rf(clusterID, name, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
