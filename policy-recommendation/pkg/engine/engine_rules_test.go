// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package engine

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/lma/pkg/api"
	calicores "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
)

var _ = Describe("EngineRules", func() {
	const timeNowRFC3339 = "2022-11-30T09:01:38Z"

	var (
		er *engineRules

		port45  = uint16(45)
		port48  = uint16(48)
		port443 = uint16(443)
		port444 = uint16(444)
		port445 = uint16(445)
		port55  = uint16(55)
		port56  = uint16(56)
	)

	BeforeEach(func() {
		// Initialize a new engineRules object before each test
		er = NewEngineRules()
	})

	Context("when adding flows to egress to domain rules", func() {
		It("should add the flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.EgressTraffic, flow: api.Flow{}},
				{direction: calicores.IngressTraffic, flow: api.Flow{}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Domains: "www.some-domain.com", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Domains: "www.some-empty-protocol-domain.com", Port: &port444}}}, // Empty Protocol
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Domains: "www.empty-ports-domain.com"}}},   // Empty Ports
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Domains: "www.some-domain.com", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Domains: "www.some-other-domain.com", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Domains: "www.some-domain.com", Port: &port444}}},  // no update necessary
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Domains: "www.some-icmp-domain.com", Port: nil}}}, // no update necessary
			}

			key1 := egressToDomainRuleKey{protocol: protocolUDP, port: numorstring.Port{MinPort: port444, MaxPort: port444}}
			key2 := egressToDomainRuleKey{protocol: protocolUDP, port: numorstring.Port{}}
			key3 := egressToDomainRuleKey{protocol: protocolICMP, port: numorstring.Port{}}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.egressToDomainRules = map[egressToDomainRuleKey]*egressToDomainRule{
				key1: &egressToDomainRule{domains: []string{"www.some-domain.com", "www.some-other-domain.com"}, protocol: protocolUDP, port: numorstring.Port{MinPort: port444, MaxPort: port444}, timestamp: timeNowRFC3339},
				key2: &egressToDomainRule{domains: []string{"www.empty-ports-domain.com"}, protocol: protocolUDP, port: numorstring.Port{}, timestamp: timeNowRFC3339},
				key3: &egressToDomainRule{domains: []string{"www.some-icmp-domain.com"}, protocol: protocolICMP, port: numorstring.Port{}, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 3

			for _, td := range testData {
				er.addFlowToEgressToDomainRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to domain rules contains the expected rules
			Expect(er.egressToDomainRules).To(HaveKeyWithValue(key1, expectedEngineRules.egressToDomainRules[key1]))
			Expect(er.egressToDomainRules).To(HaveKeyWithValue(key2, expectedEngineRules.egressToDomainRules[key2]))
			Expect(er.egressToDomainRules).To(HaveKeyWithValue(key3, expectedEngineRules.egressToDomainRules[key3]))

			// The other engine rules should be empty
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})
	})

	Context("when adding flow to egress to service rules", func() {
		It("should add the flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.EgressTraffic, flow: api.Flow{}},
				{direction: calicores.IngressTraffic, flow: api.Flow{}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc1", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{ServiceName: "svc2", Port: &port444}}},       // Empty Protocol
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc3"}}}, // Empty Ports
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc4", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc3", Port: &port55}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc4", Port: &port444}}}, // No update necessary
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{ServiceName: "svc5-icmp", Port: nil}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc3", Port: &port45}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc3", Port: &port48}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{ServiceName: "svc3", Port: &port56}}},
			}

			key1 := egressToServiceRuleKey{name: "svc1", protocol: protocolUDP}
			key2 := egressToServiceRuleKey{name: "svc3", protocol: protocolUDP}
			key3 := egressToServiceRuleKey{name: "svc4", protocol: protocolUDP}
			key4 := egressToServiceRuleKey{name: "svc5-icmp", protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.egressToServiceRules = map[egressToServiceRuleKey]*egressToServiceRule{
				key1: &egressToServiceRule{name: "svc1", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key2: &egressToServiceRule{name: "svc3", ports: []numorstring.Port{{}, {MinPort: port55, MaxPort: port55}, {MinPort: port45, MaxPort: port45}, {MinPort: port48, MaxPort: port48}, {MinPort: port56, MaxPort: port56}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &egressToServiceRule{name: "svc4", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key4: &egressToServiceRule{name: "svc5-icmp", ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 4

			for _, td := range testData {
				er.addFlowToEgressToServiceRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.egressToServiceRules).To(HaveKeyWithValue(key1, expectedEngineRules.egressToServiceRules[key1]))
			Expect(er.egressToServiceRules).To(HaveKeyWithValue(key2, expectedEngineRules.egressToServiceRules[key2]))
			Expect(er.egressToServiceRules).To(HaveKeyWithValue(key3, expectedEngineRules.egressToServiceRules[key3]))
			Expect(er.egressToServiceRules).To(HaveKeyWithValue(key4, expectedEngineRules.egressToServiceRules[key4]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})
	})

	Context("when adding flow namespace rules", func() {
		It("should add the egress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.EgressTraffic, flow: api.Flow{}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns1", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Namespace: "ns2", Port: &port444}}},       // Empty Protocol
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns3"}}}, // Empty Ports
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns4", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns3", Port: &port55}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns4", Port: &port444}}}, // No update necessary
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Namespace: "ns5-icmp", Port: nil}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns3", Port: &port45}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns3", Port: &port48}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Namespace: "ns3", Port: &port56}}},
			}

			key1 := namespaceRuleKey{namespace: "ns1", protocol: protocolUDP}
			key2 := namespaceRuleKey{namespace: "ns3", protocol: protocolUDP}
			key3 := namespaceRuleKey{namespace: "ns4", protocol: protocolUDP}
			key4 := namespaceRuleKey{namespace: "ns5-icmp", protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.namespaceRules = map[namespaceRuleKey]*namespaceRule{
				key1: &namespaceRule{namespace: "ns1", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key2: &namespaceRule{namespace: "ns3", ports: []numorstring.Port{{}, {MinPort: port55, MaxPort: port55}, {MinPort: port45, MaxPort: port45}, {MinPort: port48, MaxPort: port48}, {MinPort: port56, MaxPort: port56}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &namespaceRule{namespace: "ns4", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key4: &namespaceRule{namespace: "ns5-icmp", ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 4

			for _, td := range testData {
				er.addFlowToNamespaceRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.namespaceRules).To(HaveKeyWithValue(key1, expectedEngineRules.namespaceRules[key1]))
			Expect(er.namespaceRules).To(HaveKeyWithValue(key2, expectedEngineRules.namespaceRules[key2]))
			Expect(er.namespaceRules).To(HaveKeyWithValue(key3, expectedEngineRules.namespaceRules[key3]))
			Expect(er.namespaceRules).To(HaveKeyWithValue(key4, expectedEngineRules.namespaceRules[key4]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})

		It("should add the ingress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.IngressTraffic, flow: api.Flow{}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns1"}, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Source: api.FlowEndpointData{Namespace: "ns2"}, Destination: api.FlowEndpointData{Port: &port444}}},         // Empty Protocol
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns3"}, Destination: api.FlowEndpointData{}}}, // Empty Ports
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns4"}, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns3"}, Destination: api.FlowEndpointData{Port: &port55}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns4"}, Destination: api.FlowEndpointData{Port: &port444}}}, // No update necessary
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Source: api.FlowEndpointData{Namespace: "ns5-icmp"}, Destination: api.FlowEndpointData{Port: nil}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns4"}, Destination: api.FlowEndpointData{Port: &port443}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Namespace: "ns4"}, Destination: api.FlowEndpointData{Port: &port445}}},
			}

			key1 := namespaceRuleKey{namespace: "ns1", protocol: protocolUDP}
			key2 := namespaceRuleKey{namespace: "ns3", protocol: protocolUDP}
			key3 := namespaceRuleKey{namespace: "ns4", protocol: protocolUDP}
			key4 := namespaceRuleKey{namespace: "ns5-icmp", protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.namespaceRules = map[namespaceRuleKey]*namespaceRule{
				key1: &namespaceRule{namespace: "ns1", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key2: &namespaceRule{namespace: "ns3", ports: []numorstring.Port{{}, {MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &namespaceRule{namespace: "ns4", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}, {MinPort: port443, MaxPort: port443}, {MinPort: port445, MaxPort: port445}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key4: &namespaceRule{namespace: "ns5-icmp", ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 4

			for _, td := range testData {
				er.addFlowToNamespaceRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.namespaceRules).To(HaveKeyWithValue(key1, expectedEngineRules.namespaceRules[key1]))
			Expect(er.namespaceRules).To(HaveKeyWithValue(key2, expectedEngineRules.namespaceRules[key2]))
			Expect(er.namespaceRules).To(HaveKeyWithValue(key3, expectedEngineRules.namespaceRules[key3]))
			Expect(er.namespaceRules).To(HaveKeyWithValue(key4, expectedEngineRules.namespaceRules[key4]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})
	})

	Context("when adding flow networkset rules", func() {
		It("should add the egress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.EgressTraffic, flow: api.Flow{}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Name: "netset1", Namespace: "ns1", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Name: "netset2", Namespace: "ns2", Port: &port444}}},       // Empty Protocol
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Name: "netset3", Namespace: "ns3"}}}, // Empty Ports
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Name: "netset4", Namespace: "ns4", Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Name: "netset5", Namespace: "", Port: &port55}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Name: "netset4", Namespace: "ns4", Port: &port444}}}, // No update necessary
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Name: "netset6-icmp", Namespace: "ns6", Port: nil}}},
			}

			key1 := networkSetRuleKey{global: false, name: "netset1", namespace: "ns1", protocol: protocolUDP}
			key2 := networkSetRuleKey{global: false, name: "netset3", namespace: "ns3", protocol: protocolUDP}
			key3 := networkSetRuleKey{global: false, name: "netset4", namespace: "ns4", protocol: protocolUDP}
			key4 := networkSetRuleKey{global: true, name: "netset5", namespace: "", protocol: protocolUDP}
			key5 := networkSetRuleKey{global: false, name: "netset6-icmp", namespace: "ns6", protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.networkSetRules = map[networkSetRuleKey]*networkSetRule{
				key1: &networkSetRule{global: false, name: "netset1", namespace: "ns1", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key2: &networkSetRule{global: false, name: "netset3", namespace: "ns3", ports: []numorstring.Port{{}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &networkSetRule{global: false, name: "netset4", namespace: "ns4", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key4: &networkSetRule{global: true, name: "netset5", namespace: "", ports: []numorstring.Port{{MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key5: &networkSetRule{global: false, name: "netset6-icmp", namespace: "ns6", ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 5

			for _, td := range testData {
				er.addFlowToNetworkSetRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.networkSetRules).To(HaveKeyWithValue(key1, expectedEngineRules.networkSetRules[key1]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key2, expectedEngineRules.networkSetRules[key2]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key3, expectedEngineRules.networkSetRules[key3]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key4, expectedEngineRules.networkSetRules[key4]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key5, expectedEngineRules.networkSetRules[key5]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})

		It("should add the ingress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.IngressTraffic, flow: api.Flow{}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Name: "netset1", Namespace: "ns1"}, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Source: api.FlowEndpointData{Name: "netset2", Namespace: "ns2"}, Destination: api.FlowEndpointData{Port: &port444}}},         // Empty Protocol
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Name: "netset3", Namespace: "ns3"}, Destination: api.FlowEndpointData{}}}, // Empty Ports
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Name: "netset4", Namespace: "ns4"}, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Name: "netset5", Namespace: ""}, Destination: api.FlowEndpointData{Port: &port55}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Source: api.FlowEndpointData{Name: "netset4", Namespace: "ns4"}, Destination: api.FlowEndpointData{Port: &port444}}}, // No update necessary
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Source: api.FlowEndpointData{Name: "netset6-icmp", Namespace: "ns6"}, Destination: api.FlowEndpointData{Port: nil}}},
			}

			key1 := networkSetRuleKey{global: false, name: "netset1", namespace: "ns1", protocol: protocolUDP}
			key2 := networkSetRuleKey{global: false, name: "netset3", namespace: "ns3", protocol: protocolUDP}
			key3 := networkSetRuleKey{global: false, name: "netset4", namespace: "ns4", protocol: protocolUDP}
			key4 := networkSetRuleKey{global: true, name: "netset5", namespace: "", protocol: protocolUDP}
			key5 := networkSetRuleKey{global: false, name: "netset6-icmp", namespace: "ns6", protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.networkSetRules = map[networkSetRuleKey]*networkSetRule{
				key1: &networkSetRule{global: false, name: "netset1", namespace: "ns1", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key2: &networkSetRule{global: false, name: "netset3", namespace: "ns3", ports: []numorstring.Port{{}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &networkSetRule{global: false, name: "netset4", namespace: "ns4", ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key4: &networkSetRule{global: true, name: "netset5", namespace: "", ports: []numorstring.Port{{MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key5: &networkSetRule{global: false, name: "netset6-icmp", namespace: "ns6", ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 5

			for _, td := range testData {
				er.addFlowToNetworkSetRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.networkSetRules).To(HaveKeyWithValue(key1, expectedEngineRules.networkSetRules[key1]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key2, expectedEngineRules.networkSetRules[key2]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key3, expectedEngineRules.networkSetRules[key3]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key4, expectedEngineRules.networkSetRules[key4]))
			Expect(er.networkSetRules).To(HaveKeyWithValue(key5, expectedEngineRules.networkSetRules[key5]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})
	})

	Context("when adding flow to private network rules", func() {
		It("should add the egress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.EgressTraffic, flow: api.Flow{}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Port: &port444}}},         // Empty Protocol
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{}}}, // Empty Ports
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port55}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}}, // No update necessary
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Port: nil}}},
			}

			key1 := privateNetworkRuleKey{protocol: protocolTCP}
			key2 := privateNetworkRuleKey{protocol: protocolUDP}
			key3 := privateNetworkRuleKey{protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.privateNetworkRules = map[privateNetworkRuleKey]*privateNetworkRule{
				key1: &privateNetworkRule{ports: []numorstring.Port{{}, {MinPort: port444, MaxPort: port444}}, protocol: protocolTCP, timestamp: timeNowRFC3339},
				key2: &privateNetworkRule{ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}, {MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &privateNetworkRule{ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 3

			for _, td := range testData {
				er.addFlowToPrivateNetworkRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.privateNetworkRules).To(HaveKeyWithValue(key1, expectedEngineRules.privateNetworkRules[key1]))
			Expect(er.privateNetworkRules).To(HaveKeyWithValue(key2, expectedEngineRules.privateNetworkRules[key2]))
			Expect(er.privateNetworkRules).To(HaveKeyWithValue(key3, expectedEngineRules.privateNetworkRules[key3]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})

		It("should add the ingress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.IngressTraffic, flow: api.Flow{}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Port: &port444}}},         // Empty Protocol
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{}}}, // Empty Ports
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port55}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}}, // No update necessary
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Port: nil}}},
			}

			key1 := privateNetworkRuleKey{protocol: protocolTCP}
			key2 := privateNetworkRuleKey{protocol: protocolUDP}
			key3 := privateNetworkRuleKey{protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.privateNetworkRules = map[privateNetworkRuleKey]*privateNetworkRule{
				key1: &privateNetworkRule{ports: []numorstring.Port{{}, {MinPort: port444, MaxPort: port444}}, protocol: protocolTCP, timestamp: timeNowRFC3339},
				key2: &privateNetworkRule{ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}, {MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &privateNetworkRule{ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 3

			for _, td := range testData {
				er.addFlowToPrivateNetworkRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.privateNetworkRules).To(HaveKeyWithValue(key1, expectedEngineRules.privateNetworkRules[key1]))
			Expect(er.privateNetworkRules).To(HaveKeyWithValue(key2, expectedEngineRules.privateNetworkRules[key2]))
			Expect(er.privateNetworkRules).To(HaveKeyWithValue(key3, expectedEngineRules.privateNetworkRules[key3]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.publicNetworkRules)).To(Equal(0))
		})
	})

	Context("when adding flow to public network rules", func() {
		It("should add the egress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.EgressTraffic, flow: api.Flow{}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Port: &port444}}},         // Empty Protocol
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{}}}, // Empty Ports
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port55}}},
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}}, // No update necessary
				{direction: calicores.EgressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Port: nil}}},
			}

			key1 := publicNetworkRuleKey{protocol: protocolTCP}
			key2 := publicNetworkRuleKey{protocol: protocolUDP}
			key3 := publicNetworkRuleKey{protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.publicNetworkRules = map[publicNetworkRuleKey]*publicNetworkRule{
				key1: &publicNetworkRule{ports: []numorstring.Port{{}, {MinPort: port444, MaxPort: port444}}, protocol: protocolTCP, timestamp: timeNowRFC3339},
				key2: &publicNetworkRule{ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}, {MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &publicNetworkRule{ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 3

			for _, td := range testData {
				er.addFlowToPublicNetworkRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.publicNetworkRules).To(HaveKeyWithValue(key1, expectedEngineRules.publicNetworkRules[key1]))
			Expect(er.publicNetworkRules).To(HaveKeyWithValue(key2, expectedEngineRules.publicNetworkRules[key2]))
			Expect(er.publicNetworkRules).To(HaveKeyWithValue(key3, expectedEngineRules.publicNetworkRules[key3]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
		})

		It("should add the ingress flows correctly", func() {
			testData := []struct {
				direction calicores.DirectionType
				flow      api.Flow
			}{
				{direction: calicores.IngressTraffic, flow: api.Flow{}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Destination: api.FlowEndpointData{Port: &port444}}},         // Empty Protocol
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{}}}, // Empty Ports
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoUDP, Destination: api.FlowEndpointData{Port: &port55}}},
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoTCP, Destination: api.FlowEndpointData{Port: &port444}}}, // No update necessary
				{direction: calicores.IngressTraffic, flow: api.Flow{Proto: &api.ProtoICMP, Destination: api.FlowEndpointData{Port: nil}}},
			}

			key1 := publicNetworkRuleKey{protocol: protocolTCP}
			key2 := publicNetworkRuleKey{protocol: protocolUDP}
			key3 := publicNetworkRuleKey{protocol: protocolICMP}

			expectedEngineRules := NewEngineRules()
			expectedEngineRules.publicNetworkRules = map[publicNetworkRuleKey]*publicNetworkRule{
				key1: &publicNetworkRule{ports: []numorstring.Port{{}, {MinPort: port444, MaxPort: port444}}, protocol: protocolTCP, timestamp: timeNowRFC3339},
				key2: &publicNetworkRule{ports: []numorstring.Port{{MinPort: port444, MaxPort: port444}, {MinPort: port55, MaxPort: port55}}, protocol: protocolUDP, timestamp: timeNowRFC3339},
				key3: &publicNetworkRule{ports: []numorstring.Port{{}}, protocol: protocolICMP, timestamp: timeNowRFC3339},
			}

			expectedNumberOfRules := 3

			for _, td := range testData {
				er.addFlowToPublicNetworkRules(td.direction, td.flow, mockRealClock{})
			}

			Expect(er.size).To(Equal(expectedNumberOfRules))

			// The egress to service rules contains the expected rules
			Expect(er.publicNetworkRules).To(HaveKeyWithValue(key1, expectedEngineRules.publicNetworkRules[key1]))
			Expect(er.publicNetworkRules).To(HaveKeyWithValue(key2, expectedEngineRules.publicNetworkRules[key2]))
			Expect(er.publicNetworkRules).To(HaveKeyWithValue(key3, expectedEngineRules.publicNetworkRules[key3]))

			// The other engine rules should be empty
			Expect(len(er.egressToDomainRules)).To(Equal(0))
			Expect(len(er.egressToServiceRules)).To(Equal(0))
			Expect(len(er.namespaceRules)).To(Equal(0))
			Expect(len(er.networkSetRules)).To(Equal(0))
			Expect(len(er.privateNetworkRules)).To(Equal(0))
		})
	})
})
