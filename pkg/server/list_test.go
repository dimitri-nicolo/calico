// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server_test

import (
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/compliance/pkg/server"
	"github.com/tigera/lma/pkg/api"
)

func newArchivedReportData(reportName, reportTypeName string) *api.ArchivedReportData {
	return &api.ArchivedReportData{
		UISummary: `{"foobar":"hello-100-goodbye"}`,
		ReportData: &calicov3.ReportData{
			ReportName:     reportName,
			ReportTypeName: reportTypeName,
			StartTime:      now,
			EndTime:        nowPlusHour,
			EndpointsSummary: calicov3.EndpointsSummary{
				NumTotal: 100,
			},
			GenerationTime: now,
		},
	}
}

func newGlobalReportType(typeName string) v3.GlobalReportType {
	return v3.GlobalReportType{
		ObjectMeta: v1.ObjectMeta{
			Name: typeName,
		},
		Spec: calicov3.ReportTypeSpec{
			UISummaryTemplate: calicov3.ReportTemplate{
				Template: "{\"foobar\":\"hello-{{ .EndpointsSummary.NumTotal }}-goodbye\"}",
			},
			DownloadTemplates: []calicov3.ReportTemplate{
				{
					Name:        "boo.csv",
					Description: "This is a boo file",
					Template:    "xxx-{{ .EndpointsSummary.NumTotal }} BOO!",
				},
				{
					Name:        "bar.csv",
					Description: "This is a bar file",
					Template:    "yyy-{{ .EndpointsSummary.NumTotal }} BAR!",
				},
			},
		},
	}
}

var (
	now         = v1.Time{time.Unix(time.Now().Unix(), 0)}
	nowPlusHour = v1.Time{now.Add(time.Hour)}

	reportTypeGettable    = newGlobalReportType("inventoryGet")
	reportTypeNotGettable = newGlobalReportType("inventoryNoGo")

	reportGetTypeGet     = newArchivedReportData("Get", "inventoryGet")
	reportNoGetTypeNoGet = newArchivedReportData("somethingelse", "inventoryNoGo")
	reportGetTypeNoGet   = newArchivedReportData("Get", "inventoryNoGo")

	forecastFile1 = forecastFile{
		Format:      "boo.csv",
		FileContent: "xxx-100 BOO!",
	}

	forecastFile2 = forecastFile{
		Format:      "bar.csv",
		FileContent: "yyy-100 BAR!",
	}
)

var _ = Describe("List tests with Gettable Report and ReportType", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses")
		t.summaries = []*api.ArchivedReportData{reportGetTypeGet, reportGetTypeNoGet, reportNoGetTypeNoGet}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable, reportTypeNotGettable},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{
			{
				Id:          reportGetTypeGet.UID(),
				Name:        reportGetTypeGet.ReportName,
				Type:        reportGetTypeGet.ReportTypeName,
				StartTime:   now,
				EndTime:     nowPlusHour,
				UISummary:   map[string]interface{}{"foobar": "hello-100-goodbye"},
				DownloadURL: "/compliance/reports/" + reportGetTypeGet.UID() + "/download",
				DownloadFormats: []server.Format{
					{
						Name:        "boo.csv",
						Description: "This is a boo file",
					},
					{
						Name:        "bar.csv",
						Description: "This is a bar file",
					},
				},
				GenerationTime: now,
			},
			{
				Id:              reportGetTypeNoGet.UID(),
				Name:            reportGetTypeNoGet.ReportName,
				Type:            reportGetTypeNoGet.ReportTypeName,
				StartTime:       now,
				EndTime:         nowPlusHour,
				UISummary:       map[string]interface{}{"foobar": "hello-100-goodbye"},
				DownloadURL:     "",
				DownloadFormats: nil,
				GenerationTime:  now,
			},
		})

		By("Stopping the server")
		t.stop()
	})
})

var _ = Describe("List tests with Gettable Report and ReportType but no List", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting tester to refuse List access")
		t.listRBACControl = ""

		By("Setting responses")
		t.summaries = []*api.ArchivedReportData{reportGetTypeGet, reportGetTypeNoGet, reportNoGetTypeNoGet}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable, reportTypeNotGettable},
		}

		By("Running a list query")
		t.list(http.StatusUnauthorized, nil)

		By("Stopping the server")
		t.stop()
	})
})

