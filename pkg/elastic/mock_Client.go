// Code generated by mockery v2.3.0. DO NOT EDIT.

package elastic

import (
	context "context"

	"github.com/olivere/elastic/v7"
	api "github.com/tigera/lma/pkg/api"

	list "github.com/tigera/lma/pkg/list"

	mock "github.com/stretchr/testify/mock"

	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// Backend provides a mock function with given fields:
func (_m *MockClient) Backend() *elastic.Client {
	ret := _m.Called()

	var r0 *elastic.Client
	if rf, ok := ret.Get(0).(func() *elastic.Client); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*elastic.Client)
		}
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

// Do provides a mock function with given fields: ctx, s
func (_m *MockClient) Do(ctx context.Context, s *elastic.SearchService) (*elastic.SearchResult, error) {
	ret := _m.Called(ctx, s)

	var r0 *elastic.SearchResult
	if rf, ok := ret.Get(0).(func(context.Context, *elastic.SearchService) *elastic.SearchResult); ok {
		r0 = rf(ctx, s)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*elastic.SearchResult)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *elastic.SearchService) error); ok {
		r1 = rf(ctx, s)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAuditEvents provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockClient) GetAuditEvents(_a0 context.Context, _a1 *time.Time, _a2 *time.Time) <-chan *api.AuditEventResult {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 <-chan *api.AuditEventResult
	if rf, ok := ret.Get(0).(func(context.Context, *time.Time, *time.Time) <-chan *api.AuditEventResult); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.AuditEventResult)
		}
	}

	return r0
}

// GetBenchmarks provides a mock function with given fields: ctx, id
func (_m *MockClient) GetBenchmarks(ctx context.Context, id string) (*api.Benchmarks, error) {
	ret := _m.Called(ctx, id)

	var r0 *api.Benchmarks
	if rf, ok := ret.Get(0).(func(context.Context, string) *api.Benchmarks); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.Benchmarks)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveArchivedReport provides a mock function with given fields: id
func (_m *MockClient) RetrieveArchivedReport(id string) (*api.ArchivedReportData, error) {
	ret := _m.Called(id)

	var r0 *api.ArchivedReportData
	if rf, ok := ret.Get(0).(func(string) *api.ArchivedReportData); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ArchivedReportData)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveArchivedReportSummaries provides a mock function with given fields: cxt, q
func (_m *MockClient) RetrieveArchivedReportSummaries(cxt context.Context, q api.ReportQueryParams) (*api.ArchivedReportSummaries, error) {
	ret := _m.Called(cxt, q)

	var r0 *api.ArchivedReportSummaries
	if rf, ok := ret.Get(0).(func(context.Context, api.ReportQueryParams) *api.ArchivedReportSummaries); ok {
		r0 = rf(cxt, q)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ArchivedReportSummaries)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, api.ReportQueryParams) error); ok {
		r1 = rf(cxt, q)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveArchivedReportSummary provides a mock function with given fields: id
func (_m *MockClient) RetrieveArchivedReportSummary(id string) (*api.ArchivedReportData, error) {
	ret := _m.Called(id)

	var r0 *api.ArchivedReportData
	if rf, ok := ret.Get(0).(func(string) *api.ArchivedReportData); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ArchivedReportData)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveArchivedReportTypeAndNames provides a mock function with given fields: cxt, q
func (_m *MockClient) RetrieveArchivedReportTypeAndNames(cxt context.Context, q api.ReportQueryParams) ([]api.ReportTypeAndName, error) {
	ret := _m.Called(cxt, q)

	var r0 []api.ReportTypeAndName
	if rf, ok := ret.Get(0).(func(context.Context, api.ReportQueryParams) []api.ReportTypeAndName); ok {
		r0 = rf(cxt, q)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]api.ReportTypeAndName)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, api.ReportQueryParams) error); ok {
		r1 = rf(cxt, q)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveLastArchivedReportSummary provides a mock function with given fields: name
func (_m *MockClient) RetrieveLastArchivedReportSummary(name string) (*api.ArchivedReportData, error) {
	ret := _m.Called(name)

	var r0 *api.ArchivedReportData
	if rf, ok := ret.Get(0).(func(string) *api.ArchivedReportData); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ArchivedReportData)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveLatestBenchmarks provides a mock function with given fields: ctx, ct, filters, start, end
func (_m *MockClient) RetrieveLatestBenchmarks(ctx context.Context, ct api.BenchmarkType, filters []api.BenchmarkFilter, start time.Time, end time.Time) <-chan api.BenchmarksResult {
	ret := _m.Called(ctx, ct, filters, start, end)

	var r0 <-chan api.BenchmarksResult
	if rf, ok := ret.Get(0).(func(context.Context, api.BenchmarkType, []api.BenchmarkFilter, time.Time, time.Time) <-chan api.BenchmarksResult); ok {
		r0 = rf(ctx, ct, filters, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan api.BenchmarksResult)
		}
	}

	return r0
}

