package server

import (
	"net/http"

	authzv1 "k8s.io/api/authorization/v1"

	esprox "github.com/tigera/es-proxy-image/pkg/middleware"
)

type ReportRbacHelper interface {
	CanViewReport(string, string) (bool, error)
	CanListReports(string) (bool, error)
	CanGetReport(string) (bool, error)
	CanGetReportType(string) (bool, error)
}

// reportRbacHelper implements helper functionality to determine whether the API user is able to view
// and list reports. The helper is used for a single API request and will cache information that may be
// required multiple times.
type reportRbacHelper struct {
	canGetReportTypeByName map[string]bool
	canGetReportByName     map[string]bool
	canListReportByName    map[string]bool
	Request                *http.Request
	k8sAuth                K8sAuthInterface
}

type K8sAuthInterface interface {
	Authorize(*http.Request) (int, error)
}

type RbacHelperFactory interface {
	NewReportRbacHelper(*http.Request) ReportRbacHelper
}

type standardRbacHelperFactory struct {
	auth K8sAuthInterface
}

// newReportRbacHelper returns a new initialized reportRbacHelper.
func (f *standardRbacHelperFactory) NewReportRbacHelper(req *http.Request) ReportRbacHelper {

	return &reportRbacHelper{
		canGetReportTypeByName: make(map[string]bool),
		canGetReportByName:     make(map[string]bool),
		canListReportByName:    make(map[string]bool),
		Request:                req,
		k8sAuth:                f.auth,
	}
}

func NewStandardRbacHelperFactory(auth K8sAuthInterface) RbacHelperFactory {
	return &standardRbacHelperFactory{auth: auth}
}

// CanViewReport returns true if the caller is allowed to view a specific Report and ReportType.
func (l *reportRbacHelper) CanViewReport(reportTypeName, reportName string) (bool, error) {
	var err error

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

// CanListReports returns true if the caller is allowed to List Reports.
func (l *reportRbacHelper) CanListReports(reportName string) (bool, error) {
	var err = error(nil)
	canDo, ok := l.canListReportByName[reportName]
	if !ok {
		// Query to determine if the user can list the report.
		canDo, err = l.canListReports(reportName)
		if err == nil {
			l.canListReportByName[reportName] = canDo
		}
	}
	return canDo, err
}

// canListReports returns true if the caller is allowed to List Reports. This is an internal method.
func (l *reportRbacHelper) canListReports(reportName string) (bool, error) {
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "list",
		Group:    "projectcalico.org",
		Resource: "globalreports",
		Name:     reportName,
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

// checkAuthorized returns true if the request is allowed for the resources decribed in provieded attributes
func (l *reportRbacHelper) checkAuthorized(atr authzv1.ResourceAttributes) (bool, error) {

	ctx := esprox.NewContextWithReviewResource(l.Request.Context(), &atr)
	req := l.Request.WithContext(ctx)

	stat, err := l.k8sAuth.Authorize(req)
	if err != nil {
		return false, err
	}

	return (stat == 0), nil
}
