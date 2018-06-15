// Copyright (c) 2018 Tigera, Inc. All rights reserved.

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
	metrics    map[time.Time]float64
}

func NewMockCloudWatchMetricsClient(cmName, cmNamespace string) cloudwatchiface.CloudWatchAPI {
	return &mockCloudWatchMetricsClient{
		name:       cmName,
		namespace:  cmNamespace,
		dimensions: []*cloudwatch.Dimension{},
		metrics:    map[time.Time]float64{},
	}
}

func (m *mockCloudWatchMetricsClient) PutMetricData(input *cloudwatch.PutMetricDataInput) (*cloudwatch.PutMetricDataOutput, error) {
	m.name = *input.MetricData[0].MetricName
	m.name = *input.Namespace
	m.dimensions = input.MetricData[0].Dimensions

	now := time.Now().UTC()
	m.metrics[now] = *input.MetricData[0].Value

	log.Infof("CloudWatch metrics PutMetricData for namespace: %s and metric name: %s. Data: %v, timestamp: %v", m.namespace, m.name, *input.MetricData[0].Value, now)
	return &cloudwatch.PutMetricDataOutput{}, nil
}

func (m *mockCloudWatchMetricsClient) ListMetrics(input *cloudwatch.ListMetricsInput) (*cloudwatch.ListMetricsOutput, error) {
	resp := &cloudwatch.ListMetricsOutput{}

	idx := 0
	for range m.metrics {
		resp.Metrics[idx].Dimensions = m.dimensions
		resp.Metrics[idx].Namespace = &m.namespace
		resp.Metrics[idx].MetricName = &m.name
		resp.NextToken = &fakeToken

		idx++
	}

	log.Infof("CloudWatch metrics ListMetrics for namespace: %s and metric name: %s. DataList: %v", m.namespace, m.name, m.metrics)
	return resp, nil
}
