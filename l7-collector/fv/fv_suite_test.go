// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package fv_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestFv(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../report/fv_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "FV Suite", []Reporter{junitReporter})
}
