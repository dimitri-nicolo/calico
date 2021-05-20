// Copyright (c) 2018, 2021 Tigera, Inc. All rights reserved.

package testutil

import (
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	log "github.com/sirupsen/logrus"
)

var (
	fakeToken = "somerandom-token"
)

type mockCloudWatchMetricsClient struct {
	cloudwatchiface.CloudWatchAPI
	name       string
	namespace  string
	dimensions []*cloudwatch.Dimension
}

func NewMockCloudWatchMetricsClient(cmName, cmNamespace string, uf time.Duration) cloudwatchiface.CloudWatchAPI {
	return &mockCloudWatchMetricsClient{
		name:       cmName,
		namespace:  cmNamespace,
		dimensions: []*cloudwatch.Dimension{},
	}
}

func (m *mockCloudWatchMetricsClient) PutMetricData(input *cloudwatch.PutMetricDataInput) (*cloudwatch.PutMetricDataOutput, error) {
	m.name = *input.MetricData[0].MetricName
	m.name = *input.Namespace
	m.dimensions = input.MetricData[0].Dimensions

	log.Infof("CloudWatch metrics PutMetricData for namespace: %s and metric name: %s. Data: %v", m.namespace, m.name, *input.MetricData[0].Value)
	return &cloudwatch.PutMetricDataOutput{}, nil
}

func (m *mockCloudWatchMetricsClient) ListMetrics(input *cloudwatch.ListMetricsInput) (*cloudwatch.ListMetricsOutput, error) {
	log.Infof("CloudWatch metrics ListMetrics for namespace: %s and metric name: %s", m.namespace, m.name)

	return &cloudwatch.ListMetricsOutput{
		Metrics: []*cloudwatch.Metric{{
			Dimensions: m.dimensions,
			Namespace:  &m.namespace,
			MetricName: &m.name,
		},
		},
		NextToken: &fakeToken,
	}, nil
}
