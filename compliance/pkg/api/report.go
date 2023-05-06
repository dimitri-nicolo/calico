package api

import (
	"context"
	"time"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lma "github.com/projectcalico/calico/lma/pkg/api"
)

type ReportRetriever interface {
	RetrieveArchivedReportTypeAndNames(cxt context.Context, q ReportQueryParams) ([]ReportTypeAndName, error)
	RetrieveArchivedReportSummaries(cxt context.Context, q ReportQueryParams) (*ArchivedReportSummaries, error)
	RetrieveArchivedReport(ctx context.Context, id string) (*v1.ReportData, error)
	RetrieveLastArchivedReportSummary(ctx context.Context, name string) (*v1.ReportData, error)
}

type ReportStorer interface {
	StoreArchivedReport(r *v1.ReportData) error
}

type AuditLogReportHandler interface {
	SearchAuditEvents(ctx context.Context, filter *apiv3.AuditEventsSelection, start, end *time.Time) <-chan *AuditEventResult
}

type FlowLogReportHandler interface {
	SearchFlows(ctx context.Context, namespaces []string, start, end *time.Time) <-chan *lma.FlowLogResult
}

type ReportQueryParams struct {
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
	SortBy []ReportSortBy
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
	Reports []*v1.ReportData
}

type ReportSortBy struct {
	// The field to sort by.
	Field string

	// Whether the sort should be ascending (true) or descending (false).
	Ascending bool
}

func NewArchivedReport(reportData *apiv3.ReportData, uiSummary string) *v1.ReportData {
	return &v1.ReportData{ReportData: reportData, UISummary: uiSummary}
}
