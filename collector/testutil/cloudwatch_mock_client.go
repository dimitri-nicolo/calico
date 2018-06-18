// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package testutil

import (
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	logGroupName     string
	logStreamName    string
	sequenceToken    int
	logGroupedEvents map[string]map[string][]logEvent
}

// NewMockedCloudWatchLogsClient simulates a very basic aws cloudwatchlogs
// client thats capable of setting and returning stored messages and timestamps.
func NewMockedCloudWatchLogsClient(logGroupName string) cloudwatchlogsiface.CloudWatchLogsAPI {
	return &mockedCloudWatchLogsClient{
		logGroupedEvents: map[string]map[string][]logEvent{},
		logGroupName:     logGroupName,
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
			Message:   aws.String(o.message),
			Timestamp: aws.Int64(o.timestamp),
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

func (m *mockedCloudWatchLogsClient) DescribeLogGroups(input *cloudwatchlogs.DescribeLogGroupsInput) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	if !strings.HasPrefix(m.logGroupName, *input.LogGroupNamePrefix) {
		return nil, awserr.New(cloudwatchlogs.ErrCodeResourceNotFoundException, "Log stream resource not found", errors.New("Log stream res not found"))
	}
	// For now always no next token indicating a newly created log stream
	dlgo := &cloudwatchlogs.DescribeLogGroupsOutput{
		LogGroups: []*cloudwatchlogs.LogGroup{
			&cloudwatchlogs.LogGroup{LogGroupName: aws.String(m.logGroupName)},
		},
	}
	return dlgo, nil
}

func (m *mockedCloudWatchLogsClient) DeleteLogGroup(input *cloudwatchlogs.DeleteLogGroupInput) (*cloudwatchlogs.DeleteLogGroupOutput, error) {
	delete(m.logGroupedEvents, *input.LogGroupName)
	return nil, nil
}

func (m *mockedCloudWatchLogsClient) CreateLogStream(input *cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	m.logStreamName = *input.LogStreamName
	return &cloudwatchlogs.CreateLogStreamOutput{}, nil
}

func (m *mockedCloudWatchLogsClient) DescribeLogStreams(input *cloudwatchlogs.DescribeLogStreamsInput) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	if m.logGroupName != *input.LogGroupName {
		return nil, awserr.New(cloudwatchlogs.ErrCodeResourceNotFoundException, "Log group Resource not found", errors.New("Log group res not found"))
	}
	if !strings.HasPrefix(m.logStreamName, *input.LogStreamNamePrefix) {
		return nil, awserr.New(cloudwatchlogs.ErrCodeResourceNotFoundException, "Log stream resource not found", errors.New("Log stream res not found"))
	}
	// For now always no next token indicating a newly created log stream
	dlso := &cloudwatchlogs.DescribeLogStreamsOutput{
		LogStreams: []*cloudwatchlogs.LogStream{
			&cloudwatchlogs.LogStream{LogStreamName: aws.String(m.logStreamName)},
		},
	}
	return dlso, nil
}
