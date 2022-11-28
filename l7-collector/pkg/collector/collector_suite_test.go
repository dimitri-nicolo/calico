// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestCollector(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/collector_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Collector Suite", []Reporter{junitReporter})
}
