// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	wlEpKey1 = model.WorkloadEndpointKey{
		Hostname:       "MyHost",
		OrchestratorID: "ASDF",
		WorkloadID:     "workload1",
		EndpointID:     "endpoint1",
	}
	wlEpKey2 = model.WorkloadEndpointKey{
		Hostname:       "MyHost",
		OrchestratorID: "ASDF",
		WorkloadID:     "workload2",
		EndpointID:     "endpoint2",
	}
)

var _ = Describe("RuleTracePoint", func() {
	var rtp *RuleTracePoint
	Describe("Valid RuleTracePoint", func() {
		Context("Policy", func() {
			BeforeEach(func() {
				var prefix [64]byte
				pfxB := []byte("A|0|policy|tier")
				copy(prefix[:], pfxB)
				rtp, _ = NewRuleTracePoint(prefix, len(pfxB), wlEpKey1)
			})
			It("should have correct tier", func() {
				Expect(rtp.TierID()).To(Equal([]byte("tier")))
			})
			It("should have correct policy", func() {
				Expect(rtp.PolicyID()).To(Equal([]byte("policy")))
			})
			It("should have correct rule", func() {
				Expect(rtp.Rule()).To(Equal([]byte("0")))
			})
			It("should have correct action", func() {
				Expect(rtp.Action).To(Equal(AllowAction))
			})
		})
		Context("Profile", func() {
			BeforeEach(func() {
				var prefix [64]byte
				pfxB := []byte("A|0|policy")
				copy(prefix[:], pfxB)
				rtp, _ = NewRuleTracePoint(prefix, len(pfxB), wlEpKey1)
			})
			It("should have correct tier", func() {
				Expect(rtp.TierID()).To(Equal([]byte("profile")))
			})
			It("should have correct policy", func() {
				Expect(rtp.PolicyID()).To(Equal([]byte("policy")))
			})
			It("should have correct rule", func() {
				Expect(rtp.Rule()).To(Equal([]byte("0")))
			})
			It("should have correct action", func() {
				Expect(rtp.Action).To(Equal(AllowAction))
			})
		})
	})
})

