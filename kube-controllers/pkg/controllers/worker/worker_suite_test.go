// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package worker

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("./report/worker_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Worker Suite", []Reporter{junitReporter})
}
