// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.

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

			flowMeta, err = NewFlowMeta(input, aggregation, true)
			Expect(err).To(BeNil())
			Expect(flowMeta).Should(Equal(expected))
		},
		Entry("full endpoints and default aggregation", muWithEndpointMeta, FlowDefault, flowMetaDefault),
		Entry("full endpoints with service and default aggregation", muWithEndpointMetaWithService, FlowDefault, flowMetaDefaultWithService),
		Entry("no source endpoints and default aggregation", muWithoutSrcEndpointMeta, FlowDefault, flowMetaDefaultNoSourceMeta),
		Entry("no destination endpoints and default aggregation", muWithoutDstEndpointMeta, FlowDefault, flowMetaDefaultNoDestMeta),
		Entry("full endpoints and source ports aggregation", muWithEndpointMeta, FlowSourcePort, flowMetaSourcePorts),
		Entry("full endpoints and prefix aggregation", muWithEndpointMeta, FlowPrefixName, flowMetaPrefix),
		Entry("no source endpoints and prefix aggregation", muWithoutSrcEndpointMeta, FlowPrefixName, flowMetaPrefixNoSourceMeta),
		Entry("no destination endpoints and prefix aggregation", muWithoutDstEndpointMeta, FlowPrefixName, flowMetaPrefixNoDestMeta),
		Entry("no generated name and prefix aggregation", muWithEndpointMetaWithoutGenerateName, FlowPrefixName, flowMetaPrefixWithName),
		Entry("full endpoints and dest port aggregation", muWithEndpointMeta, FlowNoDestPorts, flowMetaNoDestPorts),
		Entry("full endpoints with service and dest port aggregation", muWithEndpointMetaWithService, FlowNoDestPorts, flowMetaNoDestPortsWithService),
		Entry("no source endpoints and dest port aggregation", muWithoutSrcEndpointMeta, FlowNoDestPorts, flowMetaNoDestPortNoSourceMeta),
		Entry("no destination and dest port aggregation", muWithoutDstEndpointMeta, FlowNoDestPorts, flowMetaNoDestPortNoDestMeta),
	)
})

