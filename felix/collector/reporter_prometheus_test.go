// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/rules"
)

var (
	srcPort1 = 54123
	srcPort2 = 54124
	srcPort3 = 54125
	srcPort4 = 54125
	srcPort5 = 54126
	srcPort6 = 54127
)

// Common Tuple definitions
var (
	tuple1 = *NewTuple(localIp1, remoteIp1, proto_tcp, srcPort1, dstPort)
	tuple2 = *NewTuple(localIp1, remoteIp2, proto_tcp, srcPort2, dstPort)
	tuple3 = *NewTuple(localIp2, remoteIp1, proto_tcp, srcPort1, dstPort)
	tuple4 = *NewTuple(localIp2, remoteIp1, proto_tcp, srcPort4, dstPort)
	tuple5 = *NewTuple(localIp2, remoteIp1, proto_tcp, srcPort5, dstPort)
	tuple6 = *NewTuple(localIp2, remoteIp1, proto_tcp, srcPort6, dstPort)
)

// Common RuleID definitions
var (
	ingressRule1Allow = calc.NewRuleID(
		"default",
		"policy1",
		"",
		0,
		rules.RuleDirIngress,
		rules.RuleActionAllow,
	)

	egressRule2Deny = calc.NewRuleID(
		"default",
		"policy2",
		"",
		0,
		rules.RuleDirEgress,
		rules.RuleActionDeny,
	)

	ingressRule3Pass = calc.NewRuleID(
		"bar",
		"policy3",
		"",
		0,
		rules.RuleDirIngress,
		rules.RuleActionPass,
	)

	egressRule3Pass = calc.NewRuleID(
		"bar",
		"policy3",
		"",
		0,
		rules.RuleDirEgress,
		rules.RuleActionPass,
	)
)

// Common MetricUpdate definitions
var (
	// Metric update without a connection (ingress stats match those of muConn1Rule1AllowUpdate).
	muNoConn1Rule1AllowUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 1,
			deltaBytes:   20,
		},
	}

	// Identical rule/direction connections with differing tuples
	muConn1Rule1AllowUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		isConnection: true,
		hasDenyRule:  false,
		inMetric: MetricValue{
			deltaPackets: 2,
			deltaBytes:   22,
		},
		outMetric: MetricValue{
			deltaPackets: 3,
			deltaBytes:   33,
		},
	}
	muConn1Rule1AllowExpire = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets: 4,
			deltaBytes:   44,
		},
		outMetric: MetricValue{
			deltaPackets: 3,
			deltaBytes:   24,
		},
	}
	muNoConn1Rule2DenyUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{egressRule2Deny},
		hasDenyRule:  true,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 2,
			deltaBytes:   40,
		},
	}
	muNoConn1Rule2DenyExpire = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{egressRule2Deny},
		hasDenyRule:  true,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 0,
			deltaBytes:   0,
		},
	}
	muConn2Rule1AllowUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple2,
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets: 7,
			deltaBytes:   77,
		},
	}
	muConn2Rule1AllowExpire = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple2,
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets: 8,
			deltaBytes:   88,
		},
	}
	muNoConn3Rule2DenyUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple3,
		ruleIDs:      []*calc.RuleID{egressRule2Deny},
		hasDenyRule:  true,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 2,
			deltaBytes:   40,
		},
	}
	muNoConn3Rule2DenyExpire = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple3,
		ruleIDs:      []*calc.RuleID{egressRule2Deny},
		hasDenyRule:  true,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 0,
			deltaBytes:   0,
		},
	}
	muConn1Rule3AllowUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule3Pass, ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets: 2,
			deltaBytes:   22,
		},
		outMetric: MetricValue{
			deltaPackets: 3,
			deltaBytes:   33,
		},
	}
	muConn1Rule3AllowExpire = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule3Pass, ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets: 4,
			deltaBytes:   44,
		},
		outMetric: MetricValue{
			deltaPackets: 3,
			deltaBytes:   24,
		},
	}
	muNoConn1Rule4DenyUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{egressRule3Pass, egressRule2Deny},
		hasDenyRule:  true,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 2,
			deltaBytes:   40,
		},
	}
	muNoConn1Rule4DenyExpire = MetricUpdate{
		updateType:   UpdateTypeExpire,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{egressRule3Pass, egressRule2Deny},
		hasDenyRule:  true,
		isConnection: false,
		inMetric: MetricValue{
			deltaPackets: 0,
			deltaBytes:   0,
		},
	}
	muConn1Rule1HTTPReqAllowUpdate = MetricUpdate{
		updateType:   UpdateTypeReport,
		tuple:        tuple1,
		ruleIDs:      []*calc.RuleID{ingressRule1Allow},
		hasDenyRule:  false,
		isConnection: true,
		inMetric: MetricValue{
			deltaPackets:             200,
			deltaBytes:               22000,
			deltaAllowedHTTPRequests: 20,
			deltaDeniedHTTPRequests:  5,
		},
		outMetric: MetricValue{
			deltaPackets: 300,
			deltaBytes:   33000,
		},
	}
)

