// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package managedcluster

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/managedcluster_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Managed cluster controller Suite", []Reporter{junitReporter})
}
