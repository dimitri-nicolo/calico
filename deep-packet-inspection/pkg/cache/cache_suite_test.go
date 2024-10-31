// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package cache_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func TestCache(t *testing.T) {
	testutils.HookLogrusForGinkgo()

	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/cache.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Cache Suite", []Reporter{junitReporter})
}
