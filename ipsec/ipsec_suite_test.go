// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"

	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../report/ipsec_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "IPSec Suite", []Reporter{junitReporter})
}
