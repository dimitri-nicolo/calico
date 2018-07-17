// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	log "github.com/sirupsen/logrus"
)

type FlowLogFormat string

const (
	FlowLogFormatJSON FlowLogFormat = "json"
	FlowLogFormatFlat FlowLogFormat = "flat"

	cwAPITimeout        = 3 * time.Second
	cwRetryWaitInterval = 1 * time.Second
	cwNumRetries        = 5
)

var (
	cwResourceNotFound = errors.New("couldn't find CloudWatch resource.")
)

// cloudWatchDispatcher implements the FlowLogDispatcher interface.
type cloudWatchDispatcher struct {
	cwl           cloudwatchlogsiface.CloudWatchLogsAPI
	logGroupName  string
	logStreamName string
	retentionDays int
	seqToken      string
}

// NewCloudWatchDispatcher will initialize a session that the aws SDK will use
// to load credentials from the shared credentials file ~/.aws/credentials,
// load your configuration from the shared configuration file ~/.aws/config,
// and create a CloudWatch Logs client.
func NewCloudWatchDispatcher(
	logGroupName, logStreamName string, retentionDays int, cwl cloudwatchlogsiface.CloudWatchLogsAPI,
) FlowLogDispatcher {
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
		retentionDays: retentionDays,
	}
	ctx := context.Background()
	cwd.Initialize(ctx)
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

func (c *cloudWatchDispatcher) Dispatch(ctx context.Context, inputLogs []*string) error {
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     constructInputEvents(inputLogs),
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
	}
	if c.seqToken != "" {
		params.SequenceToken = aws.String(c.seqToken)
	}
	putLogEventsFunc := func(params *cloudwatchlogs.PutLogEventsInput) error {
		ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
		defer cancel()
		resp, err := c.cwl.PutLogEventsWithContext(ctx, params)
		if err != nil {
			log.WithFields(log.Fields{"LogGroupName": c.logGroupName, "LogStreamName": c.logStreamName}).WithError(err).Error("PutLogevents")
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
	var err error
	for retry := 0; retry < cwNumRetries; retry++ {
		err = putLogEventsFunc(params)
		if err == nil {
			return nil
		}
		time.Sleep(cwRetryWaitInterval)
	}
	return fmt.Errorf("Retries exceeded when trying to PutLogEvents for LogGroup %v LogStream %v. Error: %v", c.logGroupName, c.logStreamName, err)
}

func (c *cloudWatchDispatcher) Initialize(ctx context.Context) error {
	log.Debugf("Initializing seq token")
	if c.cwl == nil {
		log.Debugf("Cloudwatch logs client not initialied")
		return errors.New("Cloudwatch logs client not initialied")
	}
	err := c.verifyOrCreateLogGroup(ctx)
	if err != nil {
		log.WithError(err).Error("Error when verifying/creating log group")
		return err
	}
	err = c.verifyOrCreateLogStream(ctx)
	if err != nil {
		log.WithError(err).Error("Error when verifying/creating log stream")
		return err
	}
	return nil
}

func (c *cloudWatchDispatcher) setSeqToken(ls *cloudwatchlogs.LogStream) {
	if ls.UploadSequenceToken != nil {
		log.Debugf("LS Matched setting Token %v\n", *ls.UploadSequenceToken)
		c.seqToken = *ls.UploadSequenceToken
	}
}

func (c *cloudWatchDispatcher) verifyOrCreateLogStream(ctx context.Context) error {
	var err error
	err = c.verifyLogStream(ctx)
	if err == nil {
		return nil
	}

	// LogStream doesn't exist. Time to create it.
	createLSInp := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
	}
	err = createLSInp.Validate()
	if err != nil {
		return err
	}
	log.WithField("LogStreamName", c.logStreamName).Info("Creating Log stream")
	createLSFunc := func(createLSInp *cloudwatchlogs.CreateLogStreamInput) error {
		ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
		defer cancel()
		_, err = c.cwl.CreateLogStreamWithContext(ctx, createLSInp)
		return err
	}
	for retry := 0; retry < cwNumRetries; retry++ {
		err = createLSFunc(createLSInp)
		if err == nil {
			break
		} else if isAWSError(err, cloudwatchlogs.ErrCodeResourceAlreadyExistsException) {
			// LogStream exists already. This cannot happen unless someone manually
			// created the log stream between us verifying if it exists (above) to
			// trying to create it (here).
			log.Debug("Log stream now exists")
		}
		time.Sleep(cwRetryWaitInterval)
	}
	if err != nil {
		log.WithError(err).Error("Error when CreateLogStream")
		return err
	}

	err = c.verifyLogStream(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *cloudWatchDispatcher) verifyLogStream(ctx context.Context) error {
	var err error
	// Check if the log stream exists. If it does, return it.
	descLSInp := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(c.logGroupName),
		LogStreamNamePrefix: aws.String(c.logStreamName),
	}
	err = descLSInp.Validate()
	if err != nil {
		return err
	}
	descLogStreamFunc := func(descLSInp *cloudwatchlogs.DescribeLogStreamsInput) error {
		ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
		defer cancel()
		resp, err := c.cwl.DescribeLogStreamsWithContext(ctx, descLSInp)
		if err != nil {
			log.WithError(err).Error("Error when DescribeLogStreams")
			return err
		}
		log.Debugf(resp.GoString())
		for _, ls := range resp.LogStreams {
			log.Debugf(ls.GoString())
			if *ls.LogStreamName == c.logStreamName {
				c.setSeqToken(ls)
				return nil
			}
		}
		return cwResourceNotFound
	}
	for retry := 0; retry < cwNumRetries; retry++ {
		err = descLogStreamFunc(descLSInp)
		if err == nil {
			return nil
		} else if isAWSError(err, cloudwatchlogs.ErrCodeResourceNotFoundException) || err == cwResourceNotFound {
			// LogStream does not exist. We can stop retrying and return the error.
			log.WithFields(log.Fields{"LogGroupName": c.logGroupName, "LogStreamName": c.logStreamName}).Debug("LogStream does not exist")
			return err
		}
		// For all other errors, try to retry.
		log.WithFields(log.Fields{"LogGroupName": c.logGroupName, "LogStreamName": c.logStreamName}).WithError(err).Error("Error when DescribeLogStreams")
		time.Sleep(cwRetryWaitInterval)
	}
	return fmt.Errorf("Retries exceeded. Error: %v", err)
}

