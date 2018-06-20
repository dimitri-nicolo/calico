// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flow log aggregator verification", func() {
	It("aggregates the fed metric updates", func() {
		expectFlowLog := func(msg string, t Tuple, nf, nfs, nfc int, a FlowLogAction, fd FlowLogDirection, pi, po, bi, bo int) {
			fl, err := getFlowLog(msg)
			Expect(err).To(BeNil())
			expectedFlow := newExpectedFlowLog(t, nf, nfs, nfc, a, fd, pi, po, bi, bo)
			Expect(fl).Should(Equal(expectedFlow))
		}
		calculatePacketStats := func(mus ...MetricUpdate) (epi, epo, ebi, ebo int) {
			for _, mu := range mus {
				epi += mu.inMetric.deltaPackets
				epo += mu.outMetric.deltaPackets
				ebi += mu.inMetric.deltaBytes
				ebo += mu.outMetric.deltaBytes
			}
			return
		}

		By("defalt duration")
		ca := NewCloudWatchAggregator()
		ca.FeedUpdate(muNoConn1Rule1AllowUpdate)
		messages := ca.Get()
		Expect(len(messages)).Should(Equal(1))
		message := *(messages[0])

		expectedNumFlows := 1
		expectedNumFlowsStarted := 1
		expectedNumFlowsCompleted := 0

		expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdate)
		expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

		By("source port")
		ca = NewCloudWatchAggregator().AggregateOver(SourcePort)
		ca.FeedUpdate(muNoConn1Rule1AllowUpdate)
		// Construct a similar update; same tuple but diff src ports.
		muNoConn1Rule1AllowUpdateCopy := muNoConn1Rule1AllowUpdate
		tuple1Copy := tuple1
		tuple1Copy.l4Src = 44123
		muNoConn1Rule1AllowUpdateCopy.tuple = tuple1Copy
		ca.FeedUpdate(muNoConn1Rule1AllowUpdateCopy)
		messages = ca.Get()
		// Two updates should still result in 1 flow
		Expect(len(messages)).Should(Equal(1))
	})
})
