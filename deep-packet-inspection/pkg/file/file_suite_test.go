// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package file_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestFile(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/file_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "File Suite", []Reporter{junitReporter})
}
