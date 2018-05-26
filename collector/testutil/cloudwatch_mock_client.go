// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package testutil

import (
	"strconv"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	log "github.com/sirupsen/logrus"
)

type logEvent struct {
	message   string
	timestamp int64
}

type mockedCloudWatchLogsClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	sequenceToken    int
	logGroupedEvents map[string]map[string][]logEvent
}

// NewMockedCloudWatchLogsClient simulates a very basic aws cloudwatchlogs
// client thats capable of setting and returning stored messages and timestamps.
func NewMockedCloudWatchLogsClient() cloudwatchlogsiface.CloudWatchLogsAPI {
	return &mockedCloudWatchLogsClient{
		logGroupedEvents: map[string]map[string][]logEvent{},
	}
}

func (m *mockedCloudWatchLogsClient) GetLogEvents(input *cloudwatchlogs.GetLogEventsInput) (*cloudwatchlogs.GetLogEventsOutput, error) {
	logGroupName := *input.LogGroupName
	logStreamName := *input.LogStreamName

	log.Infof("Calling mocked cloudwatchlogs GetLogEvents for log group: %s and log stream: %s", logGroupName, logStreamName)
	resp := &cloudwatchlogs.GetLogEventsOutput{}
	outputEvents := m.logGroupedEvents[logGroupName][logStreamName]
	for _, o := range outputEvents {
		resp.Events = append(resp.Events, &cloudwatchlogs.OutputLogEvent{
			Message:   &o.message,
			Timestamp: &o.timestamp,
		})
	}

	return resp, nil
}

func (m *mockedCloudWatchLogsClient) PutLogEvents(input *cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error) {
	log.Infof("Calling mocked cloudwatchlogs PutLogEvents")
	inLogEvents := input.LogEvents
	logGroupName := *input.LogGroupName
	logStreamName := *input.LogStreamName

	for _, le := range inLogEvents {
		logEventInstance := logEvent{
			message:   *le.Message,
			timestamp: *le.Timestamp,
		}
		_, ok := m.logGroupedEvents[logGroupName]
		if !ok {
			m.logGroupedEvents[logGroupName] = map[string][]logEvent{}
		}
		m.logGroupedEvents[logGroupName][logStreamName] = append(m.logGroupedEvents[logGroupName][logStreamName], logEventInstance)
		log.Infof("Stored logevent with message: %s, timestamp: %d under log group: %s and log stream: %s", *le.Message, *le.Timestamp, logGroupName, logStreamName)
	}

	// Only need to return mocked response output
	// TODO: Check for sequence token validity.
	nextSequenceToken := strconv.Itoa(m.sequenceToken)
	m.sequenceToken++
	return &cloudwatchlogs.PutLogEventsOutput{
		NextSequenceToken:     &nextSequenceToken,
		RejectedLogEventsInfo: nil,
	}, nil
}

func (m *mockedCloudWatchLogsClient) DeleteLogGroup(input *cloudwatchlogs.DeleteLogGroupInput) (*cloudwatchlogs.DeleteLogGroupOutput, error) {
	delete(m.logGroupedEvents, *input.LogGroupName)
	return nil, nil
}
