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
	BeforeEach(func() {
		cl = testutil.NewMockedCloudWatchLogsClient(logGroupName)
		cd = NewCloudWatchDispatcher(logGroupName, logStreamName, cl)
		ca = NewCloudWatchAggregator(includeLabels)
		cr = NewCloudWatchReporter(cd, flushInterval)
		cr.AddAggregator(ca)
		cr.timeNowFn = mt.getMockTime
		cr.Start()
	})
	It("reports the given metric update in form of a flow to cloudwatchlogs", func() {
		getEventsFromLogStream := func() []*cloudwatchlogs.OutputLogEvent {
			logEventsInput := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &logGroupName,
				LogStreamName: &logStreamName,
			}
			logEventsOutput, _ := cl.GetLogEvents(logEventsInput)
			return logEventsOutput.Events
		}
		expectFlowLog := func(msg string, t Tuple, nf, nfs, nfc int, a FlowLogAction, fd FlowLogDirection) {
			fl, err := getFlowLog(msg)
			Expect(err).To(BeNil())
			Expect(fl.Tuple).Should(Equal(tuple1))
			Expect(fl.NumFlows).Should(Equal(nf))
			Expect(fl.NumFlowsStarted).Should(Equal(nfs))
			Expect(fl.NumFlowsCompleted).Should(Equal(nfc))
			Expect(fl.Action).Should(Equal(a))
			Expect(fl.FlowDirection).Should(Equal(fd))
		}

		By("reporting the first MetricUpdate")
		cr.Report(muNoConn1Rule1AllowUpdate)
		// Wait for aggregation and export to happen.
		time.Sleep(1 * time.Second)
		events := getEventsFromLogStream()
		message := *(events[0].Message)
		expectedNumFlows := 1
		expectedNumFlowsStarted := 1
		expectedNumFlowsCompleted := 0
		expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn)

		By("reporting the same MetricUpdate with metrics in both directions")
		cr.Report(muConn1Rule1AllowUpdate)
		// Wait for aggregation and export to happen.
		time.Sleep(1 * time.Second)
		events = getEventsFromLogStream()
		message = *(events[0].Message)
		expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn)

		By("reporting a expired MetricUpdate for the same tuple")
		cr.Report(muConn1Rule1AllowExpire)
		// Wait for aggregation and export to happen.
		time.Sleep(1 * time.Second)
		events = getEventsFromLogStream()
		message = *(events[0].Message)
		expectedNumFlowsStarted = 0
		expectedNumFlowsCompleted = 1
		expectFlowLog(message, tuple1, expectedNumFlows, expectedNumFlowsStarted, expectedNumFlowsCompleted, FlowLogActionAllow, FlowLogDirectionIn)

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

	lip := ipStrTo16Byte(parts[10])
	rip := ipStrTo16Byte(parts[11])
	p, _ := strconv.Atoi(parts[12])
	sp, _ := strconv.Atoi(parts[13])
	dp, _ := strconv.Atoi(parts[14])
	tuple := *NewTuple(lip, rip, p, sp, dp)

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
