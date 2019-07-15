// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

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

const (
	// max batch size limit associated with PutLogsEventsInput 1MB.
	// setting the constant < 1MB, ~ 0.9MB to account
	// for other attributes in the the inputlogevent & marshalled struct padding.
	eventsBatchSize       = 900 * 1024
	eventsBatchBufferSize = 10
)

type FlowLogFormat string

const (
	FlowLogFormatJSON FlowLogFormat = "json"
	FlowLogFormatFlat FlowLogFormat = "flat"

	cwAPITimeout = 5 * time.Second
	cwNumRetries = 3
)

type eventsBatch []*cloudwatchlogs.InputLogEvent

type cloudWatchEventsBatcher struct {
	size            int
	eventsBatchChan chan eventsBatch
}

func newCloudWatchEventsBatcher(size int, bChan chan eventsBatch) *cloudWatchEventsBatcher {
	return &cloudWatchEventsBatcher{
		size:            size,
		eventsBatchChan: bChan,
	}
}

func (c *cloudWatchEventsBatcher) batch(inputLogs []*string) {
	defer close(c.eventsBatchChan)

	inputEventsSize := 0
	inputEvents := []*cloudwatchlogs.InputLogEvent{}
	inputEventsOffset := 0
	for idx, s := range inputLogs {
		inputEventsSize += len(*s)
		// check against approximately 90% of size limit
		// if greater than flush the batch to eventsBatch channel.
		// and start over again.
		if inputEventsSize > c.size {
			c.eventsBatchChan <- inputEvents[inputEventsOffset:idx]
			inputEventsOffset = idx
			inputEventsSize = 0
		}
		log.Debugf("Constructing cloud watch log event for flowlog: %s", *s)
		inputEvent := &cloudwatchlogs.InputLogEvent{
			Message:   s,
			Timestamp: aws.Int64(time.Now().Unix() * 1000),
		}
		inputEvents = append(inputEvents, inputEvent)
	}

	// done, flush & closing the channel
	c.eventsBatchChan <- inputEvents[inputEventsOffset:len(inputEvents)]
}

// cloudWatchDispatcher implements the LogDispatcher interface.
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
) LogDispatcher {
	if cwl == nil {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		cwl = cloudwatchlogs.New(sess, aws.NewConfig().WithMaxRetries(cwNumRetries))
	}

	cwd := &cloudWatchDispatcher{
		cwl:           cwl,
		logGroupName:  logGroupName,
		logStreamName: logStreamName,
		retentionDays: retentionDays,
	}
	return cwd
}

func (c *cloudWatchDispatcher) uploadEventsBatch(inputLogEvents []*cloudwatchlogs.InputLogEvent) error {
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     inputLogEvents,
		LogGroupName:  aws.String(c.logGroupName),
		LogStreamName: aws.String(c.logStreamName),
	}
	if c.seqToken != "" {
		params.SequenceToken = aws.String(c.seqToken)
	}
	ctx := context.Background()
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

func (c *cloudWatchDispatcher) uploadEventsBatches(eventsBatchChan chan eventsBatch) error {
	for {
		e, more := <-eventsBatchChan
		if !more {
			return nil
		}
		err := c.uploadEventsBatch(e)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *cloudWatchDispatcher) Dispatch(logSlice interface{}) error {
	// Keep pushing as many required <1MB sized inputLogs batches for putLogEvents.
	// Refer: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
	// eventsBatchChan with buffer in case of throttling.
	eventsBatchChan := make(chan eventsBatch, eventsBatchBufferSize)

	// Serialize the logs
	inputLogs := logSlice.([]*FlowLog)
	strBatch := make([]*string, len(inputLogs))
	for idx, f := range inputLogs {
		s := serializeCloudWatchFlowLog(f)
		strBatch[idx] = &s
	}

	b := newCloudWatchEventsBatcher(eventsBatchSize, eventsBatchChan)
	go b.batch(strBatch)

	// Concurrently start uploading the batched events
	err := c.uploadEventsBatches(eventsBatchChan)
	if err != nil {
		return err
	}

	return nil
}

// serializeCloudWatchFlowLog converts FlowLog to a string in our CloudWatch format
func serializeCloudWatchFlowLog(f *FlowLog) string {
	srcIP, dstIP, proto, l4Src, l4Dst := extractPartsFromAggregatedTuple(f.Tuple)
	srcLabels := labelsToString(f.SrcLabels)
	dstLabels := labelsToString(f.DstLabels)

	policyStr := flowLogFieldNotIncluded
	if f.FlowPolicies != nil {
		policies := []string{}
		for p := range f.FlowPolicies {
			policies = append(policies, p)
		}
		policyStr = fmt.Sprintf("%v", policies)
	}
	return fmt.Sprintf("%v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v %v",
		f.StartTime.Unix(), f.EndTime.Unix(),
		f.SrcMeta.Type, f.SrcMeta.Namespace, f.SrcMeta.Name, f.SrcMeta.AggregatedName, srcLabels,
		f.DstMeta.Type, f.DstMeta.Namespace, f.DstMeta.Name, f.DstMeta.AggregatedName, dstLabels,
		srcIP, dstIP, proto, l4Src, l4Dst,
		f.NumFlows, f.NumFlowsStarted, f.NumFlowsCompleted, f.Reporter,
		f.PacketsIn, f.PacketsOut, f.BytesIn, f.BytesOut,
		f.Action, policyStr)
}

func (c *cloudWatchDispatcher) Initialize() error {
	log.Debugf("Initializing seq token")
	if c.cwl == nil {
		log.Debugf("Cloudwatch logs client not initialied")
		return errors.New("Cloudwatch logs client not initialied")
	}
	ctx := context.Background()
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
	ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
	defer cancel()
	_, err = c.cwl.CreateLogStreamWithContext(ctx, createLSInp)
	return err
	if isAWSError(err, cloudwatchlogs.ErrCodeResourceAlreadyExistsException) {
		// LogStream exists already. This cannot happen unless someone manually
		// created the log stream between us verifying if it exists (above) to
		// trying to create it (here).
		log.Debug("Log stream now exists")
		return nil
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
	return fmt.Errorf("Cannot find log stream %v in log group %v", c.logStreamName, c.logGroupName)

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
	ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
	defer cancel()
	_, err = c.cwl.PutRetentionPolicyWithContext(ctx, putRetentionInp)
	return err
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
	ctx, cancel := context.WithTimeout(ctx, cwAPITimeout)
	defer cancel()
	log.Debugf("Describe %v\n", c.logGroupName)
	resp, err := c.cwl.DescribeLogGroupsWithContext(ctx, descLGInp)
	if err != nil {
		log.WithError(err).Errorf("Describe error %v\n", c.logGroupName)
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
	return fmt.Errorf("Could not find log group %v", c.logGroupName)
}
