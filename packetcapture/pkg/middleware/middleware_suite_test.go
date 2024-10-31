// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package middleware_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestMiddleware(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/middleware_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Middleware Suite", []Reporter{junitReporter})
}
