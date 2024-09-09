package earlynetworking_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"
)

func init() {
	testutils.HookLogrusForGinkgo()
}

func TestEarlyNetworking(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/earlynetworking_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "EarlyNetworking Suite", []Reporter{junitReporter})
}