// Common RuleAggregateKey definitions
var (
	keyRule1Allow = RuleAggregateKey{
		ruleID: *ingressRule1Allow,
	}

	keyRule2Deny = RuleAggregateKey{
		ruleID: *egressRule2Deny,
	}

	keyRule3Pass = RuleAggregateKey{
		ruleID: *ingressRule3Pass,
	}

	keyEgressRule3Pass = RuleAggregateKey{
		ruleID: *egressRule3Pass,
	}
)

var (
	retentionTime = 500 * time.Millisecond
	expectTimeout = 4 * retentionTime
)

// Mock time helper.
type mockTime struct {
	val int64
}

func (mt *mockTime) getMockTime() time.Duration {
	val := atomic.LoadInt64(&mt.val)
	return time.Duration(val)
}
func (mt *mockTime) incMockTime(inc time.Duration) {
	atomic.AddInt64(&mt.val, int64(inc))
}

func getMetricCount(m prometheus.Counter) int {
	// The get the actual number stored inside a prometheus metric we need to convert
	// into protobuf format which then has publicly available accessors.
	if m == nil {
		return -1
	}
	dtoMetric := &dto.Metric{}
	if err := m.Write(dtoMetric); err != nil {
		panic(err)
	}
	return int(*dtoMetric.Counter.Value)
}

func getDirectionalPackets(dir TrafficDirection, v *RuleAggregateValue) (ret prometheus.Counter) {
	switch dir {
	case TrafficDirInbound:
		ret = v.inPackets
	case TrafficDirOutbound:
		ret = v.outPackets
	}
	return
}

func getDirectionalBytes(dir TrafficDirection, v *RuleAggregateValue) (ret prometheus.Counter) {
	switch dir {
	case TrafficDirInbound:
		ret = v.inBytes
	case TrafficDirOutbound:
		ret = v.outBytes
	}
	return
}

func eventuallyExpectRuleAggregateKeys(pa *PolicyRulesAggregator, keys []RuleAggregateKey) {
	Eventually(pa.ruleAggStats, expectTimeout).Should(HaveLen(len(keys)))
	Consistently(pa.ruleAggStats, expectTimeout).Should(HaveLen(len(keys)))
	for _, key := range keys {
		Expect(pa.ruleAggStats).To(HaveKey(key))
	}
}

func eventuallyExpectRuleAggregates(
	pa *PolicyRulesAggregator, dir TrafficDirection, k RuleAggregateKey,
	expectedPackets int, expectedBytes int, expectedConnections int,
) {
	Eventually(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(getDirectionalPackets(dir, value))
	}, expectTimeout).Should(Equal(expectedPackets))
	Consistently(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(getDirectionalPackets(dir, value))
	}, expectTimeout).Should(Equal(expectedPackets))

	Eventually(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(getDirectionalBytes(dir, value))
	}, expectTimeout).Should(Equal(expectedBytes))
	Consistently(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(getDirectionalBytes(dir, value))
	}, expectTimeout).Should(Equal(expectedBytes))

	if ruleDirToTrafficDir(k.ruleID.Direction) != dir {
		// Don't check connections if rules doesn't match direction.
		return
	}
	Eventually(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(value.numConnections)
	}, expectTimeout).Should(Equal(expectedConnections))
	Consistently(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(value.numConnections)
	}, expectTimeout).Should(Equal(expectedConnections))
}

