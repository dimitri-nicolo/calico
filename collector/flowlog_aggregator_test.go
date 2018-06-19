// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flow log aggregator verification", func() {
	It("aggregates the fed metric updates", func() {
		expectFlowLog := func(msg string, t Tuple, nf, nfs, nfc int, a FlowLogAction, fd FlowLogDirection) {
			fl, err := getFlowLog(msg)
			Expect(err).To(BeNil())
			Expect(fl.Tuple).Should(Equal(t))
			Expect(fl.NumFlows).Should(Equal(nf))
			Expect(fl.NumFlowsStarted).Should(Equal(nfs))
			Expect(fl.NumFlowsCompleted).Should(Equal(nfc))
			Expect(fl.Action).Should(Equal(a))
			Expect(fl.FlowDirection).Should(Equal(fd))
		}

		By("defalt duration")
		ca := NewCloudWatchAggregator()
		ca.FeedUpdate(muNoConn1Rule1AllowUpdate)
		messages := ca.Get()
		message := *(messages[0])

		expectedNumFlows := 1
		expectedNumFlowsStarted := 1
		expectedNumFlowsCompleted := 0
		expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn)

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
		Expect(len(messages)).Should(Equal(1))
		message = *(messages[0])
		expectedTuple := tuple1Copy
		expectedTuple.l4Src = 0
		expectedNumFlows++
		expectedNumFlowsStarted++

		expectFlowLog(message, expectedTuple, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn)

	})
})