func consists(actual, expected []FlowProcessReportedStats) bool {
	count := 0
	for _, expflow := range expected {
		for _, actFlow := range actual {
			if compareProcessReportedStats(expflow, actFlow) {
				count = count + 1
			}
		}
	}
	if count == len(expected) {
		return true
	}
	return false
}

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

	Context("FlowStatsByProcess from MetricUpdate", func() {
		arg1List := []string{"arg1"}
		arg12List := []string{"arg1", "arg2"}
		arg3List := []string{"arg3"}
		emptyList := []string{"-"}
		It("stores the correct FlowStatsByProcess when storing process is enabled", func() {
			By("Extracting the correct information")
			fsp := NewFlowStatsByProcess(&muWithProcessName, true, 2, 5, false, 3)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			expectedReportedStats := []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "1234",
					NumProcessIDs:   1,
					ProcessArgs:     arg1List,
					NumProcessArgs:  1,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(1))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("aggregating the metric update with same process name but different process ID and args")
			fsp.aggregateFlowStatsByProcess(&muWithSameProcessNameDifferentID)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "*",
					NumProcessIDs:   2,
					ProcessArgs:     arg12List,
					NumProcessArgs:  2,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             2,
						PacketsOut:            0,
						BytesIn:               40,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              2,
						NumFlowsStarted:       2,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           12,
						TotalRetrans:      14,
						UnrecoveredRTO:    16,
						Count:             2,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(2))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("aggregating the metric update with a different process name and ID")
			fsp.aggregateFlowStatsByProcess(&muWithDifferentProcessNameDifferentID)
			Expect(fsp.statsByProcessName).Should(HaveLen(2))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-2"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "*",
					NumProcessIDs:   2,
					ProcessArgs:     arg12List,
					NumProcessArgs:  2,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             2,
						PacketsOut:            0,
						BytesIn:               40,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              2,
						NumFlowsStarted:       2,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           12,
						TotalRetrans:      14,
						UnrecoveredRTO:    16,
						Count:             2,
					},
				},
				FlowProcessReportedStats{
					ProcessName:     "test-process-2",
					NumProcessNames: 1,
					ProcessID:       "23456",
					NumProcessIDs:   1,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(3))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("aggregating the metric update with same process name with update type expire")
			fsp.aggregateFlowStatsByProcess(&muWithProcessNameExpire)
			Expect(fsp.statsByProcessName).Should(HaveLen(2))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-2"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "*",
					NumProcessIDs:   2,
					ProcessArgs:     arg12List,
					NumProcessArgs:  2,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             2,
						PacketsOut:            0,
						BytesIn:               40,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              2,
						NumFlowsStarted:       2,
						NumFlowsCompleted:     1,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           12,
						TotalRetrans:      14,
						UnrecoveredRTO:    16,
						Count:             2,
					},
				},
				FlowProcessReportedStats{
					ProcessName:     "test-process-2",
					NumProcessNames: 1,
					ProcessID:       "23456",
					NumProcessIDs:   1,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(2))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("cleaning up the stats for the process name")
			fsp.aggregateFlowStatsByProcess(&muWithSameProcessNameDifferentIDExpire)
			fsp.reset()
			remainingActiveFlowsCount := fsp.gc()
			Expect(remainingActiveFlowsCount).Should(Equal(1))
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-2"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process-2",
					NumProcessNames: 1,
					ProcessID:       "23456",
					NumProcessIDs:   1,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             0,
						PacketsOut:            0,
						BytesIn:               0,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       0,
						NumFlowsCompleted:     0,
					},
				},
			}
			By("Flow logs continues to contain process ID")
			Expect(fsp.getActiveFlowsCount()).Should(Equal(1))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("Adding a new metric update after a reset, the new process ID information is exported")
			fsp.aggregateFlowStatsByProcess(&muWithProcessName2)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-2"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process-2",
					NumProcessNames: 1,
					ProcessID:       "9876",
					NumProcessIDs:   1,
					NumProcessArgs:  1,
					ProcessArgs:     arg3List,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       0,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(1))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))
		})

		It("stores the correct FlowStatsByProcess with including process information is disabled", func() {
			By("Extracting the correct information")
			fsp := NewFlowStatsByProcess(&muWithEndpointMeta, false, 0, 5, false, 3)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("-"))
			expectedReportedStats := []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "-",
					NumProcessNames: 0,
					ProcessID:       "-",
					NumProcessIDs:   0,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(1))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("aggregating the metric update")
			fsp.aggregateFlowStatsByProcess(&muWithEndpointMetaWithService)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("-"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "-",
					NumProcessNames: 0,
					ProcessID:       "-",
					NumProcessIDs:   0,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             2,
						PacketsOut:            0,
						BytesIn:               40,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           12,
						TotalRetrans:      14,
						UnrecoveredRTO:    16,
						Count:             2,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(1))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("aggregating the metric update with update type expire")
			fsp.aggregateFlowStatsByProcess(&muWithEndpointMetaExpire)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("-"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "-",
					NumProcessNames: 0,
					ProcessID:       "-",
					NumProcessIDs:   0,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             2,
						PacketsOut:            0,
						BytesIn:               40,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     1,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           12,
						TotalRetrans:      14,
						UnrecoveredRTO:    16,
						Count:             2,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(0))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("cleaning up the stats for the process name")
			remainingActiveFlowsCount := fsp.gc()
			Expect(remainingActiveFlowsCount).Should(Equal(0))
			Expect(fsp.statsByProcessName).Should(HaveLen(0))
		})

		It("limits the process name information when converting FlowStatsByProcess when process information collection is enabled", func() {
			By("Extracting the correct information")
			fsp := NewFlowStatsByProcess(&muWithProcessName, true, 2, 5, false, 3)
			Expect(fsp.statsByProcessName).Should(HaveLen(1))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			expectedReportedStats := []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "1234",
					NumProcessIDs:   1,
					ProcessArgs:     arg1List,
					NumProcessArgs:  1,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(1))
			Expect(fsp.toFlowProcessReportedStats()).To(ConsistOf(expectedReportedStats))

			By("aggregating the metric update with different process name")
			fsp.aggregateFlowStatsByProcess(&muWithProcessName2)
			Expect(fsp.statsByProcessName).Should(HaveLen(2))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-2"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "1234",
					NumProcessIDs:   1,
					ProcessArgs:     arg1List,
					NumProcessArgs:  1,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
				FlowProcessReportedStats{
					ProcessName:     "test-process-2",
					NumProcessNames: 1,
					ProcessID:       "9876",
					NumProcessIDs:   1,
					ProcessArgs:     arg3List,
					NumProcessArgs:  1,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(2))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))

			By("aggregating the metric update with a two additional process names")
			fsp.aggregateFlowStatsByProcess(&muWithProcessName3)
			fsp.aggregateFlowStatsByProcess(&muWithProcessName4)
			Expect(fsp.statsByProcessName).Should(HaveLen(4))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process"))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-2"))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-3"))
			Expect(fsp.statsByProcessName).Should(HaveKey("test-process-4"))
			expectedReportedStats = []FlowProcessReportedStats{
				FlowProcessReportedStats{
					ProcessName:     "test-process",
					NumProcessNames: 1,
					ProcessID:       "1234",
					NumProcessIDs:   1,
					ProcessArgs:     arg1List,
					NumProcessArgs:  1,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
				FlowProcessReportedStats{
					ProcessName:     "test-process-2",
					NumProcessNames: 1,
					ProcessID:       "9876",
					NumProcessIDs:   1,
					ProcessArgs:     arg3List,
					NumProcessArgs:  1,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             1,
						PacketsOut:            0,
						BytesIn:               20,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              1,
						NumFlowsStarted:       1,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           6,
						TotalRetrans:      7,
						UnrecoveredRTO:    8,
						Count:             1,
					},
				},
				FlowProcessReportedStats{
					ProcessName:     "*",
					NumProcessNames: 2,
					ProcessID:       "*",
					NumProcessIDs:   2,
					ProcessArgs:     emptyList,
					NumProcessArgs:  0,
					FlowReportedStats: FlowReportedStats{
						PacketsIn:             2,
						PacketsOut:            0,
						BytesIn:               40,
						BytesOut:              0,
						HTTPRequestsAllowedIn: 0,
						HTTPRequestsDeniedIn:  0,
						NumFlows:              2,
						NumFlowsStarted:       2,
						NumFlowsCompleted:     0,
					},
					FlowReportedTCPStats: FlowReportedTCPStats{
						SendCongestionWnd: TCPWnd{Min: 10, Mean: 10},
						SmoothRtt:         TCPRtt{Max: 1, Mean: 1},
						MinRtt:            TCPRtt{Max: 2, Mean: 2},
						Mss:               TCPMss{Min: 4, Mean: 4},
						LostOut:           12,
						TotalRetrans:      14,
						UnrecoveredRTO:    16,
						Count:             2,
					},
				},
			}
			Expect(fsp.getActiveFlowsCount()).Should(Equal(4))
			Expect(consists(fsp.toFlowProcessReportedStats(), expectedReportedStats)).Should(Equal(true))
		})
	})
})
