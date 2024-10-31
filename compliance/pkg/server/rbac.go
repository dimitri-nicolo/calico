package server

import (
	"fmt"
	"net/http"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
)

type ReportRbacHelper interface {
	CanViewReportSummary(reportName string) (bool, error)
	CanViewReport(reportTypeName string, reportName string) (bool, error)
	CanListReports() (bool, error)
	CanGetReport(reportName string) (bool, error)
	CanGetReportType(reportTypeName string) (bool, error)
}

// reportRbacHelper implements helper functionality to determine whether the API user is able to view
// and list reports. The helper is used for a single API request and will cache information that may be
// required multiple times.
type reportRbacHelper struct {
	canGetReportTypeByName map[string]bool
	canGetReportByName     map[string]bool
	request                *http.Request
	authorizer             lmaauth.RBACAuthorizer
}

// newReportRbacHelper returns a new initialized reportRbacHelper.
func NewReportRbacHelper(authorizer lmaauth.RBACAuthorizer, req *http.Request) ReportRbacHelper {
	return &reportRbacHelper{
		canGetReportTypeByName: make(map[string]bool),
		canGetReportByName:     make(map[string]bool),
		request:                req,
		authorizer:             authorizer,
	}
}

// CanViewReport returns true if the caller is allowed to view/download a specific report/report-type.
func (l *reportRbacHelper) CanViewReport(reportTypeName, reportName string) (bool, error) {
	var err error

	// To view a report, the user must be able to get both the report and report type.
	canGetReport, err := l.CanGetReport(reportName)
	if err != nil {
		return false, err
	}
	if !canGetReport {
		return false, nil
	}

	canGetReportType, err := l.CanGetReportType(reportTypeName)
	if err != nil {
		return false, err
	}
	if !canGetReportType {
		return false, nil
	}

	return true, nil
}

// CanViewReportSummary returns true if the caller is allowed to view the report summary for a specific
// report.
func (l *reportRbacHelper) CanViewReportSummary(reportName string) (bool, error) {
	var err error

	// A user can view a report summary if they have get access to the report.
	canGetReport, err := l.CanGetReport(reportName)
	if err != nil {
		return false, err
	}
	if !canGetReport {
		return false, nil
	}

	return true, nil
}

// CanListReports returns true if the caller is allowed to List Reports.
func (l *reportRbacHelper) CanListReports() (bool, error) {
	return l.canListReports()
}

// canListReports returns true if the caller is allowed to List Reports. This is an internal method.
func (l *reportRbacHelper) canListReports() (bool, error) {
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "list",
		Group:    "projectcalico.org",
		Resource: "globalreports",
	}
	return l.checkAuthorized(*resAtr)
}

// CanGetReport returns true if the caller is allowed to Get a Report.
func (l *reportRbacHelper) CanGetReport(reportName string) (bool, error) {
	var err = error(nil)
	canDo, ok := l.canGetReportByName[reportName]
	if !ok {
		// Query to determine if the user can get the report.
		canDo, err = l.canGetReport(reportName)
		if err == nil {
			l.canGetReportByName[reportName] = canDo
		}
	}
	return canDo, err
}

// canGetReport returns true if the caller is allowed to Get a Report. This is an internal method.
func (l *reportRbacHelper) canGetReport(reportName string) (bool, error) {
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "projectcalico.org",
		Resource: "globalreports",
		Name:     reportName,
	}
	return l.checkAuthorized(*resAtr)
}

// CanGetReportType returns true if the caller is allowed to Get a ReportType.
func (l *reportRbacHelper) CanGetReportType(reportTypeName string) (bool, error) {
	var err = error(nil)
	canDo, ok := l.canGetReportTypeByName[reportTypeName]
	if !ok {
		// Query to determine if the user can get the report type.
		canDo, err = l.canGetReportType(reportTypeName)
		if err == nil {
			l.canGetReportTypeByName[reportTypeName] = canDo
		}
	}
	return canDo, err
}

// canGetReportType returns true if the caller is allowed to Get a ReportType. This is an internal method.
func (l *reportRbacHelper) canGetReportType(reportTypeName string) (bool, error) {
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "projectcalico.org",
		Resource: "globalreporttypes",
		Name:     reportTypeName,
	}
	return l.checkAuthorized(*resAtr)
}

// checkAuthorized returns true if the request is allowed for the resources described in provided attributes
func (l *reportRbacHelper) checkAuthorized(atr authzv1.ResourceAttributes) (bool, error) {
	usr, ok := request.UserFrom(l.request.Context())
	if !ok {
		return false, fmt.Errorf("no user found in request context")
	}

	authorize, err := l.authorizer.Authorize(usr, &atr, nil)
	if err != nil {
		return false, err
	}

	return authorize, nil
}
