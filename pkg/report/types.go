package report

import (
	"fmt"
	"time"

	"github.com/satori/go.uuid"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

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
