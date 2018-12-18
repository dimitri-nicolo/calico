//  Copyright (c) 2016,2018 Tigera, Inc. All rights reserved.

package main_windows_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func TestCalicoCni(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../report/cni_windows_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "CalicoCni windows Suite", []Reporter{junitReporter})
}
