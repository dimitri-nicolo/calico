package report

import (
	"context"
	"time"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/flow"
)

type ReportRetriever interface {
	RetrieveArchivedReportTypeAndNames(cxt context.Context, q QueryParams) ([]ReportTypeAndName, error)
	RetrieveArchivedReportSummaries(cxt context.Context, q QueryParams) (*ArchivedReportSummaries, error)
	RetrieveArchivedReport(id string) (*ArchivedReportData, error)
	RetrieveArchivedReportSummary(id string) (*ArchivedReportData, error)
	RetrieveLastArchivedReportSummary(name string) (*ArchivedReportData, error)
}

type ReportStorer interface {
	StoreArchivedReport(r *ArchivedReportData, t time.Time) error
}

type AuditLogReportHandler interface {
	SearchAuditEvents(ctx context.Context, filter *apiv3.AuditEventsSelection, start, end *time.Time) <-chan *event.AuditEventResult
}

type FlowLogReportHandler interface {
	SearchFlowLogs(ctx context.Context, namespaces []string, start, end *time.Time) <-chan *flow.FlowLogResult
}

type QueryParams struct {
	// The set of report type and names that should be included in the query. Each entry may include just the report
	// name or the type in which case the other field is wildcarded.
	Reports []ReportTypeAndName

	// The from time for the query.
	FromTime string

	// The to time for the query.
	ToTime string

	// The page number indexed from 0. If MaxPerPage is nil, this should be 0.
	// Not used by ReportRetriever.RetrieveArchivedReportTypeAndNames.
	Page int

	// The maximum results to return. If set to nil, all results are returned. If set to 0 the default page size
	// is used.
	// Not used by ReportRetriever.RetrieveArchivedReportTypeAndNames.
	MaxItems *int

	// The set of sort fields and whether to sort ascending or descending. Documents are sorted in the natural order
	// in the slice.
	// Not used by ReportRetriever.RetrieveArchivedReportTypeAndNames.
	SortBy []SortBy
}

// ReportTypeAndName encapsulates a report name with its corresponding report type name.
type ReportTypeAndName struct {
	// The report type name.
	ReportTypeName string

	// The report name.
	ReportName string
}

// ArchivedReportSummaries is a set of report summaries with the number of pages available. Queries should have
// Page < NumPages.
type ArchivedReportSummaries struct {
	// The total number of results available.
	Count int

	// The set of report summaries.
	Reports []*ArchivedReportData
}

type SortBy struct {
	// The field to sort by.
	Field string

	// Whether the sort should be ascending (true) or descending (false).
	Ascending bool
}
