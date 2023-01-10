// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package users

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/elasticsearch_users_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Elasticsearch Users Suite", []Reporter{junitReporter})
}
