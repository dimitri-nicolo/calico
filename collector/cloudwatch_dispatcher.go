// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	log "github.com/sirupsen/logrus"
)

// cloudWatchDispatcher implements the FlowLogDispatcher interface.
type cloudWatchDispatcher struct {
	cwl      cloudwatchlogsiface.CloudWatchLogsAPI
	seqToken string
}

// NewCloudWatchDispatcher will initialize a session that the aws SDK will use
// to load credentials from the shared credentials file ~/.aws/credentials,
// load your configuration from the shared configuration file ~/.aws/config,
// and create a CloudWatch Logs client.
// TODO: Update to process aws.Config as a param
func NewCloudWatchDispatcher(cwl cloudwatchlogsiface.CloudWatchLogsAPI) FlowLogDispatcher {
	if cwl == nil {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		cwl = cloudwatchlogs.New(sess)
	}

	return &cloudWatchDispatcher{
		cwl: cwl,
	}
}

func constructInputEvents(inputLogs []*string) []*cloudwatchlogs.InputLogEvent {
	inputEvents := []*cloudwatchlogs.InputLogEvent{}
	for _, s := range inputLogs {
		log.Infof("Constructing cloud watch log event for flowlog: %s", *s)
		inputEvent := &cloudwatchlogs.InputLogEvent{
			Message:   s,
			Timestamp: aws.Int64(time.Now().UnixNano()),
		}
		inputEvents = append(inputEvents, inputEvent)
	}
	return inputEvents
}

func (c *cloudWatchDispatcher) Dispatch(inputLogs []*string) error {
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents: constructInputEvents(inputLogs),
		// TODO: Decide on naming convention
		LogGroupName: aws.String("test-group"),
		// TODO: Decide on naming convention
		LogStreamName: aws.String("test-stream"),
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
	}
	c.seqToken = *resp.NextSequenceToken
	return nil
}
