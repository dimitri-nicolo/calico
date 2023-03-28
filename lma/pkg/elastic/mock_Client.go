// Code generated by mockery v2.14.0. DO NOT EDIT.

package elastic

import (
	context "context"

	api "github.com/projectcalico/calico/lma/pkg/api"

	mock "github.com/stretchr/testify/mock"

	time "time"

	v7 "github.com/olivere/elastic/v7"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// Backend provides a mock function with given fields:
func (_m *MockClient) Backend() *v7.Client {
	ret := _m.Called()

	var r0 *v7.Client
	if rf, ok := ret.Get(0).(func() *v7.Client); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v7.Client)
		}
	}

	return r0
}

// BulkProcessorClose provides a mock function with given fields:
func (_m *MockClient) BulkProcessorClose() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BulkProcessorFlush provides a mock function with given fields:
func (_m *MockClient) BulkProcessorFlush() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BulkProcessorInitialize provides a mock function with given fields: ctx, afterFn
func (_m *MockClient) BulkProcessorInitialize(ctx context.Context, afterFn v7.BulkAfterFunc) error {
	ret := _m.Called(ctx, afterFn)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, v7.BulkAfterFunc) error); ok {
		r0 = rf(ctx, afterFn)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BulkProcessorInitializeWithFlush provides a mock function with given fields: ctx, afterFn, bulkActions
func (_m *MockClient) BulkProcessorInitializeWithFlush(ctx context.Context, afterFn v7.BulkAfterFunc, bulkActions int) error {
	ret := _m.Called(ctx, afterFn, bulkActions)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, v7.BulkAfterFunc, int) error); ok {
		r0 = rf(ctx, afterFn, bulkActions)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClusterAlias provides a mock function with given fields: _a0
func (_m *MockClient) ClusterAlias(_a0 string) string {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ClusterIndex provides a mock function with given fields: _a0, _a1
func (_m *MockClient) ClusterIndex(_a0 string, _a1 string) string {
	ret := _m.Called(_a0, _a1)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// CreateEventsIndex provides a mock function with given fields: ctx
func (_m *MockClient) CreateEventsIndex(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteBulkSecurityEvent provides a mock function with given fields: index, id
func (_m *MockClient) DeleteBulkSecurityEvent(index string, id string) error {
	ret := _m.Called(index, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(index, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteSecurityEvent provides a mock function with given fields: ctx, index, id
func (_m *MockClient) DeleteSecurityEvent(ctx context.Context, index string, id string) (*v7.DeleteResponse, error) {
	ret := _m.Called(ctx, index, id)

	var r0 *v7.DeleteResponse
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v7.DeleteResponse); ok {
		r0 = rf(ctx, index, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v7.DeleteResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, index, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DismissBulkSecurityEvent provides a mock function with given fields: index, id
func (_m *MockClient) DismissBulkSecurityEvent(index string, id string) error {
	ret := _m.Called(index, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(index, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DismissSecurityEvent provides a mock function with given fields: ctx, index, id
func (_m *MockClient) DismissSecurityEvent(ctx context.Context, index string, id string) (*v7.UpdateResponse, error) {
	ret := _m.Called(ctx, index, id)

	var r0 *v7.UpdateResponse
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v7.UpdateResponse); ok {
		r0 = rf(ctx, index, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v7.UpdateResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, index, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Do provides a mock function with given fields: ctx, s
func (_m *MockClient) Do(ctx context.Context, s *v7.SearchService) (*v7.SearchResult, error) {
	ret := _m.Called(ctx, s)

	var r0 *v7.SearchResult
	if rf, ok := ret.Get(0).(func(context.Context, *v7.SearchService) *v7.SearchResult); ok {
		r0 = rf(ctx, s)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v7.SearchResult)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v7.SearchService) error); ok {
		r1 = rf(ctx, s)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EventsIndexExists provides a mock function with given fields: ctx
func (_m *MockClient) EventsIndexExists(ctx context.Context) (bool, error) {
	ret := _m.Called(ctx)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IndexTemplateName provides a mock function with given fields: index
func (_m *MockClient) IndexTemplateName(index string) string {
	ret := _m.Called(index)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(index)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// PutBulkSecurityEvent provides a mock function with given fields: data
func (_m *MockClient) PutBulkSecurityEvent(data api.EventsData) error {
	ret := _m.Called(data)

	var r0 error
	if rf, ok := ret.Get(0).(func(api.EventsData) error); ok {
		r0 = rf(data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PutSecurityEvent provides a mock function with given fields: ctx, data
func (_m *MockClient) PutSecurityEvent(ctx context.Context, data api.EventsData) (*v7.IndexResponse, error) {
	ret := _m.Called(ctx, data)

	var r0 *v7.IndexResponse
	if rf, ok := ret.Get(0).(func(context.Context, api.EventsData) *v7.IndexResponse); ok {
		r0 = rf(ctx, data)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v7.IndexResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, api.EventsData) error); ok {
		r1 = rf(ctx, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PutSecurityEventWithID provides a mock function with given fields: ctx, data, id
func (_m *MockClient) PutSecurityEventWithID(ctx context.Context, data api.EventsData, id string) (*v7.IndexResponse, error) {
	ret := _m.Called(ctx, data, id)

	var r0 *v7.IndexResponse
	if rf, ok := ret.Get(0).(func(context.Context, api.EventsData, string) *v7.IndexResponse); ok {
		r0 = rf(ctx, data, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v7.IndexResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, api.EventsData, string) error); ok {
		r1 = rf(ctx, data, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SearchADLogs provides a mock function with given fields: ctx, filter, start, end
func (_m *MockClient) SearchADLogs(ctx context.Context, filter *api.ADLogsSelection, start *time.Time, end *time.Time) <-chan *api.ADResult {
	ret := _m.Called(ctx, filter, start, end)

	var r0 <-chan *api.ADResult
	if rf, ok := ret.Get(0).(func(context.Context, *api.ADLogsSelection, *time.Time, *time.Time) <-chan *api.ADResult); ok {
		r0 = rf(ctx, filter, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.ADResult)
		}
	}

	return r0
}

// SearchCompositeAggregations provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockClient) SearchCompositeAggregations(_a0 context.Context, _a1 *CompositeAggregationQuery, _a2 CompositeAggregationKey) (<-chan *CompositeAggregationBucket, <-chan error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 <-chan *CompositeAggregationBucket
	if rf, ok := ret.Get(0).(func(context.Context, *CompositeAggregationQuery, CompositeAggregationKey) <-chan *CompositeAggregationBucket); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *CompositeAggregationBucket)
		}
	}

	var r1 <-chan error
	if rf, ok := ret.Get(1).(func(context.Context, *CompositeAggregationQuery, CompositeAggregationKey) <-chan error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(<-chan error)
		}
	}

	return r0, r1
}

// SearchDNSLogs provides a mock function with given fields: ctx, filter, start, end
func (_m *MockClient) SearchDNSLogs(ctx context.Context, filter *api.DNSLogsSelection, start *time.Time, end *time.Time) <-chan *api.DNSResult {
	ret := _m.Called(ctx, filter, start, end)

	var r0 <-chan *api.DNSResult
	if rf, ok := ret.Get(0).(func(context.Context, *api.DNSLogsSelection, *time.Time, *time.Time) <-chan *api.DNSResult); ok {
		r0 = rf(ctx, filter, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.DNSResult)
		}
	}

	return r0
}

// SearchSecurityEvents provides a mock function with given fields: ctx, start, end, filterData, allClusters
func (_m *MockClient) SearchSecurityEvents(ctx context.Context, start *time.Time, end *time.Time, filterData []api.EventsSearchFields, allClusters bool) <-chan *api.EventResult {
	ret := _m.Called(ctx, start, end, filterData, allClusters)

	var r0 <-chan *api.EventResult
	if rf, ok := ret.Get(0).(func(context.Context, *time.Time, *time.Time, []api.EventsSearchFields, bool) <-chan *api.EventResult); ok {
		r0 = rf(ctx, start, end, filterData, allClusters)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.EventResult)
		}
	}

	return r0
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
