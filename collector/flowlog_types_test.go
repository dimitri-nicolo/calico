// Copyright (c) 2017-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("FlowMeta construction from MetricUpdate", func() {
	DescribeTable("generates the correct FlowMeta using",
		func(input MetricUpdate, aggregation FlowAggregationKind, expected FlowMeta) {
			var flowMeta FlowMeta
			var err error

			flowMeta, err = NewFlowMeta(input, aggregation)
			Expect(err).To(BeNil())
			Expect(flowMeta).Should(Equal(expected))
		},
		Entry("full endpoints and default aggregation", muWithEndpointMeta, FlowDefault, flowMetaDefault),
		Entry("no source endpoints and default aggregation", muWithoutSrcEndpointMeta, FlowDefault, flowMetaDefaultNoSourceMeta),
		Entry("no destination endpoints and default aggregation", muWithoutDstEndpointMeta, FlowDefault, flowMetaDefaultNoDestMeta),
		Entry("full endpoints and source ports aggregation", muWithEndpointMeta, FlowSourcePort, flowMetaSourcePorts),
		Entry("full endpoints and prefix aggregation", muWithEndpointMeta, FlowPrefixName, flowMetaPrefix),
		Entry("no source endpoints and prefix aggregation", muWithoutSrcEndpointMeta, FlowPrefixName, flowMetaPrefixNoSourceMeta),
		Entry("no destination endpoints and prefix aggregation", muWithoutDstEndpointMeta, FlowPrefixName, flowMetaPrefixNoDestMeta),
		Entry("no generated name and prefix aggregation", muWithEndpointMetaWithoutGenerateName, FlowPrefixName, flowMetaPrefixWithName),
		Entry("full endpoints and dest port aggregation", muWithEndpointMeta, FlowNoDestPorts, flowMetaNoDestPorts),
		Entry("no source endpoints and dest port aggregation", muWithoutSrcEndpointMeta, FlowNoDestPorts, flowMetaNoDestPortNoSourceMeta),
		Entry("no destination and dest port aggregation", muWithoutDstEndpointMeta, FlowNoDestPorts, flowMetaNoDestPortNoDestMeta),
	)
})

var _ = Describe("Flow log types tests", func() {
	Context("FlowExtraRef from MetricUpdate", func() {
		It("generates the correct flowExtrasRef", func() {
			By("Extracting the correct information")
			fe := NewFlowExtrasRef(muWithOrigSourceIPs, testMaxBoundedSetSize)
			expectedFlowExtraRef := flowExtrasRef{
				originalSourceIPs: NewBoundedSetFromSlice(testMaxBoundedSetSize, []net.IP{net.ParseIP("1.0.0.1")}),
			}
			Expect(fe.originalSourceIPs.ToIPSlice()).Should(ConsistOf(expectedFlowExtraRef.originalSourceIPs.ToIPSlice()))
			Expect(fe.originalSourceIPs.TotalCount()).Should(Equal(expectedFlowExtraRef.originalSourceIPs.TotalCount()))
			Expect(fe.originalSourceIPs.TotalCountDelta()).Should(Equal(expectedFlowExtraRef.originalSourceIPs.TotalCountDelta()))

			By("aggregating the metric update")
			fe.aggregateFlowExtrasRef(muWithMultipleOrigSourceIPs)
			expectedFlowExtraRef = flowExtrasRef{
				originalSourceIPs: NewBoundedSetFromSlice(testMaxBoundedSetSize, []net.IP{net.ParseIP("1.0.0.1"), net.ParseIP("2.0.0.2")}),
			}
			Expect(fe.originalSourceIPs.ToIPSlice()).Should(ConsistOf(expectedFlowExtraRef.originalSourceIPs.ToIPSlice()))
			Expect(fe.originalSourceIPs.TotalCount()).Should(Equal(expectedFlowExtraRef.originalSourceIPs.TotalCount()))
			Expect(fe.originalSourceIPs.TotalCountDelta()).Should(Equal(expectedFlowExtraRef.originalSourceIPs.TotalCountDelta()))
		})
	})
})
