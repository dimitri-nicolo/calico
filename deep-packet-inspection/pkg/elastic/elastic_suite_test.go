// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package elastic_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestElastic(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/elastic_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Elastic Suite", []Reporter{junitReporter})
}
