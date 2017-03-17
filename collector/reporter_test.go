// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	srcPort1 = 54123
	srcPort2 = 54124
)

var (
	tuple1 = Tuple{string(localIp1), string(remoteIp1), proto_tcp, srcPort1, dstPort}
	tuple2 = Tuple{string(localIp1), string(remoteIp2), proto_tcp, srcPort2, dstPort}
	tuple3 = Tuple{string(localIp2), string(remoteIp1), proto_tcp, srcPort1, dstPort}
	tuple4 = Tuple{string(localIp2), string(remoteIp2), proto_tcp, srcPort2, dstPort}
)

var defTierAllowT1 = &RuleTrace{
	path: []RuleTracePoint{
		{
			TierID:   "default",
			PolicyID: "policy1",
			Rule:     "0",
			Action:   AllowAction,
			Index:    0,
		},
	},
	action: AllowAction,
}

var defTierAllowT2 = &RuleTrace{
	path: []RuleTracePoint{
		{
			TierID:   "default",
			PolicyID: "policy2",
			Rule:     "0",
			Action:   AllowAction,
			Index:    0,
		},
	},
	action: AllowAction,
}

var defTierDenyT3 = &RuleTrace{
	path: []RuleTracePoint{
		{
			TierID:   "default",
			PolicyID: "policy3",
			Rule:     "0",
			Action:   DenyAction,
			Index:    0,
		},
	},
	action: DenyAction,
}

var defTierDenyT4 = &RuleTrace{
	path: []RuleTracePoint{
		{
			TierID:   "default",
			PolicyID: "policy4",
			Rule:     "0",
			Action:   DenyAction,
			Index:    0,
		},
	},
	action: DenyAction,
}

var dummyWlEpKey = model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "workloadid",
	EndpointID:     "epid",
}

var (
	allowConn1 = &Data{
		Tuple:      tuple1,
		WlEpKey:    dummyWlEpKey,
		ctrIn:      Counter{packets: 1, bytes: 1},
		ctrOut:     Counter{packets: 1, bytes: 1},
		RuleTrace:  defTierAllowT1,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
		ageTimeout: time.Duration(10) * time.Second,
		ageTimer:   time.NewTimer(time.Duration(10) * time.Second),
		dirty:      true,
	}
	allowConn2 = &Data{
		Tuple:      tuple2,
		WlEpKey:    dummyWlEpKey,
		ctrIn:      Counter{packets: 1, bytes: 1},
		ctrOut:     Counter{packets: 1, bytes: 1},
		RuleTrace:  defTierAllowT2,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
		ageTimeout: time.Duration(10) * time.Second,
		ageTimer:   time.NewTimer(time.Duration(10) * time.Second),
		dirty:      true,
	}
	denyPacketTuple1DenyT3 = &Data{
		Tuple:      tuple1,
		WlEpKey:    dummyWlEpKey,
		ctrIn:      Counter{packets: 0, bytes: 0},
		ctrOut:     Counter{packets: 1, bytes: 1},
		RuleTrace:  defTierDenyT3,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
		ageTimeout: time.Duration(10) * time.Second,
		ageTimer:   time.NewTimer(time.Duration(10) * time.Second),
		dirty:      true,
	}
	denyPacketTuple2DenyT3 = &Data{
		Tuple:      tuple2,
		WlEpKey:    dummyWlEpKey,
		ctrIn:      Counter{packets: 0, bytes: 0},
		ctrOut:     Counter{packets: 1, bytes: 1},
		RuleTrace:  defTierDenyT3,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
		ageTimeout: time.Duration(10) * time.Second,
		ageTimer:   time.NewTimer(time.Duration(10) * time.Second),
		dirty:      true,
	}
	denyPacketTuple3DenyT4 = &Data{
		Tuple:      tuple3,
		WlEpKey:    dummyWlEpKey,
		ctrIn:      Counter{packets: 0, bytes: 0},
		ctrOut:     Counter{packets: 1, bytes: 1},
		RuleTrace:  defTierDenyT4,
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
		ageTimeout: time.Duration(10) * time.Second,
		ageTimer:   time.NewTimer(time.Duration(10) * time.Second),
		dirty:      true,
	}
)

