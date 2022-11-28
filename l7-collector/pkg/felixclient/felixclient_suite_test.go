// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package felixclient

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestCollector(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/felixclient_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Felix Client Suite", []Reporter{junitReporter})
}
