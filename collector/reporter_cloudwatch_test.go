// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"reflect"
	"time"

	"github.com/projectcalico/felix/collector/testutil"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	logGroupName  = "test-group"
	logStreamName = "test-stream"
	flushInterval = 500 * time.Millisecond
	includeLabels = false
)

var (
	pvtMeta = EndpointMetadata{Type: FlowLogEndpointTypeNet, Namespace: "-", Name: "pvt", Labels: "-"}
	pubMeta = EndpointMetadata{Type: FlowLogEndpointTypeNet, Namespace: "-", Name: "pub", Labels: "-"}
)

var _ = Describe("CloudWatch Reporter verification", func() {
	var (
		cr *cloudWatchReporter
		cd FlowLogDispatcher
		ca FlowLogAggregator
		cl cloudwatchlogsiface.CloudWatchLogsAPI
	)

	mt := &mockTime{}
	getEventsFromLogStream := func() []*cloudwatchlogs.OutputLogEvent {
		logEventsInput := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroupName,
			LogStreamName: &logStreamName,
		}
		logEventsOutput, _ := cl.GetLogEvents(logEventsInput)
		return logEventsOutput.Events
	}
	expectFlowLogsInEventStream := func(fls ...FlowLog) {
		events := getEventsFromLogStream()
		count := 0
		for _, fl := range fls {
			for _, ev := range events {
				flMsg, _ := getFlowLog(*ev.Message)
				if reflect.DeepEqual(flMsg, fl) {
					count++
					if count == len(fls) {
						break
					}
				}
			}
		}
		Expect(count).Should(Equal(len(fls)))
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
	Context("No Aggregation kind specified", func() {
		BeforeEach(func() {
			cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl)
			ca = NewCloudWatchAggregator()
			cr = NewCloudWatchReporter(cd, flushInterval, nil)
			cr.AddAggregator(ca)
			cr.timeNowFn = mt.getMockTime
			cr.Start()
		})
		AfterEach(func() {
			Expect(cl.(testutil.CloudWatchLogsExpectation).RetentionPeriod()).To(BeNumerically("==", 7))
		})
		It("reports the given metric update in form of a flow to cloudwatchlogs", func() {
			By("reporting the first MetricUpdate")
			cr.Report(muNoConn1Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdate)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta))

			By("reporting the same MetricUpdate with metrics in both directions")
			cr.Report(muConn1Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlowsStarted = 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muConn1Rule1AllowUpdate)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta))

			By("reporting a expired MetricUpdate for the same tuple")
			cr.Report(muConn1Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muConn1Rule1AllowExpire)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta))

			By("reporting a MetricUpdate for denied packets")
			cr.Report(muNoConn3Rule2DenyUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule2DenyUpdate)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple3, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogDirectionOut,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta))

			By("reporting a expired denied packet MetricUpdate for the same tuple")
			cr.Report(muNoConn3Rule2DenyExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule2DenyExpire)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple3, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogDirectionOut,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta))

		})
		It("aggregates metric updates for the duration of aggregation when reporting to cloudwatch logs", func() {

			By("reporting the same MetricUpdate twice and expiring it immediately")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn1Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muConn1Rule1AllowUpdate, muConn1Rule1AllowUpdate, muConn1Rule1AllowExpire)
			expectFlowLogsInEventStream(newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut, pvtMeta, pubMeta))

			By("reporting the same tuple different policies should be reported as separate flow logs")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muNoConn1Rule2DenyUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)

			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 := calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 := calculatePacketStats(muNoConn1Rule2DenyUpdate)
			// We only care about the flow log entry to exist and don't care about the actual order.
			expectFlowLogsInEventStream(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1, pvtMeta, pubMeta),
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogDirectionOut,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2, pvtMeta, pubMeta))
		})

		It("aggregates metric updates from multiple tuples", func() {

			By("report different connections")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn2Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)

			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 := calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 := calculatePacketStats(muConn2Rule1AllowUpdate)
			// We only care about the flow log entry to exist and don't care about the actual order.
			expectFlowLogsInEventStream(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1, pvtMeta, pubMeta),
				newExpectedFlowLog(tuple2, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2, pvtMeta, pubMeta))

			By("report expirations of the same connections")
			cr.Report(muConn1Rule1AllowExpire)
			cr.Report(muConn2Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)

			expectedNumFlows = 1
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 = calculatePacketStats(muConn1Rule1AllowExpire)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 = calculatePacketStats(muConn2Rule1AllowExpire)
			// We only care about the flow log entry to exist and don't care about the actual order.
			expectFlowLogsInEventStream(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1, pvtMeta, pubMeta),
				newExpectedFlowLog(tuple2, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2, pvtMeta, pubMeta))

		})
		It("Doesn't process flows from Hostendoint to Hostendpoint", func() {
			By("Reporting a update with host endpoint to host endpoint")
			muConn1Rule1AllowUpdateCopy := muConn1Rule1AllowUpdate
			muConn1Rule1AllowUpdateCopy.srcEp = localHostEd1
			muConn1Rule1AllowUpdateCopy.dstEp = remoteHostEd1
			cr.Report(muConn1Rule1AllowUpdateCopy)

			By("Verifying that no flow logs are logged")
			events := getEventsFromLogStream()
			Expect(len(events)).Should(Equal(0))
		})
	})
})

func getFlowLog(fl string) (FlowLog, error) {
	flowLog := &FlowLog{}
	err := flowLog.Deserialize(fl)
	return *flowLog, err
}

func newExpectedFlowLog(t Tuple, nf, nfs, nfc int, a FlowLogAction, fd FlowLogDirection, pi, po, bi, bo int, srcMeta, dstMeta EndpointMetadata) FlowLog {
	return FlowLog{
		FlowMeta{
			Tuple:     t,
			Action:    a,
			Direction: fd,
			SrcMeta:   srcMeta,
			DstMeta:   dstMeta,
		},
		FlowStats{
			NumFlows:          nf,
			NumFlowsStarted:   nfs,
			NumFlowsCompleted: nfc,
			PacketsIn:         pi,
			PacketsOut:        po,
			BytesIn:           bi,
			BytesOut:          bo,
		},
	}
}
