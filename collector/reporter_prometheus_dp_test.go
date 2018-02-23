// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"math"
	"net"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/set"
)

var dummyWlEpKey = model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "workloadid",
	EndpointID:     "epid",
}

var (
	ingressRulePolicy1Allow = &rules.RuleIDs{
		Action:    rules.ActionAllow,
		Index:     "0",
		Policy:    "policy1",
		Tier:      "default",
		Direction: rules.RuleDirIngress,
	}
	ingressRulePolicy2Allow = &rules.RuleIDs{
		Action:    rules.ActionAllow,
		Index:     "0",
		Policy:    "policy1",
		Tier:      "default",
		Direction: rules.RuleDirIngress,
	}
	ingressRulePolicy3Deny = &rules.RuleIDs{
		Action:    rules.ActionDeny,
		Index:     "0",
		Policy:    "policy3",
		Tier:      "default",
		Direction: rules.RuleDirIngress,
	}
	ingressRulePolicy4Deny = &rules.RuleIDs{
		Action:    rules.ActionDeny,
		Index:     "0",
		Policy:    "policy4",
		Tier:      "default",
		Direction: rules.RuleDirIngress,
	}
)

var (
	denyPacketTuple1DenyT3 = &MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		isConnection: true,
		ruleIDs:      ingressRulePolicy3Deny,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   1,
		},
	}
	denyPacketTuple2DenyT3 = &MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple2,
		isConnection: true,
		ruleIDs:      ingressRulePolicy3Deny,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   1,
		},
	}
	denyPacketTuple3DenyT4 = &MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple3,
		isConnection: true,
		ruleIDs:      ingressRulePolicy4Deny,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   1,
		},
	}
)

func getPolicyName(r *rules.RuleIDs) string {
	return fmt.Sprintf("%s|%s|%s|%s", r.Tier, r.Policy, r.Index, r.Action)
}

func getMetricNumber(m prometheus.Gauge) int {
	// The actual number stored inside a prometheus metric is surprisingly hard to
	// get to.
	if m == nil {
		return -1
	}
	v := reflect.ValueOf(m).Elem()
	valBits := v.FieldByName("valBits")
	return int(math.Float64frombits(valBits.Uint()))
}

