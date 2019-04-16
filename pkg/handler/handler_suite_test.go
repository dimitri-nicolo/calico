package handler_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/handler_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Handler Suite", []Reporter{junitReporter})
}
