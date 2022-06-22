package metrics_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/metrics_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Metrics Suite", []Reporter{junitReporter})
}
