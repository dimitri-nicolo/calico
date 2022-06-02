package policycalc

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPolicyCalc(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "PolicyCalc Suite")
}
