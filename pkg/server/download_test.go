// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package server_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
)

var _ = Describe("Download tests", func() {
	It("", func() {
		By("Starting a test server")
		t := startTester()

		By("Setting available types to")
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable, reportTypeNotGettable},
		}

		By("Setting reports and forecasts with allowed types and reports ")
		t.report = reportListAndGet
		forecasts := []forecastFile{forecastFile1, forecastFile2}

		By("Running a download query that should succeed")
		t.downloadSingle(reportListAndGet.UID(), http.StatusOK, forecastFile1)
		By("Running a multi download query that should succeed")
		t.downloadMulti(reportListAndGet.UID(), http.StatusOK, forecasts)

		By("Setting reports to reports with a not allowed report type")
		t.report = reportListAndGetNotType
		forecasts = []forecastFile{forecastFile1, forecastFile2}

		By("Running a download query that should be denied because of report type")
		t.downloadSingle(reportListAndGetNotType.UID(), http.StatusForbidden, forecastFile1)
		By("Running a multi download query that should be denied because of report type")
		t.downloadMulti(reportListAndGetNotType.UID(), http.StatusForbidden, forecasts)

		By("Setting reports to reports with a not allowed report")
		t.report = reportListNoGet
		forecasts = []forecastFile{forecastFile1, forecastFile2}

		By("Running a download query that should be denied because of report name")
		t.downloadSingle(reportListNoGet.UID(), http.StatusForbidden, forecastFile1)
		By("Running a multi download query that should be denied because of report name")
		t.downloadMulti(reportListNoGet.UID(), http.StatusForbidden, forecasts)

		By("Stopping the server")
		t.stop()
	})
})
