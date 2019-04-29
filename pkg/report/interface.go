package report

import (
	"context"
	"time"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/event"
	"github.com/tigera/compliance/pkg/flow"
)

type ReportRetriever interface {
	RetrieveArchivedReport(id string) (*ArchivedReportData, error)
	RetrieveArchivedReportSummaries() ([]*ArchivedReportData, error)
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
