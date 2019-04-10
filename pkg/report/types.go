package report

import (
	"fmt"
	"time"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type ArchivedReportData struct {
	*apiv3.ReportData `json:",inline"`
	UISummary         string `json:"uiSummary"`
}

func (r *ArchivedReportData) UID() string {
	return fmt.Sprintf("%s::%s::%s", r.ReportData.ReportName, r.ReportData.StartTime.Format(time.RFC3339), r.ReportData.EndTime.Format(time.RFC3339))
}

func NewArchivedReport(reportData *apiv3.ReportData, uiSummary string) *ArchivedReportData {
	return &ArchivedReportData{reportData, uiSummary}
}