func (c *cloudWatchDispatcher) setLogGroupRetention(ctx context.Context) error {
	var err error
	putRetentionInp := &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(c.logGroupName),
		RetentionInDays: aws.Int64(int64(c.retentionDays)),
	}
	err = putRetentionInp.Validate()
	if err != nil {
		log.WithError(err).Warning("Invalid input for PutRetentionPolicy call")
		return err
	}
	putRetPolFunc := func(putRetentionInp *cloudwatchlogs.PutRetentionPolicyInput) error {
		ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
		defer cancel()
		_, err = c.cwl.PutRetentionPolicyWithContext(ctx, putRetentionInp)
		if err != nil {
			log.WithError(err).Warning("Error in PutRetentionPolicy call")
			return err
		}
		return nil
	}
	for retry := 0; retry < cwNumRetries; retry++ {
		err = putRetPolFunc(putRetentionInp)
		if err == nil {
			return nil
		}
		time.Sleep(cwRetryWaitInterval)
	}
	return fmt.Errorf("Retries exceeded when trying to set retention time for LogGroup %v. Error: %v", c.logGroupName, err)
}

func (c *cloudWatchDispatcher) verifyOrCreateLogGroup(ctx context.Context) error {
	var err error
	err = c.verifyLogGroup(ctx)
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
	createLGFunc := func(createLGInp *cloudwatchlogs.CreateLogGroupInput) error {
		ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
		defer cancel()
		_, err = c.cwl.CreateLogGroupWithContext(ctx, createLGInp)
		if err == nil {
			// LogGroup just created by us; set its retention time.
			err = c.setLogGroupRetention(ctx)
			if err != nil {
				return err
			}
		} else if isAWSError(err, cloudwatchlogs.ErrCodeResourceAlreadyExistsException) {
			// LogGroup just created by another ANX node.  Don't set its retention
			// time; there's no need for more than one node to do this, and we can
			// assume that the other node has (or will) set its retention time to
			// whatever the current FelixConfiguration setting says.
			log.Debug("Log group now exists; presume just created by another ANX node")
		} else {
			// Some error other than a creation race.
			log.WithField("LogGroupName", c.logGroupName).WithError(err).Error("Error creating Log group")
			return err
		}
		return nil
	}
	for retry := 0; retry < cwNumRetries; retry++ {
		err = createLGFunc(createLGInp)
		if err == nil {
			return nil
		}
		time.Sleep(cwRetryWaitInterval)
	}
	return err
}

func isAWSError(err error, codes ...string) bool {
	matched := false
	if aerr, ok := err.(awserr.Error); ok {
		errCode := aerr.Code()
		for _, code := range codes {
			if code == errCode {
				matched = true
				break
			}
		}
	}
	return matched
}

func (c *cloudWatchDispatcher) verifyLogGroup(ctx context.Context) error {
	var err error
	descLGInp := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(c.logGroupName),
	}
	err = descLGInp.Validate()
	if err != nil {
		return err
	}
	descLogGroupFunc := func(descLGInp *cloudwatchlogs.DescribeLogGroupsInput) error {
		ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
		defer cancel()
		log.Debugf("Describe %v\n", c.logGroupName)
		resp, err := c.cwl.DescribeLogGroupsWithContext(ctx, descLGInp)
		if err != nil {
			return err
		}
		log.Debugf(resp.GoString())
		for _, lg := range resp.LogGroups {
			log.Debugf(lg.GoString())
			if *lg.LogGroupName == c.logGroupName {
				log.Debugf("Found log group %v", c.logGroupName)
				if lg.RetentionInDays == nil || *lg.RetentionInDays != int64(c.retentionDays) {
					// Log group's retention period does not match the current
					// FelixConfiguration setting, so try to change it to
					// match.  If there is an error here,
					// setLogGroupRetention() will log it, but we don't
					// propagate it any further upwards from this point.  The
					// next ANX node that starts up will try again to align
					// the period with FelixConfiguration.
					c.setLogGroupRetention(ctx)
				}
				return nil
			}
		}
		return cwResourceNotFound
	}
	for retry := 0; retry < cwNumRetries; retry++ {
		err = descLogGroupFunc(descLGInp)
		if err == nil {
			return nil
		} else if isAWSError(err, cloudwatchlogs.ErrCodeResourceNotFoundException) || err == cwResourceNotFound {
			// LogGroup does not exist. We can stop retrying and return the error.
			log.WithField("LogGroupName", c.logGroupName).Debug("Log group does not exists")
			return err
		}
		log.WithField("LogGroupName", c.logGroupName).WithError(err).Error("Error when DescribeLogGroups")
		time.Sleep(cwRetryWaitInterval)
	}
	return fmt.Errorf("Retries exceeded. Error: %v", err)
}
