// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"math"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/set"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	srcPort1 = 54123
	srcPort2 = 54124
)

var (
	tuple1 = Tuple{localIp1.String(), remoteIp1.String(), proto_tcp, srcPort1, dstPort}
	tuple2 = Tuple{localIp1.String(), remoteIp2.String(), proto_tcp, srcPort2, dstPort}
	tuple3 = Tuple{localIp2.String(), remoteIp1.String(), proto_tcp, srcPort1, dstPort}
	tuple4 = Tuple{localIp2.String(), remoteIp2.String(), proto_tcp, srcPort2, dstPort}
)

var dummyWlEpKey = model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "workloadid",
	EndpointID:     "epid",
}

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

var (
	allowConn1 = &Data{
		Tuple:            tuple1,
		ctr:              *NewCounter(1, 1),
		ctrReverse:       *NewCounter(1, 1),
		IngressRuleTrace: defTierAllowT1,
		EgressRuleTrace:  defTierAllowT1,
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
		ageTimeout:       time.Duration(10) * time.Second,
		ageTimer:         time.NewTimer(time.Duration(10) * time.Second),
		dirty:            true,
	}
	allowConn2 = &Data{
		Tuple:            tuple2,
		ctr:              *NewCounter(1, 1),
		ctrReverse:       *NewCounter(1, 1),
		IngressRuleTrace: defTierAllowT2,
		EgressRuleTrace:  defTierAllowT2,
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
		ageTimeout:       time.Duration(10) * time.Second,
		ageTimer:         time.NewTimer(time.Duration(10) * time.Second),
		dirty:            true,
	}
	denyPacketTuple1DenyT3 = &Data{
		Tuple:            tuple1,
		ctr:              *NewCounter(1, 1),
		ctrReverse:       *NewCounter(0, 0),
		IngressRuleTrace: defTierDenyT3,
		EgressRuleTrace:  defTierDenyT3,
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
		ageTimeout:       time.Duration(10) * time.Second,
		ageTimer:         time.NewTimer(time.Duration(10) * time.Second),
		dirty:            true,
	}
	denyPacketTuple2DenyT3 = &Data{
		Tuple:            tuple2,
		ctr:              *NewCounter(1, 1),
		ctrReverse:       *NewCounter(0, 0),
		IngressRuleTrace: defTierDenyT3,
		EgressRuleTrace:  defTierDenyT3,
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
		ageTimeout:       time.Duration(10) * time.Second,
		ageTimer:         time.NewTimer(time.Duration(10) * time.Second),
		dirty:            true,
	}
	denyPacketTuple3DenyT4 = &Data{
		Tuple:            tuple3,
		ctr:              *NewCounter(1, 1),
		ctrReverse:       *NewCounter(0, 0),
		IngressRuleTrace: defTierDenyT4,
		EgressRuleTrace:  defTierDenyT4,
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
		ageTimeout:       time.Duration(10) * time.Second,
		ageTimer:         time.NewTimer(time.Duration(10) * time.Second),
		dirty:            true,
	}
)

func getMetricNumber(m prometheus.Gauge) int {
	// The actual number stored inside a prometheus metric is surprisingly hard to
	// get to.
	v := reflect.ValueOf(m).Elem()
	valBits := v.FieldByName("valBits")
	return int(math.Float64frombits(valBits.Uint()))
}