var (
	allowTp0 = &RuleTracePoint{
		prefix:    [64]byte{'A', '|', 'R', '1', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    AllowAction,
		Index:     0,
		EpKey:     wlEpKey1,
	}
	denyTp0 = &RuleTracePoint{
		prefix:    [64]byte{'D', '|', 'R', '2', '|', 'P', '2', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    DenyAction,
		Index:     0,
		EpKey:     wlEpKey1,
	}
	allowTp1 = &RuleTracePoint{
		prefix:    [64]byte{'A', '|', 'R', '1', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    AllowAction,
		Index:     1,
		EpKey:     wlEpKey1,
	}
	denyTp1 = &RuleTracePoint{
		prefix:    [64]byte{'D', '|', 'R', '2', '|', 'P', '2', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    DenyAction,
		Index:     1,
		EpKey:     wlEpKey1,
	}
	allowTp2 = &RuleTracePoint{
		prefix:    [64]byte{'A', '|', 'R', '1', '|', 'P', '2', '|', 'T', '2'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    AllowAction,
		Index:     2,
		EpKey:     wlEpKey1,
	}
	denyTp2 = &RuleTracePoint{
		prefix:    [64]byte{'D', '|', 'R', '2', '|', 'P', '2', '|', 'T', '2'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    DenyAction,
		Index:     2,
		EpKey:     wlEpKey1,
	}
	nextTierTp0 = &RuleTracePoint{
		prefix:    [64]byte{'N', '|', 'R', '3', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    NextTierAction,
		Index:     0,
		EpKey:     wlEpKey1,
	}
	nextTierTp1 = &RuleTracePoint{
		prefix:    [64]byte{'N', '|', 'R', '4', '|', 'P', '2', '|', 'T', '2'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    NextTierAction,
		Index:     1,
		EpKey:     wlEpKey1,
	}
	allowTp11 = &RuleTracePoint{
		prefix:    [64]byte{'A', '|', 'R', '1', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    AllowAction,
		Index:     11,
		EpKey:     wlEpKey1,
	}
	denyTp11 = &RuleTracePoint{
		prefix:    [64]byte{'D', '|', 'R', '1', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    DenyAction,
		Index:     11,
		EpKey:     wlEpKey1,
	}
	allowTp21 = &RuleTracePoint{
		prefix:    [64]byte{'A', '|', 'R', '1', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    AllowAction,
		Index:     21,
		EpKey:     wlEpKey1,
	}
	denyTp21 = &RuleTracePoint{
		prefix:    [64]byte{'A', '|', 'R', '1', '|', 'P', '1', '|', 'T', '1'},
		pfxlen:    10,
		tierIdx:   8,
		policyIdx: 5,
		ruleIdx:   2,
		Action:    DenyAction,
		Index:     21,
		EpKey:     wlEpKey1,
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
			data.AddRuleTracePoint(allowTp0, DirIn)
		})
		It("should have path length equal to 1", func() {
			Expect(len(data.IngressRuleTrace.Path())).To(Equal(1))
		})
		It("should have action set to allow", func() {
			Expect(data.IngressAction()).To(Equal(AllowAction))
		})
		It("should be dirty", func() {
			Expect(data.IsDirty()).To(Equal(true))
		})
		It("should return a conflict for same rule index but different values", func() {
			Expect(data.AddRuleTracePoint(denyTp0, DirIn)).To(Equal(RuleTracePointConflict))
		})
	})

	Describe("RuleTrace conflicts (ingress)", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(allowTp0, DirIn)
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			var dirtyFlag bool
			BeforeEach(func() {
				dirtyFlag = data.IsDirty()
				data.AddRuleTracePoint(denyTp0, DirIn)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(1))
			})
			It("should have action unchanged and set to allow", func() {
				Expect(data.IngressAction()).To(Equal(AllowAction))
			})
			Specify("dirty flag should be unchanged", func() {
				Expect(data.IsDirty()).To(Equal(dirtyFlag))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(denyTp0, DirIn)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(1))
			})
			It("should have action set to deny", func() {
				Expect(data.IngressAction()).To(Equal(DenyAction))
			})
			It("should be dirty", func() {
				Expect(data.IsDirty()).To(Equal(true))
			})
		})
	})
	Describe("RuleTraces with next tier", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(nextTierTp0, DirIn)
		})
		Context("Adding a rule tracepoint with action", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowTp1, DirIn)
			})
			It("should have path length 2", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(2))
			})
			It("should have length unchanged and equal to initial length", func() {
				Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(AllowAction))
			})
		})
		Context("Adding a rule tracepoint with action and index past initial length", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowTp11, DirIn)
			})
			It("should have path length 2", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(2))
			})
			It("should have length twice of initial length", func() {
				Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen * 2))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(AllowAction))
			})
		})
		Context("Adding a rule tracepoint with action and index past double the initial length", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(denyTp21, DirIn)
			})
			It("should have path length 2", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(2))
			})
			It("should have length thrice of initial length", func() {
				Expect(data.IngressRuleTrace.Len()).To(Equal(RuleTraceInitLen * 3))
			})
			It("should have action set to deny", func() {
				Expect(data.IngressAction()).To(Equal(DenyAction))
			})
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowTp0, DirIn)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(1))
			})
			It("should have not have action set", func() {
				Expect(data.IngressAction()).NotTo(Equal(AllowAction))
				Expect(data.IngressAction()).NotTo(Equal(DenyAction))
				Expect(data.IngressAction()).NotTo(Equal(NextTierAction))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(allowTp0, DirIn)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(1))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(AllowAction))
			})
		})
	})
	Describe("RuleTraces with multiple tiers", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(nextTierTp0, DirIn)
			data.AddRuleTracePoint(nextTierTp1, DirIn)
			data.AddRuleTracePoint(allowTp2, DirIn)
		})
		It("should have path length equal to 3", func() {
			Expect(len(data.IngressRuleTrace.Path())).To(Equal(3))
		})
		It("should have have action set to allow", func() {
			Expect(data.IngressAction()).To(Equal(AllowAction))
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(denyTp1, DirIn)
			})
			It("should have path length unchanged and equal to 3", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(3))
			})
			It("should have have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(AllowAction))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(denyTp1, DirIn)
			})
			It("should have path length unchanged and equal to 2", func() {
				Expect(len(data.IngressRuleTrace.Path())).To(Equal(2))
			})
			It("should have action set to allow", func() {
				Expect(data.IngressAction()).To(Equal(DenyAction))
			})
		})
	})

})
