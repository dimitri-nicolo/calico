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

	"github.com/projectcalico/libcalico-go/lib/resources"
	"github.com/tigera/lma/pkg/api"
	. "github.com/tigera/lma/pkg/elastic"
	"github.com/tigera/lma/pkg/list"
)

type Resetable interface {
	Reset()
}

var _ = Describe("Compliance elasticsearch integration tests", func() {
	var (
		elasticClient Client
		ts            = time.Date(2019, 4, 15, 15, 0, 0, 0, time.UTC)
	)
	BeforeEach(func() {
		err := os.Setenv("ELASTIC_HOST", "localhost")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ELASTIC_SCHEME", "http")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ELASTIC_INDEX_SUFFIX", "test_cluster")
		Expect(err).NotTo(HaveOccurred())
		elasticClient = MustGetElasticClient()
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
			fmt.Sprintf("tigera_secure_ee_snapshots.test_cluster.%s", dateIndex),
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
		By("storing a report")
		rep := &api.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportName: "report-foo",
				EndTime:    metav1.Time{ts.Add(time.Minute)},
			},
		}
		reportTime := time.Now()
		Expect(elasticClient.StoreArchivedReport(rep, reportTime)).ToNot(HaveOccurred())

		By("having the appropriate report indices")
		dateIndex := reportTime.Format(IndexTimeFormat)
		indicesExist, _ := elasticClient.Backend().IndexExists(
			fmt.Sprintf("tigera_secure_ee_compliance_reports.test_cluster.%s", dateIndex),
		).Do(context.Background())
		Expect(indicesExist).To(Equal(true))

		By("retrieving report summaries")
		get := func() ([]*api.ArchivedReportData, error) {
			s, err := elasticClient.RetrieveArchivedReportSummaries(context.Background(), api.ReportQueryParams{})
			if err != nil {
				return nil, err
			}
			return s.Reports, nil
		}
		Eventually(get, "5s", "0.1s").Should(HaveLen(1))

		By("retrieving a specific report")
		retrievedReport, err := elasticClient.RetrieveArchivedReport(rep.UID())
		Expect(err).ToNot(HaveOccurred())
		Expect(retrievedReport.ReportName).To(Equal(rep.ReportName))

		By("retrieving a specific report summary")
		retrievedReportSummary, err := elasticClient.RetrieveArchivedReportSummary(rep.UID())
		Expect(err).ToNot(HaveOccurred())
		Expect(retrievedReportSummary.ReportName).To(Equal(rep.ReportName))

		By("storing a more recent second report")
		rep2 := &api.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportName: "report-foo",
				EndTime:    metav1.Time{ts.Add(2 * time.Minute)},
			},
		}
		Expect(elasticClient.StoreArchivedReport(rep2, reportTime.Add(time.Minute))).ToNot(HaveOccurred())

		By("retrieving last archived report summary")
		get2 := func() (time.Time, error) {
			rep, err := elasticClient.RetrieveLastArchivedReportSummary(rep.ReportName)
			if err != nil {
				return time.Time{}, err
			}
			return rep.StartTime.Time.UTC(), nil
		}
		Eventually(get2, "5s", "0.1s").Should(Equal(rep2.StartTime.Time.UTC()))

		By("storing a more recent report with a different name")
		rep3 := &api.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportName: "report-foo2",
				EndTime:    metav1.Time{ts.Add(3 * time.Minute)},
			},
		}
		Expect(elasticClient.StoreArchivedReport(rep3, reportTime.Add(time.Minute))).ToNot(HaveOccurred())

		By("retrieving report-foo and not returning report-foo2")
		Eventually(get2, "5s", "0.1s").Should(Equal(rep2.StartTime.Time.UTC()))
	})
})
