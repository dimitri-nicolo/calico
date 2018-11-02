// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package daemon_test

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
	junitReporter := reporters.NewJUnitReporter("../report/daemon_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Daemon Suite", []Reporter{junitReporter})
}
