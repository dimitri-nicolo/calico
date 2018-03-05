// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/felix/rules"
)

func expectRuleAggregateKeys(pa *PolicyRulesAggregator, keys []RuleAggregateKey) {
	Expect(pa.ruleAggStats).To(HaveLen(len(keys)))
	for _, key := range keys {
		Expect(pa.ruleAggStats).To(HaveKey(key))
	}
}

func expectRuleAggregates(
	pa *PolicyRulesAggregator, dir rules.TrafficDirection, k RuleAggregateKey,
	expectedPackets int, expectedBytes int, expectedConnections int,
) {
	Expect(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(getDirectionalPackets(dir, value))
	}()).To(Equal(expectedPackets))

	Expect(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricCount(getDirectionalBytes(dir, value))
	}()).To(Equal(expectedBytes))

	if ruleDirToTrafficDir[k.ruleIDs.Direction] != dir {
		// Don't check connections if rules doesn't match direction.
		return
	}
	Expect(func() int {
		value, ok := pa.ruleAggStats[k]
		if !ok {
			return -1
		}
		return getMetricGauge(value.numConnections)
	}()).To(Equal(expectedConnections))
}

var _ = Describe("Prometheus Policy Rules Aggregator verification", func() {
	var pa *PolicyRulesAggregator
	mt := &mockTime{}
	registry := prometheus.NewRegistry()
	BeforeEach(func() {
		// Create a PolicyRulesAggregator
		pa = NewPolicyRulesAggregator(retentionTime)
		pa.timeNowFn = mt.getMockTime
		pa.RegisterMetrics(registry)
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
		pa.OnUpdate(muConn1Rule1AllowUpdate)
		expectedPacketsInbound += muConn1Rule1AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule1AllowUpdate.inMetric.deltaPackets
		expectedPacketsOutbound += muConn1Rule1AllowUpdate.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule1AllowUpdate.outMetric.deltaBytes
		expectedConnsInbound += 1
		pa.OnUpdate(muConn2Rule1AllowUpdate)
		expectedPacketsInbound += muConn2Rule1AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn2Rule1AllowUpdate.inMetric.deltaPackets
		expectedConnsInbound += 1

		By("checking for the correct number of aggregated statistics")
		expectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pa, rules.TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		expectRuleAggregates(pa, rules.TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("reporting one of the same metrics")
		pa.OnUpdate(muConn1Rule1AllowUpdate)
		expectedPacketsInbound += muConn1Rule1AllowUpdate.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule1AllowUpdate.inMetric.deltaPackets
		expectedPacketsOutbound += muConn1Rule1AllowUpdate.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule1AllowUpdate.outMetric.deltaBytes
		expectedConnsInbound += 0 // connection already registered

		By("checking for the correct number of aggregated statistics")
		expectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pa, rules.TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		expectRuleAggregates(pa, rules.TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("expiring one of the metric updates for Rule1 Inbound and one for Outbound")
		pa.OnUpdate(muConn1Rule1AllowExpire)
		expectedPacketsInbound += muConn1Rule1AllowExpire.inMetric.deltaPackets
		expectedBytesInbound += muConn1Rule1AllowExpire.inMetric.deltaBytes
		expectedPacketsOutbound += muConn1Rule1AllowExpire.outMetric.deltaPackets
		expectedBytesOutbound += muConn1Rule1AllowExpire.outMetric.deltaBytes
		expectedConnsInbound -= 1
		// Adjust the clock, but not past the retention period, the outbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)
		pa.CheckRetainedMetrics(mt.getMockTime())

		By("checking for the correct number of aggregated statistics: outbound rule should be present for retention time")
		expectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pa, rules.TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)
		expectRuleAggregates(pa, rules.TrafficDirOutbound, keyRule1Allow, expectedPacketsOutbound, expectedBytesOutbound, expectedConnsOutbound)

		By("incrementing time by the retention time - outbound rule should be expunged")
		mt.incMockTime(retentionTime)
		pa.CheckRetainedMetrics(mt.getMockTime())
		expectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("expiring the remaining Rule1 Inbound metric")
		pa.OnUpdate(muConn2Rule1AllowExpire)
		expectedPacketsInbound += muConn2Rule1AllowExpire.inMetric.deltaPackets
		expectedBytesInbound += muConn2Rule1AllowExpire.inMetric.deltaPackets
		expectedConnsInbound -= 1
		// Adjust the clock, but not past the retention period, the inbound rule aggregate should
		// not yet be expunged.
		mt.incMockTime(retentionTime / 2)
		pa.CheckRetainedMetrics(mt.getMockTime())

		By("checking for the correct number of aggregated statistics: inbound rule should be present for retention time")
		expectRuleAggregateKeys(pa, []RuleAggregateKey{keyRule1Allow})

		By("checking for the correct packet and byte counts")
		expectRuleAggregates(pa, rules.TrafficDirInbound, keyRule1Allow, expectedPacketsInbound, expectedBytesInbound, expectedConnsInbound)

		By("incrementing time by the retention time - inbound rule should be expunged")
		mt.incMockTime(retentionTime)
		pa.CheckRetainedMetrics(mt.getMockTime())
		expectRuleAggregateKeys(pa, []RuleAggregateKey{})
	})
})
