// Code generated by mockery v2.46.3. DO NOT EDIT.

package api

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

// MockSnapshotsBackend is an autogenerated mock type for the SnapshotsBackend type
type MockSnapshotsBackend struct {
	mock.Mock
}

// Create provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockSnapshotsBackend) Create(_a0 context.Context, _a1 ClusterInfo, _a2 []v1.Snapshot) (*v1.BulkResponse, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *v1.BulkResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Snapshot) (*v1.BulkResponse, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, []v1.Snapshot) *v1.BulkResponse); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BulkResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, []v1.Snapshot) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockSnapshotsBackend) List(_a0 context.Context, _a1 ClusterInfo, _a2 *v1.SnapshotParams) (*v1.List[v1.Snapshot], error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *v1.List[v1.Snapshot]
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.SnapshotParams) (*v1.List[v1.Snapshot], error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ClusterInfo, *v1.SnapshotParams) *v1.List[v1.Snapshot]); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.List[v1.Snapshot])
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ClusterInfo, *v1.SnapshotParams) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockSnapshotsBackend creates a new instance of MockSnapshotsBackend. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSnapshotsBackend(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSnapshotsBackend {
	mock := &MockSnapshotsBackend{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
