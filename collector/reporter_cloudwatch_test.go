// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"encoding/json"
	"strconv"
	"strings"
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
	getLastMessageFromLogStream := func() string {
		events := getEventsFromLogStream()
		return *(events[len(events)-1].Message)
	}
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
	Context("No Aggregation kind specified", func() {
		BeforeEach(func() {
			cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd = NewCloudWatchDispatcher(logGroupName, logStreamName, cl)
			ca = NewCloudWatchAggregator()
			cr = NewCloudWatchReporter(cd, flushInterval)
			cr.AddAggregator(ca)
			cr.timeNowFn = mt.getMockTime
			cr.Start()
		})
		It("reports the given metric update in form of a flow to cloudwatchlogs", func() {
			By("reporting the first MetricUpdate")
			cr.Report(muNoConn1Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			message := getLastMessageFromLogStream()
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muNoConn1Rule1AllowUpdate)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

			By("reporting the same MetricUpdate with metrics in both directions")
			cr.Report(muConn1Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			message = getLastMessageFromLogStream()
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muConn1Rule1AllowUpdate)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

			By("reporting a expired MetricUpdate for the same tuple")
			cr.Report(muConn1Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			message = getLastMessageFromLogStream()
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muConn1Rule1AllowExpire)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

			By("reporting a MetricUpdate for denied packets")
			cr.Report(muNoConn3Rule2DenyUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			message = getLastMessageFromLogStream()
			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule2DenyUpdate)
			expectFlowLog(message, tuple3, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogDirectionOut,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

			By("reporting a expired denied packet MetricUpdate for the same tuple")
			cr.Report(muNoConn3Rule2DenyExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			message = getLastMessageFromLogStream()
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut = calculatePacketStats(muNoConn1Rule2DenyExpire)
			expectFlowLog(message, tuple3, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogDirectionOut,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

		})
		It("aggregates metric updates for the duration of aggregation when reporting to cloudwatch logs", func() {

			By("reporting the same MetricUpdate twice and expiring it immediately")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn1Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			message := getLastMessageFromLogStream()
			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 1
			expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut := calculatePacketStats(muConn1Rule1AllowUpdate, muConn1Rule1AllowUpdate, muConn1Rule1AllowExpire)
			expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
				expectedPacketsIn, expectedPacketsOut, expectedBytesIn, expectedBytesOut)

			By("reporting the same tuple different policies should be reported as separate flow logs")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muNoConn1Rule2DenyUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			events := getEventsFromLogStream()
			message1 := *(events[len(events)-2].Message)
			flow1, err := getFlowLog(message1)
			Expect(err).To(BeNil())
			message2 := *(events[len(events)-1].Message)
			flow2, err := getFlowLog(message2)
			Expect(err).To(BeNil())

			expectedNumFlows = 1
			expectedNumFlowsStarted = 1
			expectedNumFlowsCompleted = 0
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 := calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 := calculatePacketStats(muNoConn1Rule2DenyUpdate)
			// We only care about the flow log entry to exist and don't care about the actual order.
			Expect([]FlowLog{flow1, flow2}).Should(ConsistOf(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1),
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionDeny, FlowLogDirectionOut,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2),
			))
		})

		It("aggregates metric updates from multiple tuples", func() {

			By("report different connections")
			cr.Report(muConn1Rule1AllowUpdate)
			cr.Report(muConn2Rule1AllowUpdate)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			events := getEventsFromLogStream()
			message1 := *(events[0].Message)
			flow1, err := getFlowLog(message1)
			Expect(err).To(BeNil())
			message2 := *(events[1].Message)
			flow2, err := getFlowLog(message2)
			Expect(err).To(BeNil())

			expectedNumFlows := 1
			expectedNumFlowsStarted := 1
			expectedNumFlowsCompleted := 0
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 := calculatePacketStats(muConn1Rule1AllowUpdate)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 := calculatePacketStats(muConn2Rule1AllowUpdate)
			// We only care about the flow log entry to exist and don't care about the actual order.
			Expect([]FlowLog{flow1, flow2}).Should(ConsistOf(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1),
				newExpectedFlowLog(tuple2, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2),
			))

			By("report expirations of the same connections")
			cr.Report(muConn1Rule1AllowExpire)
			cr.Report(muConn2Rule1AllowExpire)
			// Wait for aggregation and export to happen.
			time.Sleep(1 * time.Second)
			events = getEventsFromLogStream()
			message1 = *(events[len(events)-2].Message)
			flow1, err = getFlowLog(message1)
			Expect(err).To(BeNil())
			message2 = *(events[len(events)-1].Message)
			flow2, err = getFlowLog(message2)
			Expect(err).To(BeNil())

			expectedNumFlows = 1
			expectedNumFlowsStarted = 0
			expectedNumFlowsCompleted = 1
			expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1 = calculatePacketStats(muConn1Rule1AllowExpire)
			expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2 = calculatePacketStats(muConn2Rule1AllowExpire)
			// We only care about the flow log entry to exist and don't care about the actual order.
			Expect([]FlowLog{flow1, flow2}).Should(ConsistOf(
				newExpectedFlowLog(tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn1, expectedPacketsOut1, expectedBytesIn1, expectedBytesOut1),
				newExpectedFlowLog(tuple2, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn,
					expectedPacketsIn2, expectedPacketsOut2, expectedBytesIn2, expectedBytesOut2),
			))

		})
	})
})

