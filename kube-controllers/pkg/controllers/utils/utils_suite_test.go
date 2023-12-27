// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package utils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	"github.com/onsi/ginkgo/reporters"
)

func init() {
	testutils.HookLogrusForGinkgo()
	logrus.SetLevel(logrus.DebugLevel)
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/utils_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Utils Suite", []Reporter{junitReporter})
}
