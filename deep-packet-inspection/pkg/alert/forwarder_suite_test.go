// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package alert_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestAlert(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/alert_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Alert Suite", []Reporter{junitReporter})
}
