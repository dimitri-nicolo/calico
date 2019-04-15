package report

import (
	"context"
	"time"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type ReportRetriever interface {
	RetrieveArchivedReport(id string) (*ArchivedReportData, error)
	RetrieveArchivedReportSummaries() ([]*ArchivedReportData, error)
}

type ReportStorer interface {
	StoreArchivedReport(r *ArchivedReportData) error
}

type AuditLogReportHandler interface {
	AddAuditEvents(ctx context.Context, data *v3.ReportData, filter *v3.AuditEventsSelection, start, end time.Time)
}