var _ = Describe("Prometheus Reporter", func() {
	var pr *PrometheusReporter
	BeforeEach(func() {
		pr = NewPrometheusReporter()
		pr.Start()
	})
	AfterEach(func() {
		gaugeDeniedPackets.Reset()
		gaugeDeniedBytes.Reset()
	})
	Describe("Test Report", func() {
		Context("No existing aggregated stats", func() {
			Describe("Should reject entries that are not deny-s", func() {
				BeforeEach(func() {
					pr.Report(*allowConn1)
					pr.Report(*allowConn2)
				})
				It("should have no aggregated stats entries", func() {
					Eventually(pr.aggStats).Should(HaveLen(0))
				})
			})
			Describe("Should reject entries that are not dirty", func() {
				BeforeEach(func() {
					allowConn1.dirty = false
					allowConn2.dirty = false
					pr.Report(*allowConn1)
					pr.Report(*allowConn2)
				})
				It("should have no aggregated stats entries", func() {
					Eventually(pr.aggStats).Should(HaveLen(0))
				})
				AfterEach(func() {
					allowConn1.dirty = true
					allowConn2.dirty = true
				})
			})
			Describe("Same policy and source IP but different connections", func() {
				var key AggregateKey
				var value AggregateValue
				var refs set.Set
				BeforeEach(func() {
					key = AggregateKey{
						srcIP:  localIp1.String(),
						policy: defTierDenyT3.ToString(),
					}
					refs = set.New()
					refs.AddAll([]Tuple{tuple1, tuple2})
					pr.Report(*denyPacketTuple1DenyT3)
					pr.Report(*denyPacketTuple2DenyT3)
				})
				It("should have 1 aggregated stats entry", func() {
					Eventually(pr.aggStats).Should(HaveLen(1))
				})
				It("should have correct packet and byte counts", func() {
					Eventually(func() int {
						value, _ = pr.aggStats[key]
						return getMetricNumber(value.packets)
					}).Should(Equal(2))
					Eventually(func() int {
						value, _ = pr.aggStats[key]
						return getMetricNumber(value.bytes)
					}).Should(Equal(2))
				})
				It("should have correct refs", func() {
					Eventually(func() bool {
						value, _ = pr.aggStats[key]
						return value.refs.Equals(refs)
					}).Should(BeTrue())
				})
			})
			Describe("Different source IPs and Policies", func() {
				var key1, key2 AggregateKey
				var value1, value2 AggregateValue
				var refs1, refs2 set.Set
				BeforeEach(func() {
					key1 = AggregateKey{
						srcIP:  localIp1.String(),
						policy: defTierDenyT3.ToString(),
					}
					key2 = AggregateKey{
						srcIP:  localIp2.String(),
						policy: defTierDenyT4.ToString(),
					}
					refs1 = set.New()
					refs1.AddAll([]Tuple{tuple1, tuple2})
					refs2 = set.New()
					refs2.AddAll([]Tuple{tuple3})
					pr.Report(*denyPacketTuple1DenyT3)
					pr.Report(*denyPacketTuple2DenyT3)
					pr.Report(*denyPacketTuple3DenyT4)
				})
				It("should have 2 aggregated stats entries", func() {
					Eventually(pr.aggStats).Should(HaveLen(2))
				})
				It("should have correct packet and byte counts", func() {
					Eventually(func() int {
						value1, _ = pr.aggStats[key1]
						return getMetricNumber(value1.packets)
					}).Should(Equal(2))
					Eventually(func() int {
						value1, _ = pr.aggStats[key1]
						return getMetricNumber(value1.bytes)
					}).Should(Equal(2))
					Eventually(func() int {
						value2, _ = pr.aggStats[key2]
						return getMetricNumber(value2.packets)
					}).Should(Equal(1))
					Eventually(func() int {
						value2, _ = pr.aggStats[key2]
						return getMetricNumber(value2.bytes)
					}).Should(Equal(1))
				})
				It("should have correct refs", func() {
					Eventually(func() bool {
						value1, _ = pr.aggStats[key1]
						return value1.refs.Equals(refs1)
					}).Should(BeTrue())
					Eventually(func() bool {
						value2, _ = pr.aggStats[key2]
						return value2.refs.Equals(refs2)
					}).Should(BeTrue())
				})
			})
		})
	})
	Describe("Test Expire", func() {
		var key1, key2 AggregateKey
		var value1, value2 AggregateValue
		BeforeEach(func() {
			key1 = AggregateKey{
				srcIP:  localIp1.String(),
				policy: defTierDenyT3.ToString(),
			}
			key2 = AggregateKey{
				srcIP:  localIp2.String(),
				policy: defTierDenyT4.ToString(),
			}
			label1 := prometheus.Labels{
				"srcIP":  localIp1.String(),
				"policy": defTierDenyT3.String(),
			}
			label2 := prometheus.Labels{
				"srcIP":  localIp2.String(),
				"policy": defTierDenyT4.String(),
			}
			value1 = AggregateValue{
				packets: gaugeDeniedPackets.With(label1),
				bytes:   gaugeDeniedBytes.With(label1),
				refs:    set.FromArray([]Tuple{tuple1, tuple2}),
			}
			value1.packets.Set(3)
			value1.bytes.Set(3)
			value2 = AggregateValue{
				packets: gaugeDeniedPackets.With(label2),
				bytes:   gaugeDeniedBytes.With(label2),
				refs:    set.FromArray([]Tuple{tuple3}),
			}
			value2.packets.Set(2)
			value2.bytes.Set(4)
			pr.aggStats[key1] = value1
			pr.aggStats[key2] = value2
		})
		Describe("Delete a entry has more than one reference", func() {
			var v1, v2 AggregateValue
			var refs1, refs2 set.Set
			BeforeEach(func() {
				refs1 = set.New()
				refs1.AddAll([]Tuple{tuple2})
				refs2 = set.New()
				refs2.AddAll([]Tuple{tuple3})
				denyPacketTuple1DenyT3.dirty = false
				pr.Expire(*denyPacketTuple1DenyT3)
			})
			AfterEach(func() {
				denyPacketTuple1DenyT3.dirty = true
			})
			It("should have 2 aggregated stats entries", func() {
				Eventually(pr.aggStats).Should(HaveLen(2))
			})
			It("should have correct packet and byte counts", func() {
				Eventually(func() int {
					v1, _ = pr.aggStats[key1]
					return getMetricNumber(v1.packets)
				}).Should(Equal(3))
				Eventually(func() int {
					v1, _ = pr.aggStats[key1]
					return getMetricNumber(v1.bytes)
				}).Should(Equal(3))
				Eventually(func() int {
					v2, _ = pr.aggStats[key2]
					return getMetricNumber(v2.packets)
				}).Should(Equal(2))
				Eventually(func() int {
					v2, _ = pr.aggStats[key2]
					return getMetricNumber(v2.bytes)
				}).Should(Equal(4))
			})
			It("should have correct refs", func() {
				Eventually(func() bool {
					v1, _ = pr.aggStats[key1]
					return v1.refs.Equals(refs1)
				}).Should(BeTrue())
				Eventually(func() bool {
					v2, _ = pr.aggStats[key2]
					return v2.refs.Equals(refs2)
				}).Should(BeTrue())
			})
		})
		Describe("Delete a entry has only one reference", func() {
			var v1 AggregateValue
			var refs1 set.Set
			BeforeEach(func() {
				v1 = pr.aggStats[key1]
				refs1 = set.FromArray([]Tuple{tuple1, tuple2})
				pr.Expire(*denyPacketTuple3DenyT4)
			})
			It("should have 2 stats entries", func() {
				Eventually(pr.aggStats).Should(HaveLen(2))
			})
			It("should have correct packet and byte counts", func() {
				Eventually(func() int {
					v1, _ = pr.aggStats[key1]
					return getMetricNumber(v1.packets)
				}).Should(Equal(3))
				Eventually(func() int {
					v1, _ = pr.aggStats[key1]
					return getMetricNumber(v1.bytes)
				}).Should(Equal(3))
			})
			It("should have correct refs", func() {
				Eventually(func() bool {
					v1, _ = pr.aggStats[key1]
					return v1.refs.Equals(refs1)
				}).Should(BeTrue())
			})
			It("should have the deleted entry as candidate for deletion", func() {
				Eventually(pr.deleteCandidates).Should(HaveKey(key2))
			})
		})
	})
})
