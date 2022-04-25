// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dispatcher_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestDispatcher(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/dispatcher_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Dispatcher Suite", []Reporter{junitReporter})
}
