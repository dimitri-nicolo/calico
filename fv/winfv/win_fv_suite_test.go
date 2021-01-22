// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package winfv_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"testing"

	"github.com/onsi/ginkgo/reporters"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestNetworkingWindows(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(os.Getenv("REPORT"))
	RunSpecsWithDefaultAndCustomReporters(t, "Felix windows Suite", []Reporter{junitReporter})
}
