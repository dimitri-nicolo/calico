// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestHoneypodController(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/capture_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "HoneypodController Suite", []Reporter{junitReporter})
}
