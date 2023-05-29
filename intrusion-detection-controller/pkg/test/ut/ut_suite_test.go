package ut

import (
	"testing"

	"github.com/projectcalico/calico/libcalico-go/lib/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDomainNameFeedsUT(t *testing.T) {
	testutils.HookLogrusForGinkgo()
	RegisterFailHandler(Fail)
	RunSpecs(t, "DomainName Thread Feeds UT")
}
