// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var (
	srcPort1 = 54123
	srcPort2 = 54124
)

// Common Tuple definitions
var (
	tuple1 = *NewTuple(localIp1, remoteIp1, proto_tcp, srcPort1, dstPort)
	tuple2 = *NewTuple(localIp1, remoteIp2, proto_tcp, srcPort2, dstPort)
)

// Common RuleIDs definitions
var (
	rule1Allow = RuleIDs{
		Action:    ActionAllow,
		Index:     "0",
		Policy:    "policy1",
		Tier:      "default",
		Direction: RuleDirIngress,
	}
)

// Common MetricUpdate definitions
var (
	// Identical rule/direction connections with differing tuples
	muConn1InboundRule1Allow = &MetricUpdate{
		ruleIDs:      &rule1Allow,
		trafficDir:   TrafficDirInbound,
		tuple:        tuple1,
		packets:      1,
		bytes:        1,
		deltaPackets: 1,
		deltaBytes:   1,
		isConnection: true,
	}
	muConn2InboundRule1Allow = &MetricUpdate{
		ruleIDs:      &rule1Allow,
		trafficDir:   TrafficDirInbound,
		tuple:        tuple2,
		packets:      10,
		bytes:        20,
		deltaPackets: 2,
		deltaBytes:   2,
		isConnection: true,
	}
	muConn1OutboundRule1Allow = &MetricUpdate{
		ruleIDs:      &rule1Allow,
		trafficDir:   TrafficDirOutbound,
		tuple:        tuple1,
		packets:      30,
		bytes:        40,
		deltaPackets: 3,
		deltaBytes:   3,
		isConnection: true,
	}
)

