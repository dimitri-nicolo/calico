// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package labels

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/labels_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "labels suite", []Reporter{junitReporter})
}
