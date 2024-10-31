//  Copyright (c) 2016,2018 Tigera, Inc. All rights reserved.

package main_windows_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestCalicoCni(t *testing.T) {
	RegisterFailHandler(Fail)
	reportPath := os.Getenv("REPORT")
	if reportPath == "" {
		// Default the report path if not specified.
		reportPath = "../report/windows_suite.xml"
	}
	junitReporter := reporters.NewJUnitReporter(reportPath)
	RunSpecsWithDefaultAndCustomReporters(t, "CNI suite (Windows)", []Reporter{junitReporter})
}
