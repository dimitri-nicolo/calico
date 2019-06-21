// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.
package middleware_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/middleware_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Middleware Suite", []Reporter{junitReporter})
}
