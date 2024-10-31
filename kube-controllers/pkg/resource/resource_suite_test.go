// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package resource

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("./report/copy_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Copy Suite", []Reporter{junitReporter})
}
