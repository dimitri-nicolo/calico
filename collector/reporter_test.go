// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"math"
	"net"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/libcalico-go/lib/set"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

var (
	srcPort1 = 54123
	srcPort2 = 54124
)

var (
	tuple1 = *NewTuple(localIp1, remoteIp1, proto_tcp, srcPort1, dstPort)
	tuple2 = *NewTuple(localIp1, remoteIp2, proto_tcp, srcPort2, dstPort)
	tuple3 = *NewTuple(localIp2, remoteIp1, proto_tcp, srcPort1, dstPort)
	tuple4 = *NewTuple(localIp2, remoteIp2, proto_tcp, srcPort2, dstPort)
)

var dummyWlEpKey = model.WorkloadEndpointKey{
	Hostname:       "localhost",
	OrchestratorID: "orchestrator",
	WorkloadID:     "workloadid",
	EndpointID:     "epid",
}

var defTierAllowT1 = &RuleTrace{
	path: []*RuleTracePoint{
		{
			prefix:    [64]byte{'A', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '1', '/', 'd', 'e', 'f', 'a', 'u', 'l', 't'},
			pfxlen:    19,
			tierIdx:   12,
			policyIdx: 4,
			ruleIdx:   2,
			Action:    AllowAction,
			Index:     0,
		},
	},
	action: AllowAction,
}

var defTierAllowT2 = &RuleTrace{
	path: []*RuleTracePoint{
		{
			prefix:    [64]byte{'A', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '2', '/', 'd', 'e', 'f', 'a', 'u', 'l', 't'},
			pfxlen:    19,
			tierIdx:   12,
			policyIdx: 4,
			ruleIdx:   2,
			Action:    AllowAction,
			Index:     0,
		},
	},
	action: AllowAction,
}

var defTierDenyT3 = &RuleTrace{
	path: []*RuleTracePoint{
		{
			prefix:    [64]byte{'D', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '3', '/', 'd', 'e', 'f', 'a', 'u', 'l', 't'},
			pfxlen:    19,
			tierIdx:   12,
			policyIdx: 4,
			ruleIdx:   2,
			Action:    DenyAction,
			Index:     0,
		},
	},
	action: DenyAction,
}

var defTierDenyT4 = &RuleTrace{
	path: []*RuleTracePoint{
		{
			prefix:    [64]byte{'D', '/', '0', '/', 'p', 'o', 'l', 'i', 'c', 'y', '4', '/', 'd', 'e', 'f', 'a', 'u', 'l', 't'},
			pfxlen:    19,
			tierIdx:   12,
			policyIdx: 4,
			ruleIdx:   2,
			Action:    DenyAction,
			Index:     0,
		},
	},
	action: DenyAction,
}

var (
	defTierDenyT3Buf = bytes.NewBufferString(defTierDenyT3.ToString())
	defTierDenyT4Buf = bytes.NewBufferString(defTierDenyT4.ToString())
)

var (
	denyPacketTuple1DenyT3 = &MetricUpdate{
		policy:       defTierDenyT3Buf.Bytes(),
		tuple:        tuple1,
		packets:      1,
		bytes:        1,
		deltaPackets: 1,
		deltaBytes:   1,
	}
	denyPacketTuple2DenyT3 = &MetricUpdate{
		policy:       defTierDenyT3Buf.Bytes(),
		tuple:        tuple2,
		packets:      1,
		bytes:        1,
		deltaPackets: 1,
		deltaBytes:   1,
	}
	denyPacketTuple3DenyT4 = &MetricUpdate{
		policy:       defTierDenyT4Buf.Bytes(),
		tuple:        tuple3,
		packets:      1,
		bytes:        1,
		deltaPackets: 1,
		deltaBytes:   1,
	}
)

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

var _ = Describe("Prometheus Reporter", func() {
	var pr *PrometheusReporter
	BeforeEach(func() {
		pr = NewPrometheusReporter(8089, time.Second*time.Duration(30), "", "", "")
		pr.Start()
	})
	AfterEach(func() {
		gaugeDeniedPackets.Reset()
		gaugeDeniedBytes.Reset()
	})
	Describe("Test Report", func() {
		Context("No existing aggregated stats", func() {
			Describe("Same policy and source IP but different connections", func() {
				var key AggregateKey
				var value AggregateValue
				var refs set.Set
				BeforeEach(func() {
					key = AggregateKey{
						srcIP:  net.IP(localIp1[:16]).String(),
						policy: defTierDenyT3.ToString(),
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
						srcIP:  net.IP(localIp1[:16]).String(),
						policy: defTierDenyT3.ToString(),
					}
					key2 = AggregateKey{
						srcIP:  net.IP(localIp2[:16]).String(),
						policy: defTierDenyT4.ToString(),
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
				srcIP:  net.IP(localIp1[:16]).String(),
				policy: defTierDenyT3.ToString(),
			}
			key2 = AggregateKey{
				srcIP:  net.IP(localIp2[:16]).String(),
				policy: defTierDenyT4.ToString(),
			}
			label1 := prometheus.Labels{
				"srcIP":  net.IP(localIp1[:16]).String(),
				"policy": defTierDenyT3.String(),
			}
			label2 := prometheus.Labels{
				"srcIP":  net.IP(localIp2[:16]).String(),
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
				denyPacketTuple1DenyT3.deltaPackets = 0
				denyPacketTuple1DenyT3.deltaBytes = 0
				pr.Expire(denyPacketTuple1DenyT3)
			})
			AfterEach(func() {
				denyPacketTuple1DenyT3.deltaPackets = 1
				denyPacketTuple1DenyT3.deltaBytes = 1
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
				pr.Expire(denyPacketTuple3DenyT4)
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
				Eventually(pr.retainedMetrics).Should(HaveKey(key2))
			})
		})
	})
})