// RetrieveList provides a mock function with given fields: kind, from, to, sortAscendingTime
func (_m *MockClient) RetrieveList(kind v1.TypeMeta, from *time.Time, to *time.Time, sortAscendingTime bool) (*list.TimestampedResourceList, error) {
	ret := _m.Called(kind, from, to, sortAscendingTime)

	var r0 *list.TimestampedResourceList
	if rf, ok := ret.Get(0).(func(v1.TypeMeta, *time.Time, *time.Time, bool) *list.TimestampedResourceList); ok {
		r0 = rf(kind, from, to, sortAscendingTime)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*list.TimestampedResourceList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(v1.TypeMeta, *time.Time, *time.Time, bool) error); ok {
		r1 = rf(kind, from, to, sortAscendingTime)
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

// SearchAlertLogs provides a mock function with given fields: ctx, filter, start, end
func (_m *MockClient) SearchAlertLogs(ctx context.Context, filter *api.AlertLogsSelection, start *time.Time, end *time.Time) <-chan *api.AlertResult {
	ret := _m.Called(ctx, filter, start, end)

	var r0 <-chan *api.AlertResult
	if rf, ok := ret.Get(0).(func(context.Context, *api.AlertLogsSelection, *time.Time, *time.Time) <-chan *api.AlertResult); ok {
		r0 = rf(ctx, filter, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.AlertResult)
		}
	}

	return r0
}

// SearchAuditEvents provides a mock function with given fields: ctx, filter, start, end
func (_m *MockClient) SearchAuditEvents(ctx context.Context, filter *v3.AuditEventsSelection, start *time.Time, end *time.Time) <-chan *api.AuditEventResult {
	ret := _m.Called(ctx, filter, start, end)

	var r0 <-chan *api.AuditEventResult
	if rf, ok := ret.Get(0).(func(context.Context, *v3.AuditEventsSelection, *time.Time, *time.Time) <-chan *api.AuditEventResult); ok {
		r0 = rf(ctx, filter, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.AuditEventResult)
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

// SearchFlowLogs provides a mock function with given fields: ctx, namespaces, start, end
func (_m *MockClient) SearchFlowLogs(ctx context.Context, namespaces []string, start *time.Time, end *time.Time) <-chan *api.FlowLogResult {
	ret := _m.Called(ctx, namespaces, start, end)

	var r0 <-chan *api.FlowLogResult
	if rf, ok := ret.Get(0).(func(context.Context, []string, *time.Time, *time.Time) <-chan *api.FlowLogResult); ok {
		r0 = rf(ctx, namespaces, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *api.FlowLogResult)
		}
	}

	return r0
}

// StoreArchivedReport provides a mock function with given fields: r
func (_m *MockClient) StoreArchivedReport(r *api.ArchivedReportData) error {
	ret := _m.Called(r)

	var r0 error
	if rf, ok := ret.Get(0).(func(*api.ArchivedReportData) error); ok {
		r0 = rf(r)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StoreBenchmarks provides a mock function with given fields: ctx, b
func (_m *MockClient) StoreBenchmarks(ctx context.Context, b *api.Benchmarks) error {
	ret := _m.Called(ctx, b)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *api.Benchmarks) error); ok {
		r0 = rf(ctx, b)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StoreList provides a mock function with given fields: _a0, _a1
func (_m *MockClient) StoreList(_a0 v1.TypeMeta, _a1 *list.TimestampedResourceList) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(v1.TypeMeta, *list.TimestampedResourceList) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
