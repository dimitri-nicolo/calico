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

var (
	now         = v1.Time{time.Unix(time.Now().Unix(), 0)}
	nowPlusHour = v1.Time{now.Add(time.Hour)}

	summary1 = &report.ArchivedReportData{
		ReportData: &calicov3.ReportData{
			ReportName:     "report1",
			ReportTypeName: "inventory",
			StartTime:      now,
			EndTime:        nowPlusHour,
			EndpointsSummary: calicov3.EndpointsSummary{
				NumTotal: 100,
			},
		},
	}

	reportType1 = v3.GlobalReportType{
		ObjectMeta: v1.ObjectMeta{
			Name: "inventory",
		},
		Spec: calicov3.ReportTypeSpec{
			UISummaryTemplate: calicov3.ReportTemplate{
				Template: "hello-{{ .EndpointsSummary.NumTotal }}-goodbye",
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

	forecastFile1 = forecastFile{
		Format:      "boo.csv",
		FileContent: "xxx-100 BOO!",
	}

	forecastFile2 = forecastFile{
		Format:      "bar.csv",
		FileContent: "yyy-100 BAR!",
	}
)

var _ = Describe("List tests", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting responses to")
		t.summaries = []*report.ArchivedReportData{summary1}
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportType1},
		}

		By("Running a list query")
		t.list(http.StatusOK, []server.Report{
			{
				ReportId:    summary1.UID(),
				ReportType:  summary1.ReportTypeName,
				StartTime:   now,
				EndTime:     nowPlusHour,
				UISummary:   "hello-100-goodbye",
				DownloadURL: "/compliance/reports/" + summary1.UID() + "/download",
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
			},
		})

		By("Stopping the server")
		t.stop()
	})
})
