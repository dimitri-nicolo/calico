package winpol

import (
	"encoding/json"
	"net"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestCalculateEndpointPolicies(t *testing.T) {
	RegisterTestingT(t)

	marshaller := newMockPolMarshaller(
		`{"Type": "OutBoundNAT", "ExceptionList": ["10.96.0.0/12"]}`,
		`{"Type": "SomethingElse"}`,
	)
	logger := logrus.WithField("test", "true")

	_, net1, _ := net.ParseCIDR("10.0.1.0/24")
	_, net2, _ := net.ParseCIDR("10.0.2.0/24")

	t.Log("With NAT disabled, OutBoundNAT should be filtered out")
	pols, err := CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2}, false, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"Type": "SomethingElse"}`),
	}), "OutBoundNAT should have been filtered out")

	t.Log("With NAT enabled, OutBoundNAT should be augmented")
	pols, err = CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2}, true, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"ExceptionList":["10.96.0.0/12","10.0.1.0/24","10.0.2.0/24"],"Type":"OutBoundNAT"}`),
		json.RawMessage(`{"Type": "SomethingElse"}`),
	}))

	t.Log("With NAT enabled, and no OutBoundNAT stanza, OutBoundNAT should be added")
	marshaller = newMockPolMarshaller(
		`{"Type": "SomethingElse"}`,
	)
	pols, err = CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2}, true, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"Type": "SomethingElse"}`),
		json.RawMessage(`{"ExceptionList":["10.0.1.0/24","10.0.2.0/24"],"Type":"OutBoundNAT"}`),
	}))

	t.Log("With NAT disabled, and no OutBoundNAT stanza, OutBoundNAT should not be added")
	pols, err = CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2}, false, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"Type": "SomethingElse"}`),
	}))
}

func newMockPolMarshaller(pols ...string) mockPolMarshaller {
	return mockPolMarshaller(pols)
}

type mockPolMarshaller []string

func (m mockPolMarshaller) MarshalPolicies() (out []json.RawMessage) {
	for _, p := range m {
		out = append(out, json.RawMessage(p))
	}
	return
}
