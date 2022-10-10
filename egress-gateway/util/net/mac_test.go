package net

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestMacBuilder ensures that MAC builders deterministically and consistently generate the same MACs for the same nodenames
func TestMacBuilder(t *testing.T) {
	RegisterTestingT(t)

	mb1 := NewMACBuilder()
	mb2 := NewMACBuilder()

	const testNodeName string = "foo123./9-%"
	mac1, err := mb1.GenerateMAC(testNodeName)
	Expect(err).NotTo(HaveOccurred())
	mac2, err := mb2.GenerateMAC(testNodeName)
	Expect(err).NotTo(HaveOccurred())
	Expect(mac1.String()).To(BeIdenticalTo(mac2.String()))

	mac3, err := mb1.GenerateMAC(testNodeName)
	Expect(err).NotTo(HaveOccurred())
	Expect(mac1.String()).To(BeIdenticalTo(mac3.String()))
}
