// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package stats_test

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/projectcalico/felix/go/felix/collector/stats"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	allowTp0 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R1",
		Action:   AllowAction,
		Index:    0,
	}
	denyTp0 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P2",
		Rule:     "R2",
		Action:   DenyAction,
		Index:    0,
	}
	allowTp1 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R1",
		Action:   AllowAction,
		Index:    1,
	}
	denyTp1 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P2",
		Rule:     "R2",
		Action:   DenyAction,
		Index:    1,
	}
	allowTp2 = RuleTracePoint{
		TierID:   "T2",
		PolicyID: "P2",
		Rule:     "R1",
		Action:   AllowAction,
		Index:    2,
	}
	denyTp2 = RuleTracePoint{
		TierID:   "T2",
		PolicyID: "P2",
		Rule:     "R2",
		Action:   DenyAction,
		Index:    2,
	}
	nextTierTp0 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R3",
		Action:   NextTierAction,
		Index:    0,
	}
	nextTierTp1 = RuleTracePoint{
		TierID:   "T2",
		PolicyID: "P2",
		Rule:     "R4",
		Action:   NextTierAction,
		Index:    1,
	}
	allowTp11 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R1",
		Action:   AllowAction,
		Index:    11,
	}
	denyTp11 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R1",
		Action:   DenyAction,
		Index:    11,
	}
	allowTp21 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R1",
		Action:   AllowAction,
		Index:    21,
	}
	denyTp21 = RuleTracePoint{
		TierID:   "T1",
		PolicyID: "P1",
		Rule:     "R1",
		Action:   DenyAction,
		Index:    21,
	}
	wlEpKey = model.WorkloadEndpointKey{
		Hostname:       "MyHost",
		OrchestratorID: "ASDF",
		WorkloadID:     "workload",
		EndpointID:     "endpoint",
	}
)

var _ = Describe("Rule Trace", func() {
	var data *Data
	var tuple *Tuple
	var wlEpKey model.WorkloadEndpointKey

	BeforeEach(func() {
		tuple = NewTuple(net.IP("127.0.0,1"), net.IP("127.0.0.1"), 6, 12345, 80)
		data = NewData(*tuple, wlEpKey, 0, 0, 0, 0, time.Duration(10)*time.Second)
	})

	Describe("Data with no rule trace ", func() {
		It("should have length equal to init len", func() {
			Expect(data.RuleTrace.Len()).To(Equal(RuleTraceInitLen))
		})
		It("should be dirty", func() {
			Expect(data.IsDirty()).To(Equal(true))
		})
	})

	Describe("Adding a RuleTracePoint to a Rule Trace", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(allowTp0)
		})
		It("should have path length equal to 1", func() {
			Expect(len(data.RuleTrace.Path())).To(Equal(1))
		})
		It("should have action set to allow", func() {
			Expect(data.Action()).To(Equal(AllowAction))
		})
		It("should be dirty", func() {
			Expect(data.IsDirty()).To(Equal(true))
		})
		It("should return a conflict for same rule index but different values", func() {
			Expect(data.AddRuleTracePoint(denyTp0)).To(Equal(RuleTracePointConflict))
		})
	})

	Describe("RuleTrace conflicts", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(allowTp0)
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			var dirtyFlag bool
			BeforeEach(func() {
				dirtyFlag = data.IsDirty()
				data.AddRuleTracePoint(denyTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(1))
			})
			It("should have action unchanged and set to allow", func() {
				Expect(data.Action()).To(Equal(AllowAction))
			})
			Specify("dirty flag should be unchanged", func() {
				Expect(data.IsDirty()).To(Equal(dirtyFlag))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(denyTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(1))
			})
			It("should have action set to deny", func() {
				Expect(data.Action()).To(Equal(DenyAction))
			})
			It("should be dirty", func() {
				Expect(data.IsDirty()).To(Equal(true))
			})
		})
	})
	Describe("RuleTraces with next tier", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(nextTierTp0)
		})
		Context("Adding a rule tracepoint with action", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowTp1)
			})
			It("should have path length 2", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(2))
			})
			It("should have length unchanged and equal to initial length", func() {
				Expect(data.RuleTrace.Len()).To(Equal(RuleTraceInitLen))
			})
			It("should have action set to allow", func() {
				Expect(data.Action()).To(Equal(AllowAction))
			})
		})
		Context("Adding a rule tracepoint with action and index past initial length", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowTp11)
			})
			It("should have path length 2", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(2))
			})
			It("should have length twice of initial length", func() {
				Expect(data.RuleTrace.Len()).To(Equal(RuleTraceInitLen * 2))
			})
			It("should have action set to allow", func() {
				Expect(data.Action()).To(Equal(AllowAction))
			})
		})
		Context("Adding a rule tracepoint with action and index past double the initial length", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(denyTp21)
			})
			It("should have path length 2", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(2))
			})
			It("should have length thrice of initial length", func() {
				Expect(data.RuleTrace.Len()).To(Equal(RuleTraceInitLen * 3))
			})
			It("should have action set to deny", func() {
				Expect(data.Action()).To(Equal(DenyAction))
			})
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(allowTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(1))
			})
			It("should have not have action set", func() {
				Expect(data.Action()).NotTo(Equal(AllowAction))
				Expect(data.Action()).NotTo(Equal(DenyAction))
				Expect(data.Action()).NotTo(Equal(NextTierAction))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(allowTp0)
			})
			It("should have path length unchanged and equal to 1", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(1))
			})
			It("should have action set to allow", func() {
				Expect(data.Action()).To(Equal(AllowAction))
			})
		})
	})
	Describe("RuleTraces with multiple tiers", func() {
		BeforeEach(func() {
			data.AddRuleTracePoint(nextTierTp0)
			data.AddRuleTracePoint(nextTierTp1)
			data.AddRuleTracePoint(allowTp2)
		})
		It("should have path length equal to 3", func() {
			Expect(len(data.RuleTrace.Path())).To(Equal(3))
		})
		It("should have have action set to allow", func() {
			Expect(data.Action()).To(Equal(AllowAction))
		})
		Context("Adding a rule tracepoint that conflicts", func() {
			BeforeEach(func() {
				data.AddRuleTracePoint(denyTp1)
			})
			It("should have path length unchanged and equal to 3", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(3))
			})
			It("should have have action set to allow", func() {
				Expect(data.Action()).To(Equal(AllowAction))
			})
		})
		Context("Replacing a rule tracepoint that was conflicting", func() {
			BeforeEach(func() {
				data.ReplaceRuleTracePoint(denyTp1)
			})
			It("should have path length unchanged and equal to 2", func() {
				Expect(len(data.RuleTrace.Path())).To(Equal(2))
			})
			It("should have action set to allow", func() {
				Expect(data.Action()).To(Equal(DenyAction))
			})
		})
	})

})