var _ = Describe("List tests with Not Gettable ReportType", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses to")
		t.summaries = []*api.ArchivedReportData{reportGetTypeGet, reportNoGetTypeNoGet, reportGetTypeNoGet}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeNotGettable, reportTypeGettable},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{
			{
				Id:          reportGetTypeGet.UID(),
				Name:        reportGetTypeGet.ReportName,
				Type:        reportGetTypeGet.ReportTypeName,
				StartTime:   now,
				EndTime:     nowPlusHour,
				UISummary:   map[string]interface{}{"foobar": "hello-100-goodbye"},
				DownloadURL: "/compliance/reports/" + reportGetTypeGet.UID() + "/download",
				DownloadFormats: []server.Format{
					{
						Name:        "boo.csv",
						Description: "This is a boo file",
					},
					{
						Name:        "bar.csv",
						Description: "This is a bar file",
					},
				},
				GenerationTime: now,
			},
			{
				Id:              reportGetTypeNoGet.UID(),
				Name:            reportGetTypeNoGet.ReportName,
				Type:            reportGetTypeNoGet.ReportTypeName,
				StartTime:       now,
				EndTime:         nowPlusHour,
				UISummary:       map[string]interface{}{"foobar": "hello-100-goodbye"},
				DownloadURL:     "",
				DownloadFormats: nil,
				GenerationTime:  now,
			},
		})

		By("Stopping the server")
		t.stop()
	})
})

var _ = Describe("List tests with none available", func() {
	It("Can handle list queries with no items", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses to contain no items")
		t.summaries = []*api.ArchivedReportData{reportNoGetTypeNoGet, reportNoGetTypeNoGet}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{})

		By("Stopping the server")
		t.stop()
	})
})

var _ = Describe("List query parameters", func() {
	It("Can parse parameters correctly", func() {
		By("parsing no query params in the URL")
		maxItems := 100
		v, _ := url.ParseQuery("")
		qp, err := server.GetListReportsQueryParams(v)
		Expect(err).NotTo(HaveOccurred())
		Expect(qp).To(Equal(&api.ReportQueryParams{
			Reports:  nil,
			FromTime: "",
			ToTime:   "",
			Page:     0,
			MaxItems: &maxItems,
			SortBy:   []api.ReportSortBy{{"startTime", false}, {"reportTypeName", true}, {"reportName", true}},
		}))

		By("parsing all query params in the URL")
		v, _ = url.ParseQuery("reportTypeName=type1&reportTypeName=type2&reportName=name1&reportName=name2&" +
			"page=2&fromTime=now-2d&toTime=now-4d&maxItems=4&sortBy=endTime&sortBy=reportName/ascending&" +
			"sortBy=reportTypeName/descending")
		maxItems = 4
		qp, err = server.GetListReportsQueryParams(v)
		Expect(err).NotTo(HaveOccurred())
		Expect(qp).To(Equal(&api.ReportQueryParams{
			Reports:  []api.ReportTypeAndName{{"type1", ""}, {"type2", ""}, {"", "name1"}, {"", "name2"}},
			FromTime: "now-2d",
			ToTime:   "now-4d",
			Page:     2,
			MaxItems: &maxItems,
			SortBy:   []api.ReportSortBy{{"endTime", false}, {"reportName", true}, {"reportTypeName", false}, {"startTime", false}},
		}))

		By("parsing maxItems=all with page=0")
		v, _ = url.ParseQuery("maxItems=all&page=0")
		maxItems = 4
		qp, err = server.GetListReportsQueryParams(v)
		Expect(err).NotTo(HaveOccurred())
		Expect(qp).To(Equal(&api.ReportQueryParams{
			Reports:  nil,
			FromTime: "",
			ToTime:   "",
			Page:     0,
			MaxItems: nil,
			SortBy:   []api.ReportSortBy{{"startTime", false}, {"reportTypeName", true}, {"reportName", true}},
		}))
	})

	It("Errors when supplied with invalid query parameters", func() {
		By("parsing an invalid sort name")
		v, _ := url.ParseQuery("sortBy=fred")
		_, err := server.GetListReportsQueryParams(v)
		Expect(err).To(HaveOccurred())

		By("parsing a negative page number")
		v, _ = url.ParseQuery("page=-1")
		_, err = server.GetListReportsQueryParams(v)
		Expect(err).To(HaveOccurred())

		By("parsing an invalid page number")
		v, _ = url.ParseQuery("page=x")
		_, err = server.GetListReportsQueryParams(v)
		Expect(err).To(HaveOccurred())

		By("parsing a negative maxItems")
		v, _ = url.ParseQuery("maxItems=-1")
		_, err = server.GetListReportsQueryParams(v)
		Expect(err).To(HaveOccurred())

		By("parsing an invalid maxItems")
		v, _ = url.ParseQuery("maxItems=x")
		_, err = server.GetListReportsQueryParams(v)
		Expect(err).To(HaveOccurred())

		By("parsing a non-zero page number when maxItems=all")
		v, _ = url.ParseQuery("page=1&maxItems=all")
		_, err = server.GetListReportsQueryParams(v)
		Expect(err).To(HaveOccurred())
	})
})
