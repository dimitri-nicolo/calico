//  Copyright (c) 2016,2018 Tigera, Inc. All rights reserved.

package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../report/utils_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Utils Suite", []Reporter{junitReporter})
}