var _ = Describe("DP Prometheus Reporter", func() {
	var pr *DPPrometheusReporter
	BeforeEach(func() {
		pr = NewDPPrometheusReporter(0, time.Second*time.Duration(30), "", "", "")
		pr.Start()
	})
	AfterEach(func() {
		gaugeDeniedPackets.Reset()
		gaugeDeniedBytes.Reset()
	})
	Describe("Test Report", func() {
		Context("No existing aggregated stats", func() {
			Describe("Same policy and source IP but different connections", func() {
				var (
					key   AggregateKey
					value AggregateValue
					refs  set.Set
					ok    bool
				)
				BeforeEach(func() {
					key = AggregateKey{
						srcIP:  localIp1,
						policy: getPolicyName(ingressRulePolicy3Deny),
					}
					refs = set.New()
					refs.AddAll([]Tuple{tuple1, tuple2})
					pr.Report(denyPacketTuple1DenyT3)
					pr.Report(denyPacketTuple2DenyT3)
				})
				It("should have 1 aggregated stats entry", func() {
					Eventually(pr.aggStats).Should(HaveLen(1))
				})
				It("should have correct packet and byte counts", func() {
					Eventually(func() int {
						value, ok = pr.aggStats[key]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return -1
						}
						return getMetricNumber(value.packets)
					}).Should(Equal(2))
					Eventually(func() int {
						value, ok = pr.aggStats[key]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return -1
						}
						return getMetricNumber(value.bytes)
					}).Should(Equal(2))
				})
				It("should have correct refs", func() {
					Eventually(func() bool {
						value, ok = pr.aggStats[key]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return false
						}
						return value.refs.Equals(refs)
					}).Should(BeTrue())
				})
			})
			Describe("Different source IPs and Policies", func() {
				var (
					key1, key2     AggregateKey
					value1, value2 AggregateValue
					refs1, refs2   set.Set
					ok             bool
				)
				BeforeEach(func() {
					key1 = AggregateKey{
						srcIP:  localIp1,
						policy: getPolicyName(ingressRulePolicy3Deny),
					}
					key2 = AggregateKey{
						srcIP:  localIp2,
						policy: getPolicyName(ingressRulePolicy4Deny),
					}
					refs1 = set.New()
					refs1.AddAll([]Tuple{tuple1, tuple2})
					refs2 = set.New()
					refs2.AddAll([]Tuple{tuple3})
					pr.Report(denyPacketTuple1DenyT3)
					pr.Report(denyPacketTuple2DenyT3)
					pr.Report(denyPacketTuple3DenyT4)
				})
				It("should have 2 aggregated stats entries", func() {
					Eventually(pr.aggStats).Should(HaveLen(2))
				})
				It("should have correct packet and byte counts", func() {
					Eventually(func() int {
						value1, ok = pr.aggStats[key1]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return -1
						}
						return getMetricNumber(value1.packets)
					}).Should(Equal(2))
					Eventually(func() int {
						value1, ok = pr.aggStats[key1]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return -1
						}
						return getMetricNumber(value1.bytes)
					}).Should(Equal(2))
					Eventually(func() int {
						value2, ok = pr.aggStats[key2]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return -1
						}
						return getMetricNumber(value2.packets)
					}).Should(Equal(1))
					Eventually(func() int {
						value2, ok = pr.aggStats[key2]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return -1
						}
						return getMetricNumber(value2.bytes)
					}).Should(Equal(1))
				})
				It("should have correct refs", func() {
					Eventually(func() bool {
						value1, ok = pr.aggStats[key1]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return false
						}
						return value1.refs.Equals(refs1)
					}).Should(BeTrue())
					Eventually(func() bool {
						value2, ok = pr.aggStats[key2]
						// If we didn't find the key now, we'll
						// not want to look into the value.
						if !ok {
							return false
						}
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
				srcIP:  localIp1,
				policy: getPolicyName(ingressRulePolicy3Deny),
			}
			key2 = AggregateKey{
				srcIP:  localIp2,
				policy: getPolicyName(ingressRulePolicy4Deny),
			}
			label1 := prometheus.Labels{
				"srcIP":  net.IP(localIp1[:16]).String(),
				"policy": getPolicyName(ingressRulePolicy3Deny),
			}
			label2 := prometheus.Labels{
				"srcIP":  net.IP(localIp2[:16]).String(),
				"policy": getPolicyName(ingressRulePolicy4Deny),
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
			var (
				v1, v2       AggregateValue
				refs1, refs2 set.Set
				ok           bool
			)
			BeforeEach(func() {
				refs1 = set.New()
				refs1.AddAll([]Tuple{tuple2})
				refs2 = set.New()
				refs2.AddAll([]Tuple{tuple3})
				denyPacketTuple1DenyT3.inMetric.deltaPackets = 0
				denyPacketTuple1DenyT3.inMetric.deltaBytes = 0
				denyPacketTuple1DenyT3.updateType = UpdateTypeExpire
				pr.Report(denyPacketTuple1DenyT3)
			})
			AfterEach(func() {
				denyPacketTuple1DenyT3.inMetric.deltaPackets = 1
				denyPacketTuple1DenyT3.inMetric.deltaBytes = 1
			})
			It("should have 2 aggregated stats entries", func() {
				Eventually(pr.aggStats).Should(HaveLen(2))
			})
			It("should have correct packet and byte counts", func() {
				Eventually(func() int {
					v1, ok = pr.aggStats[key1]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return -1
					}
					return getMetricNumber(v1.packets)
				}).Should(Equal(3))
				Eventually(func() int {
					v1, ok = pr.aggStats[key1]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return -1
					}
					return getMetricNumber(v1.bytes)
				}).Should(Equal(3))
				Eventually(func() int {
					v2, ok = pr.aggStats[key2]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return -1
					}
					return getMetricNumber(v2.packets)
				}).Should(Equal(2))
				Eventually(func() int {
					v2, ok = pr.aggStats[key2]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return -1
					}
					return getMetricNumber(v2.bytes)
				}).Should(Equal(4))
			})
			It("should have correct refs", func() {
				Eventually(func() bool {
					v1, ok = pr.aggStats[key1]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return false
					}
					return v1.refs.Equals(refs1)
				}).Should(BeTrue())
				Eventually(func() bool {
					v2, ok = pr.aggStats[key2]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return false
					}
					return v2.refs.Equals(refs2)
				}).Should(BeTrue())
			})
		})
		Describe("Delete a entry has only one reference", func() {
			var (
				v1 AggregateValue
				refs1 set.Set
				ok bool
			)
			BeforeEach(func() {
				v1 = pr.aggStats[key1]
				refs1 = set.FromArray([]Tuple{tuple1, tuple2})
				denyPacketTuple3DenyT4.updateType = UpdateTypeExpire
				pr.Report(denyPacketTuple3DenyT4)
			})
			It("should have 2 stats entries", func() {
				Eventually(pr.aggStats).Should(HaveLen(2))
			})
			It("should have correct packet and byte counts", func() {
				Eventually(func() int {
					v1, ok = pr.aggStats[key1]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return -1
					}
					return getMetricNumber(v1.packets)
				}).Should(Equal(3))
				Eventually(func() int {
					v1, ok = pr.aggStats[key1]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return -1
					}
					return getMetricNumber(v1.bytes)
				}).Should(Equal(3))
			})
			It("should have correct refs", func() {
				Eventually(func() bool {
					v1, ok = pr.aggStats[key1]
					// If we didn't find the key now, we'll
					// not want to look into the value.
					if !ok {
						return false
					}
					return v1.refs.Equals(refs1)
				}).Should(BeTrue())
			})
			It("should have the deleted entry as candidate for deletion", func() {
				Eventually(pr.retainedMetrics).Should(HaveKey(key2))
			})
		})
	})
})
