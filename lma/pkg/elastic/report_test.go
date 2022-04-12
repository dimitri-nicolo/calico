// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.
package elastic_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/lma/pkg/api"
	. "github.com/projectcalico/calico/lma/pkg/elastic"
)

var _ = Describe("Compliance elasticsearch report list tests", func() {
	var (
		elasticClient Client
		ts            = time.Date(2019, 4, 15, 15, 0, 0, 0, time.UTC)
		reportIdx     = 0
		numReports    = 0
	)

	// addReport is a helper function used to add a report, and track how many reports have been added.
	addReport := func(typeName, name string) *api.ArchivedReportData {
		rep := &api.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportTypeName: typeName,
				ReportName:     name,
				StartTime:      metav1.Time{Time: ts.Add(time.Duration(reportIdx) * time.Minute)},
				EndTime:        metav1.Time{Time: ts.Add((time.Duration(reportIdx) * time.Minute) + (2 * time.Minute))},
				GenerationTime: metav1.Time{Time: ts.Add(-time.Duration(reportIdx) * time.Minute)},
			},
		}
		Expect(elasticClient.StoreArchivedReport(rep)).ToNot(HaveOccurred())
		numReports++

		// Increment the report index across this set of tests. This is used to make each report unique which avoids any
		// timing windows that occur as part of the reset processing (where we may try to create a document that is in
		// the process of being deleted.
		reportIdx++
		return rep
	}

	// waitForReports is a helper function used to wait for ES to process all of the report creations.
	waitForReports := func() {
		get := func() error {
			r, err := elasticClient.RetrieveArchivedReportSummaries(context.Background(), api.ReportQueryParams{})
			if err != nil {
				return err
			}
			if int(r.Count) != int(numReports) {
				return fmt.Errorf("Expected %d results, found %d", numReports, int(r.Count))
			}
			return nil
		}
		Eventually(get, "5s", "0.1s").ShouldNot(HaveOccurred())
	}

	// ensureUTC updates the time fields in the ArchivedReportDatas are UTC so that ginkgo/gomega can be used to compare.
	ensureUTC := func(reps []*api.ArchivedReportData) {
		for ii := range reps {
			reps[ii].EndTime.Time = reps[ii].EndTime.UTC()
			reps[ii].StartTime.Time = reps[ii].StartTime.UTC()
			reps[ii].GenerationTime.Time = reps[ii].GenerationTime.UTC()
		}
	}

	BeforeEach(func() {
		err := os.Setenv("ELASTIC_HOST", "localhost")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ELASTIC_SCHEME", "http")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ELASTIC_INDEX_SUFFIX", "test_cluster")
		Expect(err).NotTo(HaveOccurred())
		elasticClient = MustGetElasticClient()
		deleteIndex(MustLoadConfig(), ReportsIndex)
		elasticClient.(Resetable).Reset()
		numReports = 0
	})

	It("should retrieve no reportTypeName/reportName combinations when no reports are added", func() {
		By("retrieving the full set of unique reportTypeName/reportName combinations")
		r, err := elasticClient.RetrieveArchivedReportTypeAndNames(context.Background(), api.ReportQueryParams{})

		By("checking no results were returned")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(0))
	})

	It("should retrieve the correct set of reportTypeName/reportName combinations", func() {
		By("storing a small number of reports with repeats")
		// Add a bunch of reports, with some repeated reportTypeName / reportName combinations.
		first := addReport("type1", "report1") // 1
		_ = addReport("type2", "report1")      // 2
		_ = addReport("type1", "report2")      // 3
		_ = addReport("type3", "report3")      // 4
		_ = addReport("type1", "report2")      // Repeat of 3
		_ = addReport("type3", "report2")      // 5
		last := addReport("type4", "report3")  // 6
		waitForReports()

		By("retrieving the full set of unique reportTypeName/reportName combinations")
		cxt, cancel := context.WithCancel(context.Background())
		r, err := elasticClient.RetrieveArchivedReportTypeAndNames(cxt, api.ReportQueryParams{})

		By("checking we have the correct set of unique combinations")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(6))
		Expect(r).To(ConsistOf(
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report1"},
			api.ReportTypeAndName{ReportTypeName: "type2", ReportName: "report1"},
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report2"},
			api.ReportTypeAndName{ReportTypeName: "type3", ReportName: "report3"},
			api.ReportTypeAndName{ReportTypeName: "type3", ReportName: "report2"},
			api.ReportTypeAndName{ReportTypeName: "type4", ReportName: "report3"},
		))

		By("retrieving the set of unique reportTypeName/reportName combinations with report filter")
		r, err = elasticClient.RetrieveArchivedReportTypeAndNames(cxt, api.ReportQueryParams{
			Reports: []api.ReportTypeAndName{{ReportTypeName: "type1"}, {ReportName: "report2"}, {ReportTypeName: "type3", ReportName: "report3"}},
		})

		By("checking we have the correct set of unique combinations")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(4))
		Expect(r).To(ConsistOf(
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report1"},
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report2"},
			api.ReportTypeAndName{ReportTypeName: "type3", ReportName: "report3"},
			api.ReportTypeAndName{ReportTypeName: "type3", ReportName: "report2"},
		))

		By("retrieving the set of unique reportTypeName/reportName combinations with upper time filter")
		r, err = elasticClient.RetrieveArchivedReportTypeAndNames(cxt, api.ReportQueryParams{
			ToTime: first.StartTime.Format(time.RFC3339), // Query up to the first report
		})

		By("checking we have the correct set of unique combinations")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(1))
		Expect(r).To(ConsistOf(
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report1"},
		))

		By("retrieving the set of unique reportTypeName/reportName combinations with lower time filter")
		r, err = elasticClient.RetrieveArchivedReportTypeAndNames(cxt, api.ReportQueryParams{
			FromTime: last.EndTime.Format(time.RFC3339), // Query from the last report
		})

		By("checking we have the correct set of unique combinations")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(1))
		Expect(r).To(ConsistOf(
			api.ReportTypeAndName{ReportTypeName: "type4", ReportName: "report3"},
		))

		By("retrieving the set of unique reportTypeName/reportName combinations with time range filter")
		r, err = elasticClient.RetrieveArchivedReportTypeAndNames(cxt, api.ReportQueryParams{
			FromTime: first.StartTime.Format(time.RFC3339), // Query from the first report
			ToTime:   last.EndTime.Format(time.RFC3339),    // to the last report.
		})

		By("checking we have the correct set of unique combinations")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(6))
		Expect(r).To(ConsistOf(
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report1"},
			api.ReportTypeAndName{ReportTypeName: "type2", ReportName: "report1"},
			api.ReportTypeAndName{ReportTypeName: "type1", ReportName: "report2"},
			api.ReportTypeAndName{ReportTypeName: "type3", ReportName: "report3"},
			api.ReportTypeAndName{ReportTypeName: "type3", ReportName: "report2"},
			api.ReportTypeAndName{ReportTypeName: "type4", ReportName: "report3"},
		))

		By("checking we handle cancelled context")
		cancel()
		_, err = elasticClient.RetrieveArchivedReportTypeAndNames(cxt, api.ReportQueryParams{})
		Expect(err).To(HaveOccurred())
	})

	It("should handle more than DefaultPageSize combinations of reportTypeName/reportName", func() {
		By("storing >DefaultPageSize unique reportTypeName/reportName combination with repeats")
		var unique []api.ReportTypeAndName
		// Add DefaultPageSize * 2 unique combinations (and add 2 reports of each)
		for ii := 0; ii < DefaultPageSize*2; ii++ {
			tn := fmt.Sprintf("type%d", ii)
			rn := fmt.Sprintf("report%d", ii)
			_ = addReport(tn, rn)
			_ = addReport(tn, rn)
			unique = append(unique, api.ReportTypeAndName{ReportTypeName: tn, ReportName: rn})
		}
		waitForReports()

		By("retrieving the full set of unique reportTypeName/reportName combinations")
		r, err := elasticClient.RetrieveArchivedReportTypeAndNames(context.Background(), api.ReportQueryParams{})
		By("checking we have the correct set of unique combinations")
		Expect(err).NotTo(HaveOccurred())
		Expect(r).To(HaveLen(DefaultPageSize * 2))
		Expect(r).To(ConsistOf(unique))
	})

	It("should retrieve no report summaries when no reports are added", func() {
		By("retrieving the full set of report summaries")
		r, err := elasticClient.RetrieveArchivedReportSummaries(context.Background(), api.ReportQueryParams{})

		By("checking no results were returned")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(0))
		Expect(r.Reports).To(HaveLen(0))
	})

	It("should retrieve the correct set of reports", func() {
		By("storing a small number of reports")
		// Add a bunch of reports, with some repeated reportTypeName / reportName combinations.
		r1 := addReport("type1", "report1")
		r2 := addReport("type2", "report1")
		r3 := addReport("type1", "report2")
		r4 := addReport("type3", "report3")
		r5 := addReport("type3", "report2")
		r6 := addReport("type4", "report3")
		waitForReports()

		By("retrieving the full set of report summaries (sort by startTime)")
		cxt, cancel := context.WithCancel(context.Background())
		r, err := elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy: []api.ReportSortBy{{Field: "startTime"}},
		})

		By("checking we have the correct set of reports in the correct order")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r6, r5, r4, r3, r2, r1}))

		By("retrieving the full set of report summaries (sort by ascending startTime)")
		r, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy: []api.ReportSortBy{{Field: "startTime", Ascending: true}},
		})

		By("checking we have the correct set of reports in the correct order")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r1, r2, r3, r4, r5, r6}))

		By("retrieving the full set of report summaries (sort by ascending endTime)")
		r, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy: []api.ReportSortBy{{Field: "endTime", Ascending: true}},
		})

		By("checking we have the correct set of reports in the correct order")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r1, r2, r3, r4, r5, r6}))

		By("retrieving the full set of report summaries (sort by generationTime)")
		r, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy: []api.ReportSortBy{{Field: "generationTime"}}, // generationTime is in opposite order to start/end times
		})

		By("checking we have the correct set of reports in the correct order")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r1, r2, r3, r4, r5, r6}))

		By("retrieving the full set of report summaries (sort by descending reportTypeName and descending startTime)")
		r, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy: []api.ReportSortBy{{Field: "reportTypeName"}, {Field: "startTime"}},
		})

		By("checking we have the correct set of reports in the correct order")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r6, r5, r4, r2, r3, r1}))

		By("retrieving the full set of report summaries (sort by ascending reportName and descending startTime), maxItems=4")
		maxItems := 4
		r, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy:   []api.ReportSortBy{{Field: "reportName", Ascending: true}, {Field: "startTime"}},
			MaxItems: &maxItems,
		})

		By("checking we can receive the results for page 0")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r2, r1, r5, r3}))

		By("checking we can query page 1")
		r, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{
			SortBy:   []api.ReportSortBy{{Field: "reportName", Ascending: true}, {Field: "startTime"}},
			MaxItems: &maxItems,
			Page:     1,
		})

		By("checking we can receive the results for page 0")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(6))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r6, r4}))

		By("checking we handle cancelled context")
		cancel()
		_, err = elasticClient.RetrieveArchivedReportSummaries(cxt, api.ReportQueryParams{})
		Expect(err).To(HaveOccurred())
	})

	It("should handle the default sort order when start times are the same, sorting by time, type, name", func() {
		By("storing a small number of reports all with the same start time")
		// Add a bunch of reports all with the same start time
		r1 := addReport("type1", "report1")
		reportIdx-- // Decrementing the report index means the next report will have the same start time.
		r2 := addReport("type2", "report1")
		reportIdx--
		r3 := addReport("type1", "report2")
		reportIdx--
		r4 := addReport("type3", "report3")
		reportIdx--
		r5 := addReport("type3", "report2")
		reportIdx--
		r6 := addReport("type4", "report3")
		r7 := addReport("type1", "report2") // Later start time from r1 (should appear first)
		waitForReports()

		By("retrieving the full set of report summaries (sort by startTime, reportTypeName, reportName)")
		r, err := elasticClient.RetrieveArchivedReportSummaries(context.Background(), api.ReportQueryParams{
			SortBy: []api.ReportSortBy{
				{Field: "startTime"}, {Field: "reportTypeName", Ascending: true}, {Field: "reportName", Ascending: true},
			},
		})

		By("checking we have the correct set of reports in the correct order")
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Count).To(Equal(7))
		ensureUTC(r.Reports) // Normalize the times to make them comparable.
		Expect(r.Reports).To(Equal([]*api.ArchivedReportData{r7, r1, r3, r2, r5, r4, r6}))
	})

	It("should create an index with the correct index settings", func() {
		cfg := MustLoadConfig()
		cfg.ElasticReplicas = 2
		cfg.ElasticShards = 7
		t := ts.Add(72 * time.Hour)
		rep := &api.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportTypeName: "testindexsettings",
				ReportName:     "testindexsettings",
				StartTime:      metav1.Time{Time: t},
				EndTime:        metav1.Time{Time: t.Add(2 * time.Minute)},
				GenerationTime: metav1.Time{Time: t.Add(-time.Minute)},
			},
		}

		elasticClient, err := NewFromConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(elasticClient.StoreArchivedReport(rep)).ToNot(HaveOccurred())

		index := elasticClient.ClusterAlias(ReportsIndex)
		testIndexSettings(cfg, index, map[string]interface{}{
			"number_of_replicas": "2",
			"number_of_shards":   "7",
			"lifecycle": map[string]interface{}{
				"name":           ReportsIndex + "_policy",
				"rollover_alias": index,
			},
		})
	})

	It("should create an index template and update it on change", func() {
		cfg := MustLoadConfig()
		cfg.ElasticReplicas = 2
		cfg.ElasticShards = 7
		t := ts.Add(72 * time.Hour)
		rep := &api.ArchivedReportData{
			ReportData: &apiv3.ReportData{
				ReportTypeName: "testindexsettings",
				ReportName:     "testindexsettings",
				StartTime:      metav1.Time{Time: t},
				EndTime:        metav1.Time{Time: t.Add(2 * time.Minute)},
				GenerationTime: metav1.Time{Time: t.Add(-time.Minute)},
			},
		}

		elasticClient, err := NewFromConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(elasticClient.StoreArchivedReport(rep)).ToNot(HaveOccurred())

		index := elasticClient.ClusterAlias(ReportsIndex)
		templateName := elasticClient.IndexTemplateName(ReportsIndex)

		testIndexTemplateSettings(cfg, templateName, map[string]interface{}{
			"index": map[string]interface{}{
				"number_of_replicas": "2",
				"number_of_shards":   "7",
				"lifecycle": map[string]interface{}{
					"name":           ReportsIndex + "_policy",
					"rollover_alias": index,
				},
			},
		})

		// Change the settings value and check template is updated with new setting values
		cfg.ElasticShards = 3
		elasticClient, err = NewFromConfig(cfg)
		Expect(err).ToNot(HaveOccurred())
		Expect(elasticClient.StoreArchivedReport(rep)).ToNot(HaveOccurred())

		testIndexTemplateSettings(cfg, templateName, map[string]interface{}{
			"index": map[string]interface{}{
				"number_of_replicas": "2",
				"number_of_shards":   "3",
				"lifecycle": map[string]interface{}{
					"name":           ReportsIndex + "_policy",
					"rollover_alias": index,
				},
			},
		})
	})
})
