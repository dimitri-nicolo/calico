package mocks

import (
	"fmt"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/report"
)

type MockReportRetriever struct {
}

func (c *MockReportRetriever) RetrieveArchivedReport(id string) (*report.ArchivedReportData, error) {

	rd := apiv3.ReportData{}
	rd.ReportName = fmt.Sprintf("Report0")
	rd.ReportSpec = apiv3.ReportSpec{ReportType: "inventory"}
	rd.StartTime = metav1.Now()
	rd.EndTime = metav1.Now()

	rd.ReportSpec.Endpoints.Selector = "EP_selector"

	rd.ReportSpec.Endpoints.Namespaces = &apiv3.NamesAndLabelsMatch{Selector: "NS_selector"}
	rd.ReportSpec.Endpoints.ServiceAccounts = &apiv3.NamesAndLabelsMatch{Selector: "SA_selector"}

	r := report.NewArchivedReport(&rd, "UI summary 0")

	return r, nil

}

func (c *MockReportRetriever) RetrieveArchivedReportSummaries() ([]*report.ArchivedReportData, error) {

	rl := make([]*report.ArchivedReportData, 5)

	for i := 0; i < 5; i++ {
		rd := apiv3.ReportData{}
		rd.ReportName = fmt.Sprintf("Report%d", i)
		rd.ReportSpec = apiv3.ReportSpec{ReportType: "inventory"}
		rd.StartTime = metav1.Now()
		rd.EndTime = metav1.Now()
		r := report.NewArchivedReport(&rd, fmt.Sprintf("UI summary %d", i))
		rl[i] = r
	}

	return rl, nil
}
