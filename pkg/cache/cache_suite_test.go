// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package cache_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	"github.com/onsi/ginkgo/reporters"
)

func TestCache(t *testing.T) {
	testutils.HookLogrusForGinkgo()

	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/cache.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Cache Suite", []Reporter{junitReporter})
}
