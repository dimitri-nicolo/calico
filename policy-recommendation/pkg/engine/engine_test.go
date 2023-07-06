// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.
package engine

import (
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/api/pkg/lib/numorstring"

	"github.com/projectcalico/calico/lma/pkg/api"
	calres "github.com/projectcalico/calico/policy-recommendation/pkg/calico-resources"
	testutils "github.com/projectcalico/calico/policy-recommendation/tests/utils"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

const (
	testDataFile = "../../tests/data/flows.json"

	// serviceName1 = "serviceName1"
	// serviceName2 = "serviceName2"
	// serviceName3 = "serviceName3"
	namespace1 = "ns1"
	namespace2 = "ns2"
	namespace3 = "ns3"

	timeNowRFC3339 = "2022-11-30T09:01:38Z"
)

type mockRealClock struct{}

func (mockRealClock) NowRFC3339() string { return timeNowRFC3339 }

var mrc mockRealClock

var _ = Describe("processFlow", func() {
	const serviceNameSuffix = "svc.cluster.local"

	var (
		recEngine *recommendationEngine

		flowData []api.Flow

		name      = "test_name"
		namespace = "namespace1"
		tier      = "test_tier"
		order     = float64(1)

		interval      = time.Duration(150 * time.Second)
		stabilization = time.Duration(10 * time.Minute)

		clock = mrc
	)

	BeforeEach(func() {
		recEngine = newRecommendationEngine(
			name, namespace, tier, &order, clock, interval, stabilization, serviceNameSuffix, *log.WithField("cluster", "my-cluster"))

		err := testutils.LoadData(testDataFile, &flowData)
		Expect(err).To(BeNil())
	})

	It("Test valid engine rule generation", func() {
		for _, data := range flowData {
			recEngine.processFlow(&data)
		}

		Expect(len(recEngine.egress.namespaceRules)).To(Equal(2))
		Expect(recEngine.egress.namespaceRules[namespaceRuleKey{namespace: "namespace1", protocol: protocolTCP}]).
			To(Equal(&namespaceRule{namespace: "namespace1", protocol: protocolTCP, ports: ports1, timestamp: "2022-11-30T09:01:38Z"}))
		Expect(recEngine.egress.namespaceRules[namespaceRuleKey{namespace: "namespace2", protocol: protocolTCP}]).
			To(Equal(&namespaceRule{namespace: "namespace2", protocol: protocolTCP, ports: ports2, timestamp: "2022-11-30T09:01:38Z"}))

		Expect(len(recEngine.ingress.namespaceRules)).To(Equal(1))
		Expect(recEngine.ingress.namespaceRules[namespaceRuleKey{namespace: "namespace1", protocol: protocolTCP}]).
			To(Equal(&namespaceRule{namespace: "namespace1", protocol: protocolTCP, ports: ports1, timestamp: "2022-11-30T09:01:38Z"}))
	})

	It("Test flow with ActionFlagDeny", func() {
		flow := &api.Flow{
			ActionFlag: api.ActionFlagDeny,
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test flow with ActionFlagEndOfTierDeny", func() {
		flow := &api.Flow{
			ActionFlag: api.ActionFlagEndOfTierDeny,
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test 'src' reported flow that matches", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeSource,
			Source: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test 'src' reported flow that is not WEP", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeSource,
			Source: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeHEP,
				Namespace: namespace,
			},
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test 'src' reported flow where the source flow is not equal to the rec engine namespace", func() {
		namespace := "not-the-engine-namespace"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeSource,
			Source: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test 'dst' reported flow that matches", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeDestination,
			Destination: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test 'dst' reported flow that is not WEP", func() {
		namespace := "namespace1"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeDestination,
			Destination: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeHEP,
				Namespace: namespace,
			},
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))
	})

	It("Test 'dst' reported flow where the source flow is not equal to the rec engine namespace", func() {
		namespace := "not-the-engine-namespace"
		flow := &api.Flow{
			ActionFlag: api.ActionFlagAllow,
			Reporter:   api.ReporterTypeDestination,
			Destination: api.FlowEndpointData{
				Type:      api.FlowLogEndpointTypeWEP,
				Namespace: namespace,
			},
		}

		recEngine.processFlow(flow)
		Expect(recEngine.egress.size).To(Equal(0))
		Expect(recEngine.ingress.size).To(Equal(0))

	})
})

