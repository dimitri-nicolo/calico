// Copyright (c) 2017-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudWatch Metrics aggregator aggregates the metric updates by ClusterID", func() {
	var agg MetricAggregator

	BeforeEach(func() {
		agg = NewCloudWatchMetricsAggregator(testClusterID)
	})

	It("aggregates MetricUpdates with ingress and egress updates", func() {
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

	It("Get does not cause a data race contention on the flowEntry after FeedUpdate adds it to the mMap", func() {
		var aggMetrics []MetricData

		time.AfterFunc(2*time.Second, func() {
			agg.FeedUpdate(muDenyIngress)
			agg.FeedUpdate(muDenyEgress)
			agg.FeedUpdate(muDenyExpireEgress)

		})

		// ok Get is a little after feedupdate because feedupdate has some preprocesssing
		// before ti accesses flowstore
		time.AfterFunc(2*time.Second+10*time.Millisecond, func() {
			aggMetrics = agg.Get()
		})

		time.Sleep(3 * time.Second)

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
