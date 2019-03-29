package fv_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/ginkgo/reporters"
	//"github.com/projectcalico/libcalico-go/lib/testutils"
)

func init() {
	//testutils.HookLogrusForGinkgo()
}

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../report/fv_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "FV Suite", []Reporter{junitReporter})
}
