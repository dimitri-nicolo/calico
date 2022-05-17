package mocks

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/lma/pkg/api"
)

type MockReportRetriever struct {
}

func (c *MockReportRetriever) RetrieveArchivedReport(id string) (*api.ArchivedReportData, error) {

	rd := apiv3.ReportData{}
	rd.ReportName = "Report0"
	rd.ReportSpec = apiv3.ReportSpec{ReportType: "inventory"}
	rd.StartTime = metav1.Now()
	rd.EndTime = metav1.Now()

	rd.ReportSpec.Endpoints.Selector = "EP_selector"

	rd.ReportSpec.Endpoints.Namespaces = &apiv3.NamesAndLabelsMatch{Selector: "NS_selector"}
	rd.ReportSpec.Endpoints.ServiceAccounts = &apiv3.NamesAndLabelsMatch{Selector: "SA_selector"}

	r := api.NewArchivedReport(&rd, "UI summary 0")

	return r, nil

}

func (c *MockReportRetriever) RetrieveArchivedReportSummaries() ([]*api.ArchivedReportData, error) {

	rl := make([]*api.ArchivedReportData, 5)

	for i := 0; i < 5; i++ {
		rd := apiv3.ReportData{}
		rd.ReportName = fmt.Sprintf("Report%d", i)
		rd.ReportSpec = apiv3.ReportSpec{ReportType: "inventory"}
		rd.StartTime = metav1.Now()
		rd.EndTime = metav1.Now()
		r := api.NewArchivedReport(&rd, fmt.Sprintf("UI summary %d", i))
		rl[i] = r
	}

	return rl, nil
}
