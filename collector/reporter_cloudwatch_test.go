// Copyright (c) 2017-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"os"
	"time"

	"github.com/projectcalico/felix/collector/testutil"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	localIp  = [16]byte{10, 0, 0, 1}
	remoteIp = [16]byte{20, 0, 0, 1}
)

// Common Tuple definitions
var (
	tuple = *NewTuple(localIp, remoteIp, proto_tcp, srcPort, dstPort)
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
		log.SetOutput(os.Stdout)
		cl = testutil.NewMockedCloudWatchLogsClient()
		cd = NewCloudWatchDispatcher(cl)
		ca = NewCloudWatchAggregator()
		cr = newCloudWatchReporter(cd, ca, retentionTime)
		cr.timeNowFn = mt.getMockTime
		go cr.run()
	})
	It("reports the given metric update in form of a flow to cloudwatchlogs", func() {
		cr.Report(muNoConn1Rule1AllowUpdate)
		time.Sleep(3 * time.Second)
		logGroupName := "test-group"
		logStreamName := "test-stream"
		logEventsInput := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroupName,
			LogStreamName: &logStreamName,
		}
		logEventsOutput, _ := cl.GetLogEvents(logEventsInput)
		events := logEventsOutput.Events
		// the sole Message
		message := *(events[0].Message)
		Expect(message).Should(Equal("10.0.0.1 20.0.0.1 80 6"))
	})
})
