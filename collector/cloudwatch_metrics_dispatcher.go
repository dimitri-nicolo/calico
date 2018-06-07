// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

type cloudWatchMetricsDispatcher struct {
	cwAPI cloudwatchiface.CloudWatchAPI
}

func NewCloudWatchMetricsDispatcher(cwAPI cloudwatchiface.CloudWatchAPI) MetricDispatcher {
	if cwAPI == nil {
		// Initialize a session that the SDK uses to load
		// credentials from the shared credentials file ~/.aws/credentials
		// and configuration from the shared configuration file ~/.aws/config.
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		// Create new cloudwatch client.
		cwAPI = cloudwatch.New(sess)
	}
	return &cloudWatchMetricsDispatcher{
		cwAPI: cwAPI,
	}
}

func (c *cloudWatchMetricsDispatcher) Dispatch(dp []MetricData, namespace string) error {

	for _, metric := range dp {
		metricData := cloudwatch.MetricDatum{
			MetricName: aws.String(metric.Name),
			Unit:       aws.String(metric.Unit),
			Value:      aws.Float64(metric.Value),
			Dimensions: []*cloudwatch.Dimension{},
		}

		for k, v := range metric.Dimensions {
			metricData.Dimensions = append(metricData.Dimensions, &cloudwatch.Dimension{
				Name:  aws.String(k),
				Value: aws.String(v),
			})
		}

		for retry := 0; retry < 5; retry++ {
			result, err := c.cwAPI.PutMetricData(&cloudwatch.PutMetricDataInput{
				MetricData: []*cloudwatch.MetricDatum{
					&metricData,
				},
				Namespace: aws.String(namespace),
			})

			if err != nil {
				// Failed to push metric data, so sleep for a second and retry.
				log.WithField("Metric", metricData).Errorf("failed to push metrics to CloudWatch: %s. Retry: %d", err, retry)
				time.Sleep(time.Second)
			} else {
				log.WithFields(log.Fields{"Metric": metricData, "Result": result}).Debug("successfully pushed metric data to CloudWatch")
				break
			}
		}
	}

	return nil
}
