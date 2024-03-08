// Copyright 2019 Tigera Inc. All rights reserved.
package main

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

const (
	stateFilePfx   = "elf-state"
	stateFileSep   = "_"
	rubyFileSep    = "/"
	rubyFileSepSub = "-"
)

// Setup AWS session to cloudwatch logs service, returns session handler.
func AwsSetupLogSession() *cloudwatchlogs.CloudWatchLogs {
	cwSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigDisable,
	}))

	cwLogs := cloudwatchlogs.New(cwSession)

	return cwLogs
}

// Using AWS session handler, cloudwatch logs specifics and a timestamp, return a log token.
func AwsGetStateFileWithToken(logs *cloudwatchlogs.CloudWatchLogs, group, prefix string, startTime int64) (map[string]string, error) {
	results := make(map[string]string)

	streams, err := getLogStreams(logs, group, prefix)
	if err != nil {
		return nil, err
	}

	for _, stream := range streams {
		token, err := getToken(logs, group, *stream, startTime)
		if err != nil {
			return nil, err
		}

		replaced := strings.ReplaceAll(*stream, rubyFileSep, rubyFileSepSub)
		stateFile := stateFilePfx + stateFileSep + replaced
		results[stateFile] = token
	}

	return results, nil
}

// Wrapper over cloudwatchlogs description, this function returns a slice of log-stream name using log-group name and stream prefix.
func getLogStreams(logs *cloudwatchlogs.CloudWatchLogs, groupName, streamPrefix string) ([]*string, error) {
	var streams []*string

	// Logstream name is dynamic for each EKS deployment. We use LogStreamName prefix to gather the actual stream name.
	resp, err := logs.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(groupName),
		LogStreamNamePrefix: aws.String(streamPrefix),
	})
	if err != nil {
		return streams, err
	}

	for _, stream := range resp.LogStreams {
		streams = append(streams, stream.LogStreamName)
	}
	return streams, nil
}

// Get cloudwatchlogs token pointing to the log stream forward.
func getToken(logs *cloudwatchlogs.CloudWatchLogs, group, stream string, startTime int64) (string, error) {
	resp, err := logs.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		Limit:         aws.Int64(1),
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
		StartTime:     aws.Int64(startTime),
	})
	if err != nil {
		return "", err
	}

	return *resp.NextForwardToken, nil
}
