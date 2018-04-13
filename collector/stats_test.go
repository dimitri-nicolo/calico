// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/backend/model"

	"github.com/projectcalico/felix/lookup"
	"github.com/projectcalico/felix/rules"
)

var (
	wlEpKey1 = model.WorkloadEndpointKey{
		Hostname:       "MyHost",
		OrchestratorID: "ASDF",
		WorkloadID:     "workload1",
		EndpointID:     "endpoint1",
	}
)

var (
	allowIngressTp0 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionAllow,
			Index:     1,
			IndexStr:  "1",
			Name:      "P1",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        0,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	denyIngressTp0 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionDeny,
			Index:     2,
			IndexStr:  "2",
			Name:      "P2",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        0,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	allowIngressTp1 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionAllow,
			Index:     1,
			IndexStr:  "1",
			Name:      "P1",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        1,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	denyIngressTp1 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionDeny,
			Index:     2,
			IndexStr:  "2",
			Name:      "P2",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        1,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	allowIngressTp2 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionAllow,
			Index:     1,
			IndexStr:  "1",
			Name:      "P2",
			Tier:      "T2",
			Direction: rules.RuleDirIngress,
		},
		Index:        2,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	nextTierIngressTp0 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionNextTier,
			Index:     3,
			IndexStr:  "3",
			Name:      "P1",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        0,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	nextTierIngressTp1 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionNextTier,
			Index:     4,
			IndexStr:  "4",
			Name:      "P2",
			Tier:      "T2",
			Direction: rules.RuleDirIngress,
		},
		Index:        1,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	allowIngressTp11 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionAllow,
			Index:     1,
			IndexStr:  "1",
			Name:      "P1",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        11,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	denyIngressTp21 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionDeny,
			Index:     1,
			IndexStr:  "1",
			Name:      "P1",
			Tier:      "T1",
			Direction: rules.RuleDirIngress,
		},
		Index:        21,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}

	nextTierEgressTp0 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionNextTier,
			Index:     2,
			IndexStr:  "2",
			Name:      "P4",
			Tier:      "T1",
			Direction: rules.RuleDirEgress,
		},
		Index:        0,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
	allowEgressTp2 = &RuleTracePoint{
		RuleID: &lookup.RuleID{
			Action:    rules.RuleActionAllow,
			Index:     3,
			IndexStr:  "3",
			Name:      "P3",
			Tier:      "T2",
			Direction: rules.RuleDirEgress,
		},
		Index:        2,
		EndpointName: "WEP(ASDF/workload1/endpoint1)",
	}
)

var _ = Describe("Tuple", func() {
	var tuple *Tuple
	Describe("Parse Ipv4 Tuple", func() {
		BeforeEach(func() {
			var src, dst [16]byte
			copy(src[:], net.ParseIP("127.0.0.1").To16())
			copy(dst[:], net.ParseIP("127.1.1.1").To16())
			tuple = NewTuple(src, dst, 6, 12345, 80)
		})
		It("should parse correctly", func() {
			Expect(net.IP(tuple.src[:16]).String()).To(Equal("127.0.0.1"))
			Expect(net.IP(tuple.dst[:16]).String()).To(Equal("127.1.1.1"))
		})
	})
})

