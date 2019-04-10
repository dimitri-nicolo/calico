package report

import (
	"time"
)

type RawComplianceReport struct {
	ReportType  string      `json:"reportType"`
	ReportSpec  interface{} `json:"reportSpec"`
	StartTime   time.Time   `json:"startTime"`
	EndTime     time.Time   `json:"endTime"`
	UISummary   string      `json:"reportType"`
	Endpoints   interface{} `json:"endpoints"`
	Namespaces  interface{} `json:"namespaces"`
	Services    interface{} `json:"services"`
	AuditEvents interface{} `json:"auditEvents"`
}

type RawComplianceReportStore interface {
	GetRawComplianceReports() ([]*RawComplianceReport, error)
}
