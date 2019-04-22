package server

import (
	"net/http"

	esprox "github.com/tigera/es-proxy-image/pkg/middleware"

	"github.com/tigera/compliance/pkg/report"
	authzv1 "k8s.io/api/authorization/v1"
)

type ReportRbacHelper interface {
	CanViewReport(string, string) (bool, error)
	CanListAnyReportsIn([]*report.ArchivedReportData) (bool, error)
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
	canGetReport, ok := l.canGetReportByName[reportName]
	if !ok {
		// Query to determine if the user can get the report.
		canGetReport, err = l.CanGetReport(reportName)
		if err != nil {
			return false, err
		}
		l.canGetReportByName[reportName] = canGetReport
	}
	if !canGetReport {
		return false, nil
	}
	canGetReportType, ok := l.canGetReportTypeByName[reportName]
	if !ok {
		// Query to determine if the user can get the reportType.
		canGetReportType, err = l.CanGetReportType(reportTypeName)
		if err != nil {
			return false, err
		}
		l.canGetReportTypeByName[reportName] = canGetReportType
	}
	if !canGetReportType {
		return false, nil
	}

	return true, nil
}

// CanListAnyReportsIn returns true if the caller can view any of the reports
func (l *reportRbacHelper) CanListAnyReportsIn(reps []*report.ArchivedReportData) (bool, error) {
	for _, r := range reps {
		ok, err := l.CanListReports(r.ReportName)
		if err != nil {
			return false, err
		}
		//done as soon we find one that can be returned
		if ok == true {
			return true, nil
		}
	}
	//if we don't find any then the reports cannot be listed
	return false, nil
}

// CanListReports returns true if the caller is allowed to List Reports.
func (l *reportRbacHelper) CanListReports(reportName string) (bool, error) {
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "list",
		Group:    "projectcalico.org",
		Resource: "globalreports",
		Name:     reportName,
	}
	return l.checkAuthorized(*resAtr)
}

// CanGetReport returns true if the caller is allowed to Get a Report. This is an internal method. Consumers
// of the reportRbacHelper should use the canViewReport entry point.
func (l *reportRbacHelper) CanGetReport(reportName string) (bool, error) {
	resAtr := &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "projectcalico.org",
		Resource: "globalreports",
		Name:     reportName,
	}
	return l.checkAuthorized(*resAtr)
}

// CanGetReportType returns true if the caller is allowed to Get a ReportType. This is an internal method.
// Consumers of the reportRbacHelper should use the canViewReport entry point.
func (l *reportRbacHelper) CanGetReportType(reportTypeName string) (bool, error) {
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