// Common RuleAggregateKey definitions
var (
	keyRule1AllowInbound = RuleAggregateKey{
		ruleIDs:    rule1Allow,
		trafficDir: TrafficDirInbound,
	}
	keyRule1AllowOutbound = RuleAggregateKey{
		ruleIDs:    rule1Allow,
		trafficDir: TrafficDirOutbound,
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

func getMetricGauge(m prometheus.Gauge) int {
	// The get the actual number stored inside a prometheus metric we need to convert
	// into protobuf format which then has publicly available accessors.
	if m == nil {
		return -1
	}
	dtoMetric := &dto.Metric{}
	if err := m.Write(dtoMetric); err != nil {
		panic(err)
	}
	return int(*dtoMetric.Gauge.Value)
}

func expectRuleAggregateKeys(pr *PrometheusReporter, keys []RuleAggregateKey) {
	Eventually(pr.ruleAggStats, expectTimeout).Should(HaveLen(len(keys)))
	Consistently(pr.ruleAggStats, expectTimeout).Should(HaveLen(len(keys)))
	for _, key := range keys {
		Expect(pr.ruleAggStats).To(HaveKey(key))
	}
}

func expectRuleAggregates(
	pr *PrometheusReporter, k RuleAggregateKey,
	expectedPackets int, expectedBytes int, expectedConnections int,
) {
	Eventually(func() int {
		value, ok := pr.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(value.packets)
	}, expectTimeout).Should(Equal(expectedPackets))
	Consistently(func() int {
		value, ok := pr.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(value.packets)
	}, expectTimeout).Should(Equal(expectedPackets))

	Eventually(func() int {
		value, ok := pr.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(value.bytes)
	}, expectTimeout).Should(Equal(expectedBytes))
	Consistently(func() int {
		value, ok := pr.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(value.bytes)
	}, expectTimeout).Should(Equal(expectedBytes))

	Eventually(func() int {
		value, ok := pr.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricGauge(value.numConnections)
	}, expectTimeout).Should(Equal(expectedConnections))
	Consistently(func() int {
		value, ok := pr.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricGauge(value.numConnections)
	}, expectTimeout).Should(Equal(expectedConnections))
}

var _ = Describe("Prometheus Reporter verification", func() {
	var pr *PrometheusReporter
	mt := &mockTime{}
	BeforeEach(func() {
		// Create a PrometheusReporter and start the reporter without starting the HTTP service.
		pr = NewPrometheusReporter(0, retentionTime, "", "", "")
		pr.timeNowFn = mt.getMockTime
		go pr.startReporter()
	})
	AfterEach(func() {
		counterRulePackets.Reset()
		counterRuleBytes.Reset()
	})
	// First set of test handle adding the same rules with two different connections and
	// traffic directions.  Connections should not impact the number of Prometheus metrics,
	// but traffic direction does.
	It("handles the same rule but with two different connections and traffic directions", func() {
		var expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound int
		var expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound int

		By("reporting two separate metrics for same rule and traffic direction, but different connections")
		pr.Report(muConn1InboundRule1Allow)
		expectedPacketsInbound += muConn1InboundRule1Allow.deltaPackets
		expectedBytesInbound += muConn1InboundRule1Allow.deltaPackets
		expectedConnsInbound += 1
		pr.Report(muConn2InboundRule1Allow)
		expectedPacketsInbound += muConn2InboundRule1Allow.deltaPackets
		expectedBytesInbound += muConn2InboundRule1Allow.deltaPackets
		expectedConnsInbound += 1

		By("checking for the correct number of aggregated statistics")
		expectRuleAggregateKeys(pr, []RuleAggregateKey{keyRule1AllowInbound})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pr, keyRule1AllowInbound, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)

		By("reporting one of the same metrics")
		pr.Report(muConn1InboundRule1Allow)
		expectedPacketsInbound += muConn1InboundRule1Allow.deltaPackets
		expectedBytesInbound += muConn1InboundRule1Allow.deltaPackets
		expectedConnsInbound += 0 // connection already registered

		By("checking for the correct number of aggregated statistics")
		expectRuleAggregateKeys(pr, []RuleAggregateKey{keyRule1AllowInbound})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pr, keyRule1AllowInbound, expectedPacketsInbound, expectedBytesInbound, 2)

		By("reporting one of the same metrics but for a different traffic direction")
		pr.Report(muConn1OutboundRule1Allow)
		expectedPacketsOutbound += muConn1OutboundRule1Allow.deltaPackets
		expectedBytesOutbound += muConn1OutboundRule1Allow.deltaPackets
		expectedConnsOutbound += 1

		By("checking for the correct number of aggregated statistics")
		expectRuleAggregateKeys(pr, []RuleAggregateKey{keyRule1AllowInbound, keyRule1AllowOutbound})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pr, keyRule1AllowInbound, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		expectRuleAggregates(pr, keyRule1AllowOutbound, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("expiring one of the metric updates for Rule1 Inbound and one for Outbound")
		pr.Expire(muConn1InboundRule1Allow)
		expectedPacketsInbound += muConn1InboundRule1Allow.deltaPackets
		expectedBytesInbound += muConn1InboundRule1Allow.deltaPackets
		expectedConnsInbound -= 1
		pr.Expire(muConn1OutboundRule1Allow)
		expectedPacketsOutbound += muConn1OutboundRule1Allow.deltaPackets
		expectedBytesOutbound += muConn1OutboundRule1Allow.deltaPackets
		expectedConnsOutbound -= 1
		// Adjust the clock, but not past the retention period, the outbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)

		By("checking for the correct number of aggregated statistics: outbound rule should be present for retention time")
		expectRuleAggregateKeys(pr, []RuleAggregateKey{keyRule1AllowInbound, keyRule1AllowOutbound})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pr, keyRule1AllowInbound, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		expectRuleAggregates(pr, keyRule1AllowOutbound, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("incrementing time by the retention time - outbound rule should be expunged")
		mt.incMockTime(retentionTime)
		expectRuleAggregateKeys(pr, []RuleAggregateKey{keyRule1AllowInbound})

		By("expiring the remaining Rule1 Inbound metric")
		pr.Expire(muConn2InboundRule1Allow)
		expectedPacketsInbound += muConn2InboundRule1Allow.deltaPackets
		expectedBytesInbound += muConn2InboundRule1Allow.deltaPackets
		expectedConnsInbound -= 1
		// Adjust the clock, but not past the retention period, the inbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)

		By("checking for the correct number of aggregated statistics: inbound rule should be present for retention time")
		expectRuleAggregateKeys(pr, []RuleAggregateKey{keyRule1AllowInbound})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pr, keyRule1AllowInbound, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)

		By("incrementing time by the retention time - inbound rule should be expunged")
		mt.incMockTime(retentionTime)
		expectRuleAggregateKeys(pr, []RuleAggregateKey{})
	})
})
