package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	authzv1 "k8s.io/api/authorization/v1"

	esprox "github.com/tigera/es-proxy/pkg/middleware"
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
		Request:                req,
		k8sAuth:                f.auth,
	}
}

func NewStandardRbacHelperFactory(auth K8sAuthInterface) RbacHelperFactory {
	return &standardRbacHelperFactory{auth: auth}
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

// checkAuthorized returns true if the request is allowed for the resources decribed in provieded attributes
func (l *reportRbacHelper) checkAuthorized(atr authzv1.ResourceAttributes) (bool, error) {

	ctx := esprox.NewContextWithReviewResource(l.Request.Context(), &atr)
	req := l.Request.WithContext(ctx)

	stat, err := l.k8sAuth.Authorize(req)
	switch stat {
	case 0:
		log.WithField("stat", stat).Info("Request authorized")
		return true, nil
	case http.StatusForbidden:
		log.WithField("stat", stat).WithError(err).Info("Forbidden - not authorized")
		return false, nil
	}
	log.WithField("stat", stat).WithError(err).Info("Error authorizing")
	return false, err
}
