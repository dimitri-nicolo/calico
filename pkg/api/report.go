package api

import (
	"context"
	"fmt"
	"time"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	uuid "github.com/satori/go.uuid"
)

type ReportRetriever interface {
	RetrieveArchivedReportTypeAndNames(cxt context.Context, q ReportQueryParams) ([]ReportTypeAndName, error)
	RetrieveArchivedReportSummaries(cxt context.Context, q ReportQueryParams) (*ArchivedReportSummaries, error)
	RetrieveArchivedReport(id string) (*ArchivedReportData, error)
	RetrieveArchivedReportSummary(id string) (*ArchivedReportData, error)
	RetrieveLastArchivedReportSummary(name string) (*ArchivedReportData, error)
}

type ReportStorer interface {
	StoreArchivedReport(r *ArchivedReportData) error
}

type AuditLogReportHandler interface {
	SearchAuditEvents(ctx context.Context, filter *apiv3.AuditEventsSelection, start, end *time.Time) <-chan *AuditEventResult
}

type FlowLogReportHandler interface {
	SearchFlowLogs(ctx context.Context, namespaces []string, start, end *time.Time) <-chan *FlowLogResult
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
	Reports []*ArchivedReportData
}

type ReportSortBy struct {
	// The field to sort by.
	Field string

	// Whether the sort should be ascending (true) or descending (false).
	Ascending bool
}

type ArchivedReportData struct {
	*apiv3.ReportData `json:",inline"`
	UISummary         string `json:"uiSummary"`
}

func (r *ArchivedReportData) UID() string {
	name := fmt.Sprintf("%s::%s::%s", r.ReportData.ReportName, r.ReportData.StartTime.Format(time.RFC3339), r.ReportData.EndTime.Format(time.RFC3339))
	id := uuid.NewV5(uuid.NamespaceURL, name) //V5 uuids are deterministic

	// Encode the report name and report type name into the UID - we use this so that we can perform RBAC without
	// needing to download the report.
	return fmt.Sprintf("%s_%s_%s", r.ReportData.ReportName, r.ReportData.ReportTypeName, id.String())
}

func NewArchivedReport(reportData *apiv3.ReportData, uiSummary string) *ArchivedReportData {
	return &ArchivedReportData{reportData, uiSummary}
}
