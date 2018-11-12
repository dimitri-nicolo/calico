package winpol

import (
	"encoding/json"
	"net"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var mgmtIPNet *net.IPNet
var mgmtIP net.IP

func init() {
	var err error
	mgmtIP, mgmtIPNet, err = net.ParseCIDR("10.11.128.13/19")
	if err != nil {
		panic(err)
	}
	mgmtIPNet.IP = mgmtIP // We want the full IP, not the masked version.
}

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
	pols, err := CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2, mgmtIPNet}, false, mgmtIP, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"Type": "SomethingElse"}`),
		json.RawMessage(`{"DestinationPrefix":"10.11.128.13/32","NeedEncap":true,"Type":"ROUTE"}`),
	}), "OutBoundNAT should have been filtered out")

	t.Log("With NAT enabled, OutBoundNAT should be augmented")
	pols, err = CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2, mgmtIPNet}, true, mgmtIP, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"ExceptionList":["10.96.0.0/12","10.0.1.0/24","10.0.2.0/24","10.11.128.0/19"],"Type":"OutBoundNAT"}`),
		json.RawMessage(`{"Type": "SomethingElse"}`),
		json.RawMessage(`{"DestinationPrefix":"10.11.128.13/32","NeedEncap":true,"Type":"ROUTE"}`),
	}))

	t.Log("With NAT enabled, and no OutBoundNAT stanza, OutBoundNAT should be added")
	marshaller = newMockPolMarshaller(
		`{"Type": "SomethingElse"}`,
	)
	pols, err = CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2, mgmtIPNet}, true, mgmtIP, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"Type": "SomethingElse"}`),
		json.RawMessage(`{"ExceptionList":["10.0.1.0/24","10.0.2.0/24","10.11.128.0/19"],"Type":"OutBoundNAT"}`),
		json.RawMessage(`{"DestinationPrefix":"10.11.128.13/32","NeedEncap":true,"Type":"ROUTE"}`),
	}))

	t.Log("With NAT disabled, and no OutBoundNAT stanza, OutBoundNAT should not be added")
	pols, err = CalculateEndpointPolicies(marshaller, []*net.IPNet{net1, net2, mgmtIPNet}, false, mgmtIP, logger)
	Expect(err).NotTo(HaveOccurred())
	Expect(pols).To(Equal([]json.RawMessage{
		json.RawMessage(`{"Type": "SomethingElse"}`),
		json.RawMessage(`{"DestinationPrefix":"10.11.128.13/32","NeedEncap":true,"Type":"ROUTE"}`),
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