var _ = Describe("ProcessRecommendation", func() {
	const serviceNameSuffix = "svc.cluster.local"

	var (
		recEngine *recommendationEngine

		flowData []api.Flow

		name      = "test_name"
		namespace = "namespace1"
		tier      = "test_tier"
		order     = float64(1)

		interval      = time.Duration(150 * time.Second)
		stabilization = time.Duration(5 * time.Minute)

		clock = mrc
	)

	BeforeEach(func() {
		recEngine = newRecommendationEngine(
			name, namespace, tier, &order, clock, interval, stabilization, serviceNameSuffix, *log.WithField("cluster", "my-cluster"))

		err := testutils.LoadData(testDataFile, &flowData)
		Expect(err).To(BeNil())
	})

	// TODO(dimitrin): Add back UTs - [EV-2415] UTs
	It("Test valid engine rule generation", func() {
		for _, data := range flowData {
			recEngine.processFlow(&data)
		}

		Expect(len(recEngine.egress.namespaceRules)).To(Equal(2))
		Expect(len(recEngine.ingress.namespaceRules)).To(Equal(1))

		// Define a new staged network policy to place the egress rules.
		ctrl := true
		bod := false
		owner := metav1.OwnerReference{
			APIVersion:         "projectcalico.org/v3",
			Kind:               "PolicyRecommendationScope",
			Name:               "default",
			UID:                "orikr-9df4d-0k43m",
			Controller:         &ctrl,
			BlockOwnerDeletion: &bod,
		}

		snp := calres.NewStagedNetworkPolicy(utils.GetPolicyName(tier, "name1", func() string { return "xv5fb" }), "namespace1", tier, owner)
		snp.Spec.Egress = currentNamespaceRules

		recEngine.processRecommendation([]*api.Flow{}, snp)
		Expect(compareSnps(snp, &expectedSnp)).To(BeTrue())
	})
})

