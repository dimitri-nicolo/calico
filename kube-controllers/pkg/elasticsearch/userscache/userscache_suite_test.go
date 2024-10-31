// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package userscache

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("./report/userscache.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "OIDCUserCache Suite", []Reporter{junitReporter})
}
