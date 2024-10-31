// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package syncer_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestSyncer(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/syncer_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Syncer Suite", []Reporter{junitReporter})
}
