package syncclientutils_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/calico/typha/pkg/syncclientutils"
)

var _ = Describe("Test TyphaConfig", func() {

	BeforeEach(func() {
		os.Setenv("FELIX_TYPHACAFILE", "cafile")
		os.Setenv("FELIX_TYPHAFIPSMODEENABLED", "true")
		os.Setenv("FELIX_TYPHAREADTIMEOUT", "100")

	})

	It("should be able to read all types", func() {
		typhaConfig := syncclientutils.ReadTyphaConfig([]string{"FELIX_"})
		Expect(typhaConfig.CAFile).To(Equal("cafile"))
		Expect(typhaConfig.FIPSModeEnabled).To(BeTrue())
		Expect(typhaConfig.ReadTimeout.Seconds()).To(Equal(100.))
	})
})
