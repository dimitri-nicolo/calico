// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	calicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/compliance/pkg/report"
	"github.com/tigera/compliance/pkg/server"
)

func newArchivedReportData(reportName, reportTypeName string) *report.ArchivedReportData {
	return &report.ArchivedReportData{
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

	reportListAndGet  = newArchivedReportData("ListGet", "inventoryGet")
	reportNotListable = newArchivedReportData("somethingelse", "inventoryGet")
	reportListNoGet   = newArchivedReportData("List", "inventoryGet")

	reportListAndGetNotType  = newArchivedReportData("ListGet", "inventoryNoGo")
	reportNotListableNotType = newArchivedReportData("somethingelse", "inventoryNoGo")
	reportListNoGetNotType   = newArchivedReportData("List", "inventoryNoGo")

	forecastFile1 = forecastFile{
		Format:      "boo.csv",
		FileContent: "xxx-100 BOO!",
	}

	forecastFile2 = forecastFile{
		Format:      "bar.csv",
		FileContent: "yyy-100 BAR!",
	}
)

var _ = Describe("List tests with Gettable ReportType", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses to")
		t.summaries = []*report.ArchivedReportData{reportListAndGet, reportListNoGet, reportNotListable}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{
			{
				Id:        reportListAndGet.UID(),
				Name:      reportListAndGet.ReportName,
				Type:      reportListAndGet.ReportTypeName,
				StartTime: now,
				EndTime:   nowPlusHour,
				UISummary: map[string]interface{}{
					"foobar": "hello-100-goodbye",
				},
				DownloadURL: "/compliance/reports/" + reportListAndGet.UID() + "/download",
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
				Id:        reportListNoGet.UID(),
				Name:      reportListNoGet.ReportName,
				Type:      reportListNoGet.ReportTypeName,
				StartTime: now,
				EndTime:   nowPlusHour,
				UISummary: map[string]interface{}{
					"foobar": "hello-100-goodbye",
				},
				DownloadURL:     "",
				DownloadFormats: nil,
				GenerationTime:  now,
			},
		})

		By("Stopping the server")
		t.stop()
	})
})

var _ = Describe("List tests with Not Gettable ReportType", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses to")
		t.summaries = []*report.ArchivedReportData{reportListAndGetNotType, reportNotListableNotType, reportListNoGetNotType}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeNotGettable},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{
			{
				Id:        reportListAndGetNotType.UID(),
				Name:      reportListAndGetNotType.ReportName,
				Type:      reportListAndGetNotType.ReportTypeName,
				StartTime: now,
				EndTime:   nowPlusHour,
				UISummary: map[string]interface{}{
					"foobar": "hello-100-goodbye",
				},
				DownloadURL:     "",
				DownloadFormats: nil,
				GenerationTime:  now,
			},
			{
				Id:        reportListNoGetNotType.UID(),
				Name:      reportListNoGetNotType.ReportName,
				Type:      reportListNoGetNotType.ReportTypeName,
				StartTime: now,
				EndTime:   nowPlusHour,
				UISummary: map[string]interface{}{
					"foobar": "hello-100-goodbye",
				},
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
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses to")
		t.summaries = []*report.ArchivedReportData{reportNotListable, reportNotListableNotType}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{})

		By("Stopping the server")
		t.stop()
	})
})
