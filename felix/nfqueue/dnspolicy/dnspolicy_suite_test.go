// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dnspolicy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestPolicysync(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/nfqueue_dnspolicy_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Nfqueue DNS Policy Test Suite", []Reporter{junitReporter})
}
