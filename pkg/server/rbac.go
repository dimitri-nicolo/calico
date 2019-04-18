package server

import "net/http"

// reportRbacHelper implements helper functionality to determine whether the API user is able to view
// and list reports. The helper is used for a single API request and will cache information that may be
// required multiple times.
type reportRbacHelper struct {
	canGetReportTypeByName map[string]bool
	canGetReportByName     map[string]bool
}

// newReportRbacHelper returns a new initialized reportRbacHelper.
func newReportRbacHelper(s *server, _ *http.Request) *reportRbacHelper {
	return &reportRbacHelper{
		canGetReportTypeByName: make(map[string]bool),
		canGetReportByName:     make(map[string]bool),
	}
}

// canListReports returns true if the caller is allowed to List Reports.
func (l *reportRbacHelper) canListReports() (bool, error) {
	return true, nil
}

// canViewReport returns true if the caller is allowed to view a specific Report and ReportType.
func (l *reportRbacHelper) canViewReport(reportType, report string) (bool, error) {
	var err error
	canGetReport, ok := l.canGetReportByName[report]
	if !ok {
		// Query to determine if the user can get the report.
		canGetReport, err = l.canGetReport(report)
		if err != nil {
			return false, err
		}
		l.canGetReportByName[report] = canGetReport
	}
	if !canGetReport {
		return false, nil
	}
	canGetReportType, ok := l.canGetReportTypeByName[report]
	if !ok {
		// Query to determine if the user can get the report.
		canGetReportType, err = l.canGetReportType(report)
		if err != nil {
			return false, err
		}
		l.canGetReportTypeByName[report] = canGetReportType
	}
	if !canGetReportType {
		return false, nil
	}

	return true, nil
}

// canGetReport returns true if the caller is allowed to Get a Report. This is an internal method. Consumers
// of the reportRbacHelper should use the canViewReport entry point.
func (l *reportRbacHelper) canGetReport(report string) (bool, error) {
	return true, nil
}

// canGetReportType returns true if the caller is allowed to Get a ReportType. This is an internal method.
// Consumers of the reportRbacHelper should use the canViewReport entry point.
func (l *reportRbacHelper) canGetReportType(reportType string) (bool, error) {
	return true, nil
}
