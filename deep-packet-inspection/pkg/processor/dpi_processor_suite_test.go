// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package processor_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestDPIProcessor(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/dpi_processor_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Processor Suite", []Reporter{junitReporter})
}
