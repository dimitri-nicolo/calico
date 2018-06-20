package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudWatch Metrics aggregator aggregates the metric updates by ClusterID", func() {
	It("aggregates MetricUpdates with ingress and egress updates", func() {
		agg := NewCloudWatchMetricsAggregator(testClusterID)
		agg.FeedUpdate(muDenyIngress)
		agg.FeedUpdate(muDenyEgress)

		aggMetrics := agg.Get()

		Expect(len(aggMetrics)).Should(Equal(1))
		Expect(aggMetrics[0].Name).Should(Equal(dpMetricName))
		Expect(aggMetrics[0].Unit).Should(Equal(defaultDPUnit))
		Expect(aggMetrics[0].Value).Should(Equal(countTotalPackets(muDenyEgress, muDenyIngress)))
		Expect(aggMetrics[0].Dimensions).Should(Equal(map[string]string{"ClusterID": testClusterID}))
	})

	It("aggregates MetricUpdates with report and expire updates", func() {
		agg := NewCloudWatchMetricsAggregator(testClusterID)
		agg.FeedUpdate(muDenyIngress)
		agg.FeedUpdate(muDenyExpireEgress)

		aggMetrics := agg.Get()

		Expect(len(aggMetrics)).Should(Equal(1))
		Expect(aggMetrics[0].Name).Should(Equal(dpMetricName))
		Expect(aggMetrics[0].Unit).Should(Equal(defaultDPUnit))
		Expect(aggMetrics[0].Value).Should(Equal(countTotalPackets(muDenyExpireEgress, muDenyIngress)))
		Expect(aggMetrics[0].Dimensions).Should(Equal(map[string]string{"ClusterID": testClusterID}))
	})

	It("aggregates MetricUpdates with more than two updates", func() {
		agg := NewCloudWatchMetricsAggregator(testClusterID)
		agg.FeedUpdate(muDenyIngress)
		agg.FeedUpdate(muDenyEgress)
		agg.FeedUpdate(muDenyExpireEgress)

		aggMetrics := agg.Get()

		Expect(len(aggMetrics)).Should(Equal(1))
		Expect(aggMetrics[0].Name).Should(Equal(dpMetricName))
		Expect(aggMetrics[0].Unit).Should(Equal(defaultDPUnit))
		Expect(aggMetrics[0].Value).Should(Equal(countTotalPackets(muDenyExpireEgress, muDenyEgress, muDenyIngress)))
		Expect(aggMetrics[0].Dimensions).Should(Equal(map[string]string{"ClusterID": testClusterID}))
	})
})

func countTotalPackets(mu ...MetricUpdate) float64 {
	total := 0
	for _, v := range mu {
		total += v.inMetric.deltaPackets + v.outMetric.deltaPackets
	}

	return float64(total)
}
