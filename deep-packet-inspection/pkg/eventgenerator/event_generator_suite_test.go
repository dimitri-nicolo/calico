// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package eventgenerator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestReader(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/reader_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "EventGenerator Suite", []Reporter{junitReporter})
}
