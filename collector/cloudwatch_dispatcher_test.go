// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/felix/collector/testutil"
)

var _ = Describe("CloudWatch Dispatcher verification", func() {
	var (
		batcher *cloudWatchEventsBatcher
	)

	Context("Events batcher", func() {
		msg1 := "hello"
		msg2 := "world"
		It("Batches events", func() {
			By("flushing out events even when total input array size less than batch size")
			testBatchSize := 50 // bytes
			batchChan := make(chan eventsBatch)
			batcher = newCloudWatchEventsBatcher(testBatchSize, batchChan)
			storeMsg := map[string]bool{msg1: true, msg2: true}
			go batcher.batch([]*string{&msg1, &msg2})

			batchCount := 0
			for {
				events, more := <-batcher.eventsBatchChan
				if !more {
					break
				}
				batchCount++
				for _, e := range events {
					delete(storeMsg, *e.Message)
				}
			}
			// At the end of it, all messages should have been seen
			Expect(len(storeMsg)).Should(Equal(0))
			Expect(batchCount).Should(Equal(1))

			By("flushing out events in batches when total input array size more than batch size")
			testBatchSize = 5 // bytes
			batchChan = make(chan eventsBatch)
			batcher = newCloudWatchEventsBatcher(testBatchSize, batchChan)
			storeMsg = map[string]bool{msg1: true, msg2: true}
			go batcher.batch([]*string{&msg1, &msg2})

			batchCount = 0
			for {
				events, more := <-batcher.eventsBatchChan
				if !more {
					break
				}
				batchCount++
				for _, e := range events {
					delete(storeMsg, *e.Message)
				}
			}
			// At the end of it, all messages should have been seen
			Expect(len(storeMsg)).Should(Equal(0))
			Expect(batchCount).Should(Equal(2))
		})
	})

	Context("Events Uploader", func() {
		It("Uploads batches to cloudwatchlogs", func() {
			msg1 := "hello"
			msg2 := "world"
			By("Multiple batches")
			cl := testutil.NewMockedCloudWatchLogsClient(logGroupName)
			cd := NewCloudWatchDispatcher(logGroupName, logStreamName, 7, cl).(*cloudWatchDispatcher)

			testBatchSize := 5 // bytes
			batchChan := make(chan eventsBatch)
			batcher := newCloudWatchEventsBatcher(testBatchSize, batchChan)
			storeMsg := map[string]bool{msg1: true, msg2: true}
			go batcher.batch([]*string{&msg1, &msg2})
			cd.uploadEventsBatches(batchChan)

			logEventsInput := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &logGroupName,
				LogStreamName: &logStreamName,
			}
			logEventsOutput, _ := cl.GetLogEvents(logEventsInput)

			for _, e := range logEventsOutput.Events {
				delete(storeMsg, *e.Message)
			}
			// At the end of it, all messages should have been seen
			Expect(len(storeMsg)).Should(Equal(0))

		})
	})

})