var _ = Describe("Prometheus Reporter", func() {
	var pr *PrometheusReporter
	BeforeEach(func() {
		pr = NewPrometheusReporter()
	})
	Describe("Test Update", func() {
		Context("No existing aggregated stats", func() {
			Describe("Should reject entries that are not deny-s", func() {
				BeforeEach(func() {
					pr.Update(*allowConn1)
					pr.Update(*allowConn2)
				})
				It("should have no aggregated stats entries", func() {
					Expect(pr.aggStats).Should(HaveLen(0))
				})
			})
			Describe("Should reject entries that are not dirty", func() {
				BeforeEach(func() {
					allowConn1.dirty = false
					allowConn2.dirty = false
					pr.Update(*allowConn1)
					pr.Update(*allowConn2)
				})
				It("should have no aggregated stats entries", func() {
					Expect(pr.aggStats).Should(HaveLen(0))
				})
				AfterEach(func() {
					allowConn1.dirty = true
					allowConn2.dirty = true
				})
			})
			Describe("Same policy and source IP but different connections", func() {
				var key AggregateKey
				var value AggregateValue
				BeforeEach(func() {
					pr.Update(*denyPacketTuple1DenyT3)
					pr.Update(*denyPacketTuple2DenyT3)
					key = AggregateKey{
						srcIP:  string(localIp1),
						policy: defTierDenyT3.ToString(),
					}
					value, _ = pr.aggStats[key]
				})
				It("should have 1 aggregated stats entry", func() {
					Expect(pr.aggStats).Should(HaveLen(1))
				})
				It("should have packet count 2 and byte count 2", func() {
					Expect(value.bytes).To(Equal(2))
					Expect(value.packets).To(Equal(2))
				})
				It("should have ref count 2", func() {
					Expect(value.refs).To(Equal(2))
				})
			})
			Describe("Different source IPs and Policies", func() {
				var key1, key2 AggregateKey
				var value1, value2 AggregateValue
				BeforeEach(func() {
					pr.Update(*denyPacketTuple1DenyT3)
					pr.Update(*denyPacketTuple2DenyT3)
					pr.Update(*denyPacketTuple3DenyT4)
					key1 = AggregateKey{
						srcIP:  string(localIp1),
						policy: defTierDenyT3.ToString(),
					}
					key2 = AggregateKey{
						srcIP:  string(localIp2),
						policy: defTierDenyT4.ToString(),
					}
					value1, _ = pr.aggStats[key1]
					value2, _ = pr.aggStats[key2]
				})
				It("should have 2 aggregated stats entries", func() {
					Expect(pr.aggStats).Should(HaveLen(2))
				})
				It("should have correct packet and byte counts", func() {
					Expect(value1.bytes).To(Equal(2))
					Expect(value1.packets).To(Equal(2))
					Expect(value2.bytes).To(Equal(1))
					Expect(value2.packets).To(Equal(1))
				})
				It("should have correct ref counts", func() {
					Expect(value1.refs).To(Equal(2))
					Expect(value2.refs).To(Equal(1))
				})
			})
		})
	})
	Describe("Test Delete", func() {
		var key1, key2 AggregateKey
		var value1, value2 AggregateValue
		BeforeEach(func() {
			key1 = AggregateKey{
				srcIP:  string(localIp1),
				policy: defTierDenyT3.ToString(),
			}
			key2 = AggregateKey{
				srcIP:  string(localIp2),
				policy: defTierDenyT4.ToString(),
			}
			value1 = AggregateValue{
				Counter: Counter{
					packets: 3,
					bytes:   3,
				},
				refs: 2,
			}
			value2 = AggregateValue{
				Counter: Counter{
					packets: 2,
					bytes:   4,
				},
				refs: 1,
			}
			pr.aggStats[key1] = value1
			pr.aggStats[key2] = value2
		})
		Describe("Delete a entry has more than one reference", func() {
			var v1, v2 AggregateValue
			BeforeEach(func() {
				pr.Delete(*denyPacketTuple1DenyT3)
				v1 = pr.aggStats[key1]
				v2 = pr.aggStats[key2]
			})
			It("should have 2 aggregated stats entries", func() {
				Expect(pr.aggStats).Should(HaveLen(2))
			})
			It("should have correct packet and byte counts", func() {
				Expect(v1.bytes).To(Equal(3))
				Expect(v1.packets).To(Equal(3))
				Expect(v2.bytes).To(Equal(4))
				Expect(v2.packets).To(Equal(2))
			})
			It("should have correct ref counts", func() {
				Expect(v1.refs).To(Equal(1))
				Expect(v2.refs).To(Equal(1))
			})
		})
		Describe("Delete a entry has only one reference", func() {
			var v1 AggregateValue
			BeforeEach(func() {
				pr.Delete(*denyPacketTuple3DenyT4)
				v1 = pr.aggStats[key1]
			})
			It("should have 1 aggregated stats entries", func() {
				Expect(pr.aggStats).Should(HaveLen(1))
			})
			It("should have correct packet and byte counts", func() {
				Expect(v1.bytes).To(Equal(3))
				Expect(v1.packets).To(Equal(3))
			})
			It("should have correct ref counts", func() {
				Expect(v1.refs).To(Equal(2))
			})
			It("should not have the deleted key", func() {
				Expect(pr.aggStats).ShouldNot(HaveKey(key2))
			})
		})
	})
})
