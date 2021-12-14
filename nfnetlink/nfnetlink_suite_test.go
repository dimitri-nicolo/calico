// Copyright (c) 2016-2020 Tigera, Inc. All rights reserved.
package nfnetlink_test

import (
	"testing"

	"github.com/onsi/ginkgo/reporters"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/logutils"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func TestNfnetlink(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../report/ip_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Nfnetlink Suite", []Reporter{junitReporter})
}

func init() {
	testutils.HookLogrusForGinkgo()
	logrus.AddHook(&logutils.ContextHook{})
	logrus.SetFormatter(&logutils.Formatter{})
}
