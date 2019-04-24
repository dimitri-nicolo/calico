package elastic_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/config"
	. "github.com/tigera/compliance/pkg/elastic"
	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/report"
	"github.com/tigera/compliance/pkg/resources"
)

type Resetable interface {
	Reset()
}

var _ = Describe("Elastic", func() {
	var (
		elasticClient Client
		ts            = time.Date(2019, 4, 15, 15, 0, 0, 0, time.UTC)
	)
	BeforeEach(func() {
		os.Setenv("ELASTIC_HOST", "localhost")
		cfg := config.MustLoadConfig()
		elasticClient = MustGetElasticClient(cfg)
		elasticClient.(Resetable).Reset()
	})

	It("should store and retrieve lists properly", func() {
		By("storing a network policy list")
		npResList := &list.TimestampedResourceList{
			ResourceList:              apiv3.NewNetworkPolicyList(),
			RequestStartedTimestamp:   metav1.Time{ts.Add(time.Minute)},
			RequestCompletedTimestamp: metav1.Time{ts.Add(time.Minute)},
		}
		npResList.ResourceList.GetObjectKind().SetGroupVersionKind((&resources.TypeCalicoNetworkPolicies).GroupVersionKind())

		Expect(elasticClient.StoreList(resources.TypeCalicoNetworkPolicies, npResList)).ToNot(HaveOccurred())

		By("storing a second network policy list one hour in the future")
		npResList.RequestStartedTimestamp = metav1.Time{ts.Add(2 * time.Minute)}
		npResList.RequestCompletedTimestamp = metav1.Time{ts.Add(2 * time.Minute)}
		Expect(elasticClient.StoreList(resources.TypeCalicoNetworkPolicies, npResList)).ToNot(HaveOccurred())

		By("having the appropriate snapshot indices")
		dateIndex := npResList.RequestCompletedTimestamp.Format(IndexTimeFormat)
		indicesExist, _ := elasticClient.Backend().IndexExists(
			fmt.Sprintf("tigera_secure_ee_snapshots.cluster.%s", dateIndex),
		).Do(context.Background())
		Expect(indicesExist).To(Equal(true))

		By("retrieving the network policy list, earliest first")
		start := ts.Add(-12 * time.Hour)
		end := ts.Add(12 * time.Hour)

		get := func() (*list.TimestampedResourceList, error) {
			return elasticClient.RetrieveList(resources.TypeCalicoNetworkPolicies, &start, &end, true)
		}
		Eventually(get, "5s", "0.1s").ShouldNot(BeNil())
	})

	It("should store and retrieve reports properly", func() {
		rep := &report.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportName:        "report-foo",
				ReportTypeName:    "report-type-bar",
				StartTime:         metav1.Time{ts},
				EndTime:           metav1.Time{ts.Add(time.Minute)},
				EndpointsSummary:  apiv3.EndpointsSummary{},
				NamespacesSummary: apiv3.EndpointsSummary{},
				ServicesSummary:   apiv3.EndpointsSummary{},
			},
			UISummary: "random-summary",
		}
		reportTime := time.Now()
		By("storing a report")
		Expect(elasticClient.StoreArchivedReport(rep, reportTime)).ToNot(HaveOccurred())
		time.Sleep(time.Second)

		By("having the appropriate report indices")
		dateIndex := reportTime.Format(IndexTimeFormat)
		indicesExist, _ := elasticClient.Backend().IndexExists(
			fmt.Sprintf("tigera_secure_ee_compliance_reports.cluster.%s", dateIndex),
		).Do(context.Background())
		Expect(indicesExist).To(Equal(true))

		By("retrieving report summaries")
		get := func() ([]*report.ArchivedReportData, error) {
			return elasticClient.RetrieveArchivedReportSummaries()
		}
		Eventually(get, "5s", "0.1s").Should(HaveLen(1))

		By("retrieving a specific report")
		retrievedReport, err := elasticClient.RetrieveArchivedReport(rep.UID())
		Expect(err).ToNot(HaveOccurred())
		Expect(retrievedReport.ReportName).To(Equal(rep.ReportName))
	})
})