var _ = Describe("Rule Trace", func() {
	var data *Data
	var tuple *Tuple

	BeforeEach(func() {
		var src, dst [16]byte
		copy(src[:], net.ParseIP("127.0.0.1").To16())
		copy(dst[:], net.ParseIP("127.1.1.1").To16())
		tuple = NewTuple(src, dst, 6, 12345, 80)
		data = NewData(*tuple, time.Duration(10)*time.Second)
	})

	Describe("Data with no ingress or egress rule trace ", func() {
		It("should have length equal to init len", func() {
			Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen))
			Expect(data.EgressRuleTrace.Len()).To(Equal(RuleTraceInitLen))
		})
		It("should be dirty", func() {
			Expect(data.IsDirty()).To(Equal(true))
		})
	})

	Describe("Adding a RuleTracePoint to the Ingress Rule Trace", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(allowIngressTp0)
		})
		It("should have path length equal to 1", func() {
			Expect(data.IngressRuleTrace.Path()).To(HaveLen(1))
		})
		It("should have action set to allow", func() {
			Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
		})
		It("should be dirty", func() {
			Expect(data.IsDirty()).To(Equal(true))
		})
		It("should return a conflict for same rule Index but different values", func() {
			Expect(data.AddRuleTracePoint(denyIngressTp0)).To(Equal(RuleTracePointConflict))
		})
	})

	Describe("RuleTrace conflicts (ingress)", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(allowIngressTp0)
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			var dirtyFlag bool
			BeforeEach(func() {
				dirtyFlag = data.IsDirty()
				data.AddRuleTracePoint(denyIngressTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(data.IngressRuleTrace.Path()).To(HaveLen(1))
			})
			It("should have action unchanged and set to allow", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
			})
			Specify("dirty flag should be unchanged", func() {
				Expect(data.IsDirty()).To(Equal(dirtyFlag))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(denyIngressTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(data.IngressRuleTrace.Path()).To(HaveLen(1))
			})
			It("should have action set to deny", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionDeny))
			})
			It("should be dirty", func() {
				Expect(data.IsDirty()).To(Equal(true))
			})
		})
	})
	Describe("RuleTraces with next Tier", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(nextTierIngressTp0)
		})
		Context("Adding a rule tracepoint with action", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowIngressTp1)
			})
			It("should have path length 2", func() {
				Expect(data.IngressRuleTrace.Path()).To(HaveLen(2))
			})
			It("should have length unchanged and equal to initial length", func() {
				Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
			})
		})
		Context("Adding a rule tracepoint with action and Index past initial length", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowIngressTp11)
			})
			It("should have path length 2", func() {
				Expect(data.IngressRuleTrace.Path()).To(HaveLen(2))
			})
			It("should have length twice of initial length", func() {
				Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen * 2))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
			})
		})
		Context("Adding a rule tracepoint with action and Index past double the initial length", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(denyIngressTp21)
			})
			It("should have path length 2", func() {
				Expect(data.IngressRuleTrace.Path()).To(HaveLen(2))
			})
			It("should have length thrice of initial length", func() {
				Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen * 3))
			})
			It("should have action set to deny", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionDeny))
			})
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowIngressTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(data.IngressRuleTrace.Path()).To(HaveLen(1))
			})
			It("should have not have action set", func() {
				Expect(data.IngressAction()).NotTo(Equal(rules.RuleActionAllow))
				Expect(data.IngressAction()).NotTo(Equal(rules.RuleActionDeny))
				Expect(data.IngressAction()).NotTo(Equal(rules.RuleActionNextTier))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(allowIngressTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(1))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
			})
		})
	})
	Describe("RuleTraces with multiple tiers", func() {
		BeforeEach(func() {
			// Ingress
			err := data.AddRuleTracePoint(nextTierIngressTp0)
			Expect(err).NotTo(HaveOccurred())
			err = data.AddRuleTracePoint(nextTierIngressTp1)
			Expect(err).NotTo(HaveOccurred())
			err = data.AddRuleTracePoint(allowIngressTp2)
			Expect(err).NotTo(HaveOccurred())
			// Egress
			err = data.AddRuleTracePoint(nextTierEgressTp0)
			Expect(err).NotTo(HaveOccurred())
			err = data.AddRuleTracePoint(allowEgressTp2)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should have ingress path length equal to 3", func() {
			Expect(data.IngressRuleTrace.Path()).To(HaveLen(3))
		})
		It("should have egress path length equal to 2", func() {
			Expect(data.EgressRuleTrace.Path()).To(HaveLen(2))
		})
		It("should have have ingress action set to allow", func() {
			Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
		})
		It("should have have egress action set to allow", func() {
			Expect(data.EgressAction()).To(Equal(rules.RuleActionAllow))
		})
		Context("Adding an ingress rule tracepoint that conflicts", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(denyIngressTp1)
			})
			It("should have path length unchanged and equal to 3", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(3))
			})
			It("should have have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionAllow))
			})
		})
		Context("Replacing an ingress rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(denyIngressTp1)
			})
			It("should have path length unchanged and equal to 2", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(2))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(rules.RuleActionDeny))
			})
		})
	})

})
