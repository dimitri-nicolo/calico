package report

import (
	"context"
	"time"

	"github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type ArchivedReportStore interface {
	RetrieveArchivedReportSummaries() ([]*ArchivedReportData, error)
	RetrieveArchivedReport(string) (*ArchivedReportData, error)
	StoreArchivedReport(*ArchivedReportData) error
}

type AuditLogReportHandler interface {
	AddAuditEvents(ctx context.Context, data *v3.ReportData, filter *v3.AuditEventsSelection, start, end time.Time)
}
