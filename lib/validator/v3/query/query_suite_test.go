// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package query

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

func TestQueryValidator(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../../../report/v3_query_validator_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "v3 Query Validator Suite", []Reporter{junitReporter})
}