var _ = Describe("Prometheus Reporter verification", func() {
	var (
		pr *PrometheusReporter
		pa *PolicyRulesAggregator
	)
	mt := &mockTime{}
	BeforeEach(func() {
		// Create a PrometheusReporter and start the reporter without starting the HTTP service.
		pr = NewPrometheusReporter(prometheus.NewRegistry(), 0, retentionTime, "", "", "")
		pa = NewPolicyRulesAggregator(retentionTime, "testHost")
		pr.timeNowFn = mt.getMockTime
		pa.timeNowFn = mt.getMockTime
		pr.AddAggregator(pa)
		go pr.startReporter()
	})
	AfterEach(func() {
		counterRulePackets.Reset()
		counterRuleBytes.Reset()
		counterRuleConns.Reset()
	})
	// First set of test handle adding the same rules with two different connections and
	// traffic directions.  Connections should not impact the number of Prometheus metrics,
	// but traffic direction does.
	It("handles the same rule but with two different connections and traffic directions", func() {
		var expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound int
		var expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound int

		By("reporting two separate metrics for same rule and traffic direction, but different connections")
		pr.Report(muConn1Rule1AllowUpdate)
		expectedPacketsInbound += muConn1Rule1AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule1AllowUpdate.inMetric.deltaBytes
		expectedPacketsOutbound += muConn1Rule1AllowUpdate.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule1AllowUpdate.outMetric.deltaBytes
		expectedConnsInbound += 1
		pr.Report(muConn2Rule1AllowUpdate)
		expectedPacketsInbound += muConn2Rule1AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn2Rule1AllowUpdate.inMetric.deltaBytes
		expectedConnsInbound += 1

		By("checking for the correct number of aggregated statistics")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("reporting one of the same metrics")
		pr.Report(muConn1Rule1AllowUpdate)
		expectedPacketsInbound += muConn1Rule1AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule1AllowUpdate.inMetric.deltaBytes
		expectedPacketsOutbound += muConn1Rule1AllowUpdate.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule1AllowUpdate.outMetric.deltaBytes
		expectedConnsInbound += 0 // connection already registered

		By("checking for the correct number of aggregated statistics")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("expiring one of the metric updates for Rule1 Inbound and one for Outbound")
		pr.Report(muConn1Rule1AllowExpire)
		expectedPacketsInbound += muConn1Rule1AllowExpire.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule1AllowExpire.inMetric.deltaBytes
		expectedPacketsOutbound += muConn1Rule1AllowExpire.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule1AllowExpire.outMetric.deltaBytes
		// Adjust the clock, but not past the retention period, the outbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)

		By("checking for the correct number of aggregated statistics: outbound rule should be present for retention time")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("incrementing time by the retention time - outbound rule should be expunged")
		mt.incMockTime(retentionTime)
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("expiring the remaining Rule1 Inbound metric")
		pr.Report(muConn2Rule1AllowExpire)
		expectedPacketsInbound += muConn2Rule1AllowExpire.inMetric.deltaPackets
		expectedBytesInbound += muConn2Rule1AllowExpire.inMetric.deltaBytes
		// Adjust the clock, but not past the retention period, the inbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)

		By("checking for the correct number of aggregated statistics: inbound rule should be present for retention time")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)

		By("incrementing time by the retention time - inbound rule should be expunged")
		mt.incMockTime(retentionTime)
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{})
	})
	It("handles multiple rules within the metric update and in both directions", func() {
		var expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound, expectedPassConns int
		var expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound int

		By("reporting ingress direction metrics with multiple rules")
		pr.Report(muConn1Rule3AllowUpdate)
		expectedPacketsInbound += muConn1Rule3AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule3AllowUpdate.inMetric.deltaBytes
		expectedPacketsOutbound += muConn1Rule3AllowUpdate.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule3AllowUpdate.outMetric.deltaBytes
		expectedConnsInbound += 1
		By("checking for the correct number of aggregated statistics")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule3Pass, keyRule1Allow})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule3Pass, expectedPacketsInbound, expectedBytesInbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule3Pass, expectedPacketsOutbound, expectedBytesOutbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("expiring one of the metric updates for Rule1 Inbound and one for Outbound")
		pr.Report(muConn1Rule3AllowExpire)
		expectedPacketsInbound += muConn1Rule3AllowExpire.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule3AllowExpire.inMetric.deltaBytes
		expectedPacketsOutbound += muConn1Rule3AllowExpire.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule3AllowExpire.outMetric.deltaBytes
		// Adjust the clock, but not past the retention period, the outbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)

		By("checking for the correct number of aggregated statistics: outbound rule should be present for retention time")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule3Pass, keyRule1Allow})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule3Pass, expectedPacketsInbound, expectedBytesInbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule3Pass, expectedPacketsOutbound, expectedBytesOutbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)
	})
	It("handles multiple rules within the metric update which is a deny", func() {
		var expectedPacketsInbound, expectedBytesInbound, expectedPassConns int
		var expectedPacketsOutbound, expectedBytesOutbound int

		By("reporting ingress direction metrics with multiple rules")
		pr.Report(muNoConn1Rule4DenyUpdate)
		expectedPacketsInbound += muNoConn1Rule4DenyUpdate.inMetric.deltaPackets
		expectedBytesInbound += muNoConn1Rule4DenyUpdate.inMetric.deltaBytes
		expectedPacketsOutbound += muNoConn1Rule4DenyUpdate.outMetric.deltaPackets
		expectedBytesOutbound += muNoConn1Rule4DenyUpdate.outMetric.deltaBytes

		By("checking for the correct number of aggregated statistics")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyEgressRule3Pass, keyRule2Deny})

		By("checking for the correct packet and byte counts")
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyEgressRule3Pass, expectedPacketsInbound, expectedBytesInbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyEgressRule3Pass, expectedPacketsOutbound, expectedBytesOutbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirInbound, keyEgressRule3Pass, expectedPacketsInbound, expectedBytesInbound, expectedPassConns)
		eventuallyExpectRuleAggregates(pa, TrafficDirOutbound, keyEgressRule3Pass, expectedPacketsOutbound, expectedBytesOutbound, expectedPassConns)

		By("expiring the deny metric")
		pr.Report(muNoConn1Rule4DenyExpire)
		// no counters should change.
		mt.incMockTime(retentionTime / 2)
		By("checking for the correct number of aggregated statistics: ")
		eventuallyExpectRuleAggregateKeys(pa, []RuleAggregateKey{keyEgressRule3Pass, keyRule2Deny})
	})
})