func getFlowLog(fl string) (FlowLog, error) {
	// Format is
	// startTime endTime srcType srcNamespace srcName srcLabels dstType dstNamespace dstName dstLabels srcIP dstIP proto srcPort dstPort numFlows numFlowsStarted numFlowsCompleted flowDirection packetsIn packetsOut bytesIn bytesOut action
	var (
		srcLabels, dstLabels map[string]string
		srcType, dstType     FlowLogEndpointType
	)

	parts := strings.Split(fl, " ")

	switch parts[2] {
	case "wep":
		srcType = FlowLogEndpointTypeWep
	case "hep":
		srcType = FlowLogEndpointTypeHep
	case "ns":
		srcType = FlowLogEndpointTypeNs
	case "pvt":
		srcType = FlowLogEndpointTypePvt
	case "pub":
		srcType = FlowLogEndpointTypePub
	}

	srcMeta := EndpointMetadata{
		Type:      srcType,
		Namespace: parts[3],
		Name:      parts[4],
	}
	_ = json.Unmarshal([]byte(parts[5]), &srcLabels)
	srcMeta.Labels = srcLabels

	switch parts[6] {
	case "wep":
		dstType = FlowLogEndpointTypeWep
	case "hep":
		dstType = FlowLogEndpointTypeHep
	case "ns":
		dstType = FlowLogEndpointTypeNs
	case "pvt":
		dstType = FlowLogEndpointTypePvt
	case "pub":
		dstType = FlowLogEndpointTypePub
	}

	dstMeta := EndpointMetadata{
		Type:      dstType,
		Namespace: parts[7],
		Name:      parts[8],
	}
	_ = json.Unmarshal([]byte(parts[9]), &dstLabels)
	dstMeta.Labels = srcLabels

	var sip [16]byte
	if parts[10] != "-" {
		sip = ipStrTo16Byte(parts[10])
	}
	dip := ipStrTo16Byte(parts[11])
	p, _ := strconv.Atoi(parts[12])
	sp, _ := strconv.Atoi(parts[13])
	dp, _ := strconv.Atoi(parts[14])
	tuple := *NewTuple(sip, dip, p, sp, dp)

	nf, _ := strconv.Atoi(parts[15])
	nfs, _ := strconv.Atoi(parts[16])
	nfc, _ := strconv.Atoi(parts[17])

	var fd FlowLogDirection
	switch parts[18] {
	case "I":
		fd = FlowLogDirectionIn
	case "O":
		fd = FlowLogDirectionOut
	}

	pi, _ := strconv.Atoi(parts[19])
	po, _ := strconv.Atoi(parts[20])
	bi, _ := strconv.Atoi(parts[21])
	bo, _ := strconv.Atoi(parts[22])

	var a FlowLogAction
	switch parts[23] {
	case "A":
		a = FlowLogActionAllow
	case "D":
		a = FlowLogActionDeny
	}

	return FlowLog{
		Tuple:             tuple,
		SrcMeta:           srcMeta,
		DstMeta:           dstMeta,
		NumFlows:          nf,
		NumFlowsStarted:   nfs,
		NumFlowsCompleted: nfc,
		FlowDirection:     fd,
		PacketsIn:         pi,
		PacketsOut:        po,
		BytesIn:           bi,
		BytesOut:          bo,
		Action:            a,
	}, nil
}

func newExpectedFlowLog(t Tuple, nf, nfs, nfc int, a FlowLogAction, fd FlowLogDirection, pi, po, bi, bo int) FlowLog {
	return FlowLog{
		Tuple:             t,
		NumFlows:          nf,
		NumFlowsStarted:   nfs,
		NumFlowsCompleted: nfc,
		FlowDirection:     fd,
		Action:            a,
		PacketsIn:         pi,
		PacketsOut:        po,
		BytesIn:           bi,
		BytesOut:          bo,
	}
}
