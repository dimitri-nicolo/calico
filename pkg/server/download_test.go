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

		By("Setting responses to")
		t.report = reportListAndGet
		t.reportTypeList = &v3.GlobalReportTypeList{
			Items: []v3.GlobalReportType{reportTypeGettable},
		}

		By("Running a download query")
		t.downloadSingle(reportListAndGet.UID(), http.StatusOK, forecastFile1)

		forecasts := []forecastFile{forecastFile1, forecastFile2}
		By("Running a multi download query")
		t.downloadMulti(reportListAndGet.UID(), http.StatusOK, forecasts)

		By("Stopping the server")
		t.stop()
	})
})
