// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package userscache

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/userscache.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "OIDCUserCache Suite", []Reporter{junitReporter})
}