var _ = Describe("compPorts", func() {
	testCases := []struct {
		a        []numorstring.Port
		b        []numorstring.Port
		expected int
	}{
		{[]numorstring.Port{{MinPort: 0, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, -1},
		{[]numorstring.Port{{MinPort: 1, MaxPort: 1, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, -1},
		{[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 3, PortName: "A"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, -1},
		{[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, 0},
		{[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 5, PortName: "C"}}, 1},
		{[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "A"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, 1},
		{[]numorstring.Port{{MinPort: 1, MaxPort: 2, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, []numorstring.Port{{MinPort: 1, MaxPort: 1, PortName: "A"}, {MinPort: 3, MaxPort: 4, PortName: "B"}, {MinPort: 5, MaxPort: 6, PortName: "C"}}, 1},
	}

	for _, testCase := range testCases {
		It(fmt.Sprintf("returns %v when comparing %v and %v", testCase.expected, testCase.a, testCase.b), func() {
			Expect(compPorts(testCase.a, testCase.b)).To(Equal(testCase.expected))
		})
	}
})

var _ = Describe("CompStrArrays", func() {
	testCases := []struct {
		a        []string
		b        []string
		expected bool
	}{
		{[]string{"apple"}, []string{"Apple"}, false},
		{[]string{"apple", "banana"}, []string{"apple", "banana", "cherry"}, true},
		{[]string{"apple", "banana", "cherry"}, []string{"apple", "banana", "cherry"}, true},
		{[]string{"apple", "banana", "cherry"}, []string{"apple", "banana", "apple"}, false},
		{[]string{"apple", "banana", "cherry"}, []string{"banana", "cherry", "date"}, false},
		{[]string{"apple", "banana", "cherry"}, []string{"grape", "kiwi", "mango"}, true},
	}

	for _, testCase := range testCases {
		It(fmt.Sprintf("returns %v when comparing %v and %v", testCase.expected, testCase.a, testCase.b), func() {
			Expect(compStrArrays(testCase.a, testCase.b)).To(Equal(testCase.expected))
		})
	}
})

// compareSnps is a helper function used to compare the policy recommendation relevant parameters
// between two staged network policies.
func compareSnps(left, right *v3.StagedNetworkPolicy) bool {
	Expect(left.ObjectMeta.Name).To(Equal(right.ObjectMeta.Name))
	Expect(left.ObjectMeta.Namespace).To(Equal(right.ObjectMeta.Namespace))
	Expect(left.ObjectMeta.Labels).To(Equal(right.ObjectMeta.Labels))
	Expect(left.ObjectMeta.Annotations).To(Equal(right.ObjectMeta.Annotations))
	Expect(reflect.DeepEqual(left.ObjectMeta.OwnerReferences, right.ObjectMeta.OwnerReferences)).
		To(BeTrue(), "%+v should equal %+v", left.ObjectMeta.OwnerReferences, right.ObjectMeta.OwnerReferences)

	Expect(left.Spec.StagedAction).To(Equal(right.Spec.StagedAction))
	Expect(left.Spec.Tier).To(Equal(right.Spec.Tier))
	Expect(left.Spec.Selector).To(Equal(right.Spec.Selector))
	Expect(left.Spec.Types).To(Equal(right.Spec.Types))
	Expect(reflect.DeepEqual(left.Spec.Egress, left.Spec.Egress)).To(BeTrue())
	Expect(reflect.DeepEqual(left.Spec.Ingress, left.Spec.Ingress)).To(BeTrue())

	return true
}

var (
	ports1 = []numorstring.Port{
		{
			MinPort: 443,
			MaxPort: 443,
		},
	}

	ports2 = []numorstring.Port{
		{
			MinPort: 8080,
			MaxPort: 8080,
		},
		{
			MinPort: 5432,
			MaxPort: 5432,
		},
	}

	portsOrdered1 = []numorstring.Port{
		{
			MinPort: 5,
			MaxPort: 59,
		},
		{
			MinPort: 22,
			MaxPort: 22,
		},
		{
			MinPort: 44,
			MaxPort: 56,
		},
	}
	// portsOrdered2 = []numorstring.Port{
	// 	{
	// 		MinPort: 1,
	// 		MaxPort: 99,
	// 	},
	// 	{
	// 		MinPort: 3,
	// 		MaxPort: 3,
	// 	},
	// 	{
	// 		MinPort: 24,
	// 		MaxPort: 35,
	// 	},
	// }
	portsOrdered3 = []numorstring.Port{
		{
			MinPort: 8080,
			MaxPort: 8081,
		},
	}

	protocolTCP  = numorstring.ProtocolFromString("TCP")
	protocolUDP  = numorstring.ProtocolFromString("UDP")
	protocolICMP = numorstring.ProtocolFromString("ICMP")

	//TODO(dimitrin): Add back data for remaining UT tests.
	// EgressToDomain
	// currentEgressToDomainRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"calico.org"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"kubernetes.io"},
	// 			Ports:   portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 13:04:05 PST",
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"tigera.io"},
	// 			Ports:   portsOrdered2,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Thu, 30 Nov 2022 12:30:05 PST",
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// incomingEgressToDomainRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"calico.org"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.ScopeKey: "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"projectcalico.com"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.ScopeKey: "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"tigera.io"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.ScopeKey: "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// expectedEgressToDomainRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"calico.org"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:30:05 PST",
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"kubernetes.io"},
	// 			Ports:   portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 13:04:05 PST",
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"projectcalico.com"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"tigera.io"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// expectedEgressToDomainRulesEmptyCurrent = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"calico.org"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"projectcalico.com"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),s.clock
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Domains: []string{"tigera.io"},
	// 			Ports:   portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.ScopeKey:       "Domains",
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// // EgressToService
	// currentEgressToServiceRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace1,
	// 			},
	// 			Ports: portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Thu, 30 Nov 2022 06:04:05 PST",
	// 				calres.NameKey:        serviceName2,
	// 				calres.NamespaceKey:   namespace1,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName1,
	// 				Namespace: namespace1,
	// 			},
	// 			Ports: portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
	// 				calres.NameKey:        serviceName1,
	// 				calres.NamespaceKey:   namespace1,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace2,
	// 			},
	// 			Ports: portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
	// 				calres.NameKey:        serviceName2,
	// 				calres.NamespaceKey:   namespace2,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName3,
	// 				Namespace: namespace3,
	// 			},
	// 			Ports: portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
	// 				calres.NameKey:        serviceName3,
	// 				calres.NamespaceKey:   namespace3,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// incomingEgressToServiceRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace1,
	// 			},
	// 			Ports: portsOrdered2,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NameKey:      serviceName2,
	// 				calres.NamespaceKey: namespace1,
	// 				calres.ScopeKey:     string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace2,
	// 			},
	// 			Ports: portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NameKey:      serviceName2,
	// 				calres.NamespaceKey: namespace2,
	// 				calres.ScopeKey:     string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName1,
	// 				Namespace: namespace1,
	// 			},
	// 			Ports: portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NameKey:      serviceName1,
	// 				calres.NamespaceKey: namespace1,
	// 				calres.ScopeKey:     string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace2,
	// 			},
	// 			Ports: portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NameKey:      serviceName2,
	// 				calres.NamespaceKey: namespace2,
	// 				calres.ScopeKey:     string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// expectedEgressToServiceRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace1,
	// 			},
	// 			Ports: portsOrdered2,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.NameKey:        serviceName2,
	// 				calres.NamespaceKey:   namespace1,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace2,
	// 			},
	// 			Ports: portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.NameKey:        serviceName2,
	// 				calres.NamespaceKey:   namespace2,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName1,
	// 				Namespace: namespace1,
	// 			},
	// 			Ports: portsOrdered1,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
	// 				calres.NameKey:        serviceName1,
	// 				calres.NamespaceKey:   namespace1,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName2,
	// 				Namespace: namespace2,
	// 			},
	// 			Ports: portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
	// 				calres.NameKey:        serviceName2,
	// 				calres.NamespaceKey:   namespace2,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			Services: &v3.ServiceMatch{
	// 				Name:      serviceName3,
	// 				Namespace: namespace3,
	// 			},
	// 			Ports: portsOrdered3,
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
	// 				calres.NameKey:        serviceName3,
	// 				calres.NamespaceKey:   namespace3,
	// 				calres.ScopeKey:       string(calres.EgressToServiceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// Namespace
	currentNamespaceRules = []v3.Rule{
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace1,
				Ports:             portsOrdered1,
				Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Thu, 30 Nov 2022 06:04:05 PST",
					calres.NamespaceKey:   namespace1,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolTCP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace1,
				Ports:             portsOrdered1,
				Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
					calres.NamespaceKey:   namespace1,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace2,
				Ports:             portsOrdered3,
				Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NamespaceKey:   namespace2,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
		{
			Action: v3.Allow,
			Destination: v3.EntityRule{
				NamespaceSelector: namespace3,
				Ports:             portsOrdered3,
				Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
			},
			Metadata: &v3.RuleMetadata{
				Annotations: map[string]string{
					calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
					calres.NamespaceKey:   namespace3,
					calres.ScopeKey:       string(calres.NamespaceScope),
				},
			},
			Protocol: &protocolUDP,
		},
	}

	// incomingNamespaceRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace1,
	// 			Ports:             portsOrdered2,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NamespaceKey: namespace1,
	// 				calres.ScopeKey:     string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace2,
	// 			Ports:             portsOrdered1,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NamespaceKey: namespace2,
	// 				calres.ScopeKey:     string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace1,
	// 			Ports:             portsOrdered1,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NamespaceKey: namespace1,
	// 				calres.ScopeKey:     string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace2,
	// 			Ports:             portsOrdered3,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.NamespaceKey: namespace2,
	// 				calres.ScopeKey:     string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	// expectedNamespaceRules = []v3.Rule{
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace1,
	// 			Ports:             portsOrdered2,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.NamespaceKey:   namespace1,
	// 				calres.ScopeKey:       string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace2,
	// 			Ports:             portsOrdered1,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: mrc.NowRFC3339(),
	// 				calres.NamespaceKey:   namespace2,
	// 				calres.ScopeKey:       string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolTCP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace1,
	// 			Ports:             portsOrdered1,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:04:05 PST",
	// 				calres.NamespaceKey:   namespace1,
	// 				calres.ScopeKey:       string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace2,
	// 			Ports:             portsOrdered3,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
	// 				calres.NamespaceKey:   namespace2,
	// 				calres.ScopeKey:       string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// 	{
	// 		Action: v3.Allow,
	// 		Destination: v3.EntityRule{
	// 			NamespaceSelector: namespace3,
	// 			Ports:             portsOrdered3,
	// 			Selector:          fmt.Sprintf("%s/orchestrator == k8s", calres.PolicyRecKeyName),
	// 		},
	// 		Metadata: &v3.RuleMetadata{
	// 			Annotations: map[string]string{
	// 				calres.LastUpdatedKey: "Wed, 29 Nov 2022 14:05:05 PST",
	// 				calres.NamespaceKey:   namespace3,
	// 				calres.ScopeKey:       string(calres.NamespaceScope),
	// 			},
	// 		},
	// 		Protocol: &protocolUDP,
	// 	},
	// }

	expectedOrder          = float64(1)
	expectedCtrl           = true
	exptedBlockOwnerDelete = false
	expectedSnp            = v3.StagedNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test_tier.name1-xv5fb",
			Namespace: "namespace1",
			Labels: map[string]string{
				"policyrecommendation.tigera.io/scope":  "namespace",
				"projectcalico.org/spec.stagedAction":   "Learn",
				"projectcalico.org/tier":                "test_tier",
				"projectcalico.org/ownerReference.kind": "PolicyRecommendationScope",
			},
			Annotations: map[string]string{
				"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
				"policyrecommendation.tigera.io/status":      "NoData",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "projectcalico.org/v3",
					Kind:               "PolicyRecommendationScope",
					Name:               "default",
					UID:                "orikr-9df4d-0k43m",
					Controller:         &expectedCtrl,
					BlockOwnerDeletion: &exptedBlockOwnerDelete,
				},
			},
		},
		Spec: v3.StagedNetworkPolicySpec{
			StagedAction: v3.StagedActionLearn,
			Tier:         "test_tier",
			Order:        &expectedOrder,
			Selector:     "projectcalico.org/namespace == 'namespace1'",
			Types: []v3.PolicyType{
				"Egress", "Ingress",
			},
			Egress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/orchestrator == k8s",
						NamespaceSelector: "projectcalico.org/name == namespace1",
						Ports: []numorstring.Port{
							{
								MinPort: 443,
								MaxPort: 443,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Mon, 05 Dec 2022 06:00:20 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"projectcalico.org/lastUpdated":              "2022-12-05 06:00:20.239538363 -0800 PST m=+0.033443085",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "projectcalico.org/orchestrator == k8s",
						NamespaceSelector: "projectcalico.org/name == namespace2",
						Ports: []numorstring.Port{
							{
								MinPort: 5432,
								MaxPort: 8080,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Mon, 05 Dec 2022 06:00:20 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"projectcalico.org/lastUpdated":              "2022-12-05 06:00:20.239549329 -0800 PST m=+0.033454042",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "policyrecommendation.tigera.io/orchestrator == k8s",
						NamespaceSelector: "ns1",
						Ports: []numorstring.Port{
							{
								MinPort: 1,
								MaxPort: 99,
							},
							{
								MinPort: 3,
								MaxPort: 3,
							},
							{
								MinPort: 24,
								MaxPort: 35,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "2022-11-30T09:01:38Z",
							"policyrecommendation.tigera.io/namespace":   "ns1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "policyrecommendation.tigera.io/orchestrator == k8s",
						NamespaceSelector: "ns1",
						Ports: []numorstring.Port{
							{
								MinPort: 5,
								MaxPort: 59,
							},
							{
								MinPort: 22,
								MaxPort: 22,
							},
							{
								MinPort: 44,
								MaxPort: 56,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:04:05 PST",
							"policyrecommendation.tigera.io/namespace":   "ns1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "policyrecommendation.tigera.io/orchestrator == k8s",
						NamespaceSelector: "ns2",
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/namespace":   "ns2",
							"policyrecommendation.tigera.io/scope":       "Namespace",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolUDP,
					Source:   v3.EntityRule{},
					Destination: v3.EntityRule{
						Selector:          "policyrecommendation.tigera.io/orchestrator == k8s",
						NamespaceSelector: "ns3",
						Ports: []numorstring.Port{
							{
								MinPort: 8080,
								MaxPort: 8081,
							},
						},
					},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Wed, 29 Nov 2022 14:05:05 PST",
							"policyrecommendation.tigera.io/namespace":   "ns3",
							"policyrecommendation.tigera.io/scope":       "Namespace",
						},
					},
				},
			},
			Ingress: []v3.Rule{
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/orchestrator == k8s",
						NamespaceSelector: "projectcalico.org/name == namespace1",
						Ports: []numorstring.Port{
							{
								MinPort: 443,
								MaxPort: 443,
							},
						},
					},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Mon, 05 Dec 2022 06:35:38 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"projectcalico.org/lastUpdated":              "2022-12-05 06:35:38.338846583 -0800 PST m=+0.048006979",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/orchestrator == k8s",
						NamespaceSelector: "projectcalico.org/name == namespace1",
						Ports: []numorstring.Port{
							{
								MinPort: 443,
								MaxPort: 443,
							},
						},
					},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Mon, 05 Dec 2022 06:35:38 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"projectcalico.org/lastUpdated":              "2022-12-05 06:35:38.338846583 -0800 PST m=+0.048006979",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/orchestrator == k8s",
						NamespaceSelector: "projectcalico.org/name == namespace1",
						Ports: []numorstring.Port{
							{
								MinPort: 443,
								MaxPort: 443,
							},
						},
					},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Mon, 05 Dec 2022 06:35:38 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"projectcalico.org/lastUpdated":              "2022-12-05 06:35:38.338846583 -0800 PST m=+0.048006979",
						},
					},
				},
				{
					Action:   v3.Allow,
					Protocol: &protocolTCP,
					Source: v3.EntityRule{
						Selector:          "projectcalico.org/orchestrator == k8s",
						NamespaceSelector: "projectcalico.org/name == namespace1",
						Ports: []numorstring.Port{
							{
								MinPort: 443,
								MaxPort: 443,
							},
						},
					},
					Destination: v3.EntityRule{},
					Metadata: &v3.RuleMetadata{
						Annotations: map[string]string{
							"policyrecommendation.tigera.io/lastUpdated": "Mon, 05 Dec 2022 06:35:38 PST",
							"policyrecommendation.tigera.io/namespace":   "namespace1",
							"policyrecommendation.tigera.io/scope":       "Namespace",
							"projectcalico.org/lastUpdated":              "2022-12-05 06:35:38.338846583 -0800 PST m=+0.048006979",
						},
					},
				},
			},
		},
	}
)
