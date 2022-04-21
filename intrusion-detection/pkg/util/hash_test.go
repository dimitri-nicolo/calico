package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HashUtils", func() {
	Context("ComputeSha256HashWithLimit", func() {
		It("generate sha256 hashstring for a small string char much less than expected limit", func() {
			smallString := "a"

			result := ComputeSha256HashWithLimit(smallString, 10)
			Expect(result).ToNot(BeNil())
			Expect(len(result) < 10).To(BeTrue())
		})

		It("generate sha256 hashstring for a small string char much less than expected limit", func() {
			smallString := "cluster-test-globalalert-name"

			result := ComputeSha256HashWithLimit(smallString, 10)
			Expect(result).ToNot(BeNil())
			Expect(len(result) == 10).To(BeTrue())
		})
	})
})
