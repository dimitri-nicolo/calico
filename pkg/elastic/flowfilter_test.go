// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package elastic_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tigera/lma/pkg/rbac"

	"github.com/tigera/lma/pkg/elastic"
)

var _ = Describe("RBAC handling", func() {
	It("handles flow inclusion", func() {

		By("allowing pod flows, disallowing hep flows")
		r := rbac.NewMockFlowHelper(map[string]bool{"pods": true}, false)
		filter := elastic.NewFlowFilterUserRBAC(r)
		flow := &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "wep"}, {"source_namespace", "ns1"}, {"source_name", "a"},
				{"dest_type", "net"}, {"dest_namespace", ""}, {"dest_name", "b"},
			},
		}

		include, err := filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeTrue())

		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "net"}, {"source_namespace", ""}, {"source_name", "a"},
				{"dest_type", "wep"}, {"dest_namespace", "ns1"}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeTrue())

		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "hep"}, {"source_namespace", "ns1"}, {"source_name", "a"},
				{"dest_type", "net"}, {"dest_namespace", "ns1"}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeFalse())

		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "net"}, {"source_namespace", ""}, {"source_name", "a"},
				{"dest_type", "hep"}, {"dest_namespace", ""}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeFalse())

		By("allowing global network sets")
		r = rbac.NewMockFlowHelper(map[string]bool{"globalnetworksets": true}, false)
		filter = elastic.NewFlowFilterUserRBAC(r)
		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "ns"}, {"source_namespace", "ns1"}, {"source_name", "a"},
				{"dest_type", "net"}, {"dest_namespace", ""}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeFalse())

		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "ns"}, {"source_namespace", ""}, {"source_name", "a"},
				{"dest_type", "wep"}, {"dest_namespace", "ns1"}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeTrue())

		By("allowing network sets")
		r = rbac.NewMockFlowHelper(map[string]bool{"networksets": true}, false)
		filter = elastic.NewFlowFilterUserRBAC(r)
		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "ns"}, {"source_namespace", "ns1"}, {"source_name", "a"},
				{"dest_type", "net"}, {"dest_namespace", ""}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeTrue())

		flow = &elastic.CompositeAggregationBucket{
			CompositeAggregationKey: elastic.CompositeAggregationKey{
				{"source_type", "ns"}, {"source_namespace", ""}, {"source_name", "a"},
				{"dest_type", "wep"}, {"dest_namespace", "ns1"}, {"dest_name", "b"},
			},
		}
		include, err = filter.IncludeFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(include).To(BeFalse())
	})

	It("handles policy obfuscation", func() {

		By("allowing all policy types and checking for no obfuscation")
		r := rbac.NewMockFlowHelper(map[string]bool{
			"tiers":                            true,
			"tier.networkpolicies":             true,
			"tier.globalnetworkpolicies":       true,
			"tier.stagednetworkpolicies":       true,
			"tier.stagedglobalnetworkpolicies": true,
			"networkpolicies":                  true,
			"stagedkubernetesnetworkpolicies":  true,
		}, false)
		filter := elastic.NewFlowFilterUserRBAC(r)
		flow := &elastic.CompositeAggregationBucket{
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"policies": {
					Buckets: map[interface{}]int64{
						"0|tier1|tier1.np1|pass":                     1000,
						"1|tier1|staged:tier1.np1|allow":             1000,
						"2|tier2|ns1/tier2.np1|pass":                 1000,
						"3|tier2|ns1/staged:tier2.np1|allow":         1000,
						"4|default|ns1/staged:knp.default.np1|allow": 1000,
						"5|default|ns1/knp.default.np1|allow":        1000,
					},
				},
			},
		}
		err := filter.ModifyFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(flow.AggregatedTerms["policies"].Buckets).To(Equal(map[interface{}]int64{
			"0|tier1|tier1.np1|pass":                     1000,
			"1|tier1|staged:tier1.np1|allow":             1000,
			"2|tier2|ns1/tier2.np1|pass":                 1000,
			"3|tier2|ns1/staged:tier2.np1|allow":         1000,
			"4|default|ns1/staged:knp.default.np1|allow": 1000,
			"5|default|ns1/knp.default.np1|allow":        1000,
		}))

		By("disallowing tier gets - checking staged policies removed and multiple passes contracted")
		r = rbac.NewMockFlowHelper(map[string]bool{
			"tier.networkpolicies":             true,
			"tier.globalnetworkpolicies":       true,
			"tier.stagednetworkpolicies":       true,
			"tier.stagedglobalnetworkpolicies": true,
			"networkpolicies":                  true,
			"stagedkubernetesnetworkpolicies":  true,
		}, false)
		filter = elastic.NewFlowFilterUserRBAC(r)
		flow = &elastic.CompositeAggregationBucket{
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"policies": {
					Buckets: map[interface{}]int64{
						"0|tier1|tier1.np1|pass":                     1000,
						"1|tier1|staged:tier1.np1|allow":             1000,
						"2|tier2|ns1/tier2.np1|pass":                 1000,
						"3|tier2|ns1/staged:tier2.np1|allow":         1000,
						"4|default|ns1/staged:knp.default.np1|allow": 1000,
						"5|default|ns1/knp.default.np1|allow":        1000,
					},
				},
			},
		}
		err = filter.ModifyFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(flow.AggregatedTerms["policies"].Buckets).To(Equal(map[interface{}]int64{
			"0|*|*|pass": 1000,
			"1|default|ns1/staged:knp.default.np1|allow": 1000,
			"2|default|ns1/knp.default.np1|allow":        1000,
		}))

		By("disallowing staged policies - checking staged policies removed")
		r = rbac.NewMockFlowHelper(map[string]bool{
			"tiers":                      true,
			"tier.networkpolicies":       true,
			"tier.globalnetworkpolicies": true,
			"networkpolicies":            true,
		}, false)
		filter = elastic.NewFlowFilterUserRBAC(r)
		flow = &elastic.CompositeAggregationBucket{
			AggregatedTerms: map[string]*elastic.AggregatedTerm{
				"policies": {
					Buckets: map[interface{}]int64{
						"0|tier1|tier1.np1|pass":                     1000,
						"1|tier1|staged:tier1.np1|allow":             1000,
						"2|tier2|ns1/tier2.np1|pass":                 1000,
						"3|tier2|ns1/staged:tier2.np1|allow":         1000,
						"4|default|ns1/staged:knp.default.np1|allow": 1000,
						"5|default|ns1/knp.default.np1|allow":        1000,
					},
				},
			},
		}
		err = filter.ModifyFlow(flow)
		Expect(err).NotTo(HaveOccurred())
		Expect(flow.AggregatedTerms["policies"].Buckets).To(Equal(map[interface{}]int64{
			"0|tier1|tier1.np1|pass":              1000,
			"1|tier2|ns1/tier2.np1|pass":          1000,
			"2|default|ns1/knp.default.np1|allow": 1000,
		}))
	})
})
