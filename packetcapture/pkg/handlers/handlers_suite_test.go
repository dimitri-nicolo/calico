// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handlers_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestHandlers(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/handlers_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Handlers Suite", []Reporter{junitReporter})
}
