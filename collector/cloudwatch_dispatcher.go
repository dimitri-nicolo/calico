// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	log "github.com/sirupsen/logrus"
)

type FlowLogFormat string

const (
	FlowLogFormatJSON FlowLogFormat = "json"
	FlowLogFormatFlat FlowLogFormat = "flat"

	LogGroupNamePrefix  = "/tigera/flowlogs"
	LogStreamNameSuffix = "Flowlogs"
)

// cloudWatchDispatcher implements the FlowLogDispatcher interface.
type cloudWatchDispatcher struct {
	logGroupName  string
	logStreamName string
	seqToken      string
	cwl           cloudwatchlogsiface.CloudWatchLogsAPI
}

// NewCloudWatchDispatcher will initialize a session that the aws SDK will use
// to load credentials from the shared credentials file ~/.aws/credentials,
// load your configuration from the shared configuration file ~/.aws/config,
// and create a CloudWatch Logs client.
func NewCloudWatchDispatcher(logGroupName, logStreamName string, cwl cloudwatchlogsiface.CloudWatchLogsAPI) FlowLogDispatcher {
	if cwl == nil {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		cwl = cloudwatchlogs.New(sess)
	}

	cwd := &cloudWatchDispatcher{
		cwl:           cwl,
		logGroupName:  logGroupName,
		logStreamName: logStreamName,
	}

	// TODO(doublek): Add some retries before bailing.
	err := cwd.init()
	if err != nil {
		log.WithError(err).Fatal("Could not initialize sequence token")
		return nil
	}
	return cwd
}

func constructInputEvents(inputLogs []*string) []*cloudwatchlogs.InputLogEvent {
	inputEvents := []*cloudwatchlogs.InputLogEvent{}
	for _, s := range inputLogs {
		log.Debugf("Constructing cloud watch log event for flowlog: %s", *s)
		inputEvent := &cloudwatchlogs.InputLogEvent{
			Message:   s,
			Timestamp: aws.Int64(time.Now().Unix() * 1000),
		}
		inputEvents = append(inputEvents, inputEvent)
	}
	return inputEvents
}

func (c *cloudWatchDispatcher) Dispatch(inputLogs []*string) error {
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     constructInputEvents(inputLogs),
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
		SequenceToken: aws.String(c.seqToken),
	}
	resp, err := c.cwl.PutLogEvents(params)
	if err != nil {
		return err
	}
	if resp.RejectedLogEventsInfo != nil {
		log.Warnf("expired log event end index: %d", resp.RejectedLogEventsInfo.ExpiredLogEventEndIndex)
		log.Warnf("too new log event start index: %d", resp.RejectedLogEventsInfo.TooNewLogEventStartIndex)
		log.Warnf("too old log event end index: %d", resp.RejectedLogEventsInfo.TooOldLogEventEndIndex)
		return errors.New("Got rejected put log events")
	}
	c.seqToken = *resp.NextSequenceToken
	return nil
}

func (c *cloudWatchDispatcher) init() error {
	log.Debugf("Initializing seq token")
	if c.cwl == nil {
		log.Debugf("Cloudwatch logs client not initialied")
		return errors.New("Cloudwatch logs client not initialied")
	}
	err := c.verifyOrCreateLogGroup()
	if err != nil {
		log.WithError(err).Error("Error when verifying/creating log group")
		return err
	}
	ls, err := c.verifyOrCreateLogStream()
	if err != nil {
		log.WithError(err).Error("Error when verifying/creating log stream")
		return err
	}
	if ls.UploadSequenceToken != nil {
		log.Debugf("LS Matched setting Token %v\n", *ls.UploadSequenceToken)
		c.seqToken = *ls.UploadSequenceToken
	}
	return nil
}

func (c *cloudWatchDispatcher) verifyOrCreateLogStream() (*cloudwatchlogs.LogStream, error) {

	ls, err := c.verifyLogStream()
	if err == nil {
		return ls, nil
	}

	// LogStream doesn't exist. Time to create it.
	createLSInp := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
	}
	err = createLSInp.Validate()
	if err != nil {
		return nil, err
	}
	log.WithField("LogStreamName", c.logStreamName).Info("Creating Log stream")
	_, err = c.cwl.CreateLogStream(createLSInp)
	if err != nil {
		log.Debugf("Error when CreateLogStream %v\n", err)
		return nil, err
	}

	ls, err = c.verifyLogStream()
	if err != nil {
		return nil, err
	}
	return ls, nil
}

func (c *cloudWatchDispatcher) verifyLogStream() (*cloudwatchlogs.LogStream, error) {
	// Check if the log stream exists. If it does, return it.
	descLSInp := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(c.logGroupName),
		LogStreamNamePrefix: aws.String(c.logStreamName),
	}
	err := descLSInp.Validate()
	if err != nil {
		return nil, err
	}
	log.Debugf("Describe %v\n", c.logStreamName)
	resp, err := c.cwl.DescribeLogStreams(descLSInp)
	if err != nil {
		log.Debugf("Error when DescribeLogStreams %v\n", err)
		return nil, err
	}
	log.Debugf(resp.GoString())
	for _, ls := range resp.LogStreams {
		log.Debugf(ls.GoString())
		if *ls.LogStreamName == c.logStreamName {
			return ls, nil
		}
	}
	return nil, fmt.Errorf("Cannot find log stream %v in log group %v", c.logStreamName, c.logGroupName)
}

func (c *cloudWatchDispatcher) verifyOrCreateLogGroup() error {

	err := c.verifyLogGroup()
	if err == nil {
		return nil
	}

	// LogGroup doesn't exist. Time to create it.
	createLGInp := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(c.logGroupName),
	}
	err = createLGInp.Validate()
	if err != nil {
		return err
	}
	log.WithField("LogGroupName", c.logGroupName).Info("Creating Log group")
	_, err = c.cwl.CreateLogGroup(createLGInp)
	if err != nil {
		log.Debugf("Error when CreateLogGroup %v\n", err)
		return err
	}

	err = c.verifyLogGroup()
	return err
}

func (c *cloudWatchDispatcher) verifyLogGroup() error {
	descLGInp := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(c.logGroupName),
	}
	err := descLGInp.Validate()
	if err != nil {
		return err
	}
	log.Debugf("Describe %v\n", c.logGroupName)
	resp, err := c.cwl.DescribeLogGroups(descLGInp)
	if err != nil {
		log.Debugf("Error when DescribeLogGroups %v\n", err)
		return err
	}
	log.Debugf(resp.GoString())
	for _, lg := range resp.LogGroups {
		log.Debugf(lg.GoString())
		if *lg.LogGroupName == c.logGroupName {
			log.Debugf("Found log group %v", c.logGroupName)
			return nil
		}
	}
	return fmt.Errorf("Could not find log group %v", c.logGroupName)
}
