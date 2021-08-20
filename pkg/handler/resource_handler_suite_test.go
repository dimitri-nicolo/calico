// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package handler_test

import (
	"testing"

	"github.com/onsi/ginkgo/reporters"

	"github.com/projectcalico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/resource_handler_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Handler Suite", []Reporter{junitReporter})
}
