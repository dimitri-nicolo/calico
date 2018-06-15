// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
)

const (
	dpMetricName      = "Denied Packets"
	defaultDPUnit     = cloudwatch.StandardUnitCount
	cwCustomNamespace = "Tigera Metrics"
)

type MetricData struct {
	Name       string
	Dimensions map[string]string
	Value      float64
	Unit       string
	Timestamp  time.Time
}

type MetricAggregator interface {
	Get() []MetricData
	FeedUpdate(MetricUpdate) error
}

type MetricDispatcher interface {
	Dispatch([]MetricData, string) error
}

type cloudWatchMetricReporter struct {
	dispatcher      MetricDispatcher
	aggregator      MetricAggregator
	updateFrequency time.Duration
	timeNowFn       func() time.Duration
}

func NewCloudWatchMetricsReporter(updateFrequency time.Duration, clusterGUID string) MetricsReporter {
	return newCloudWatchMetricReporter(NewCloudWatchMetricsDispatcher(nil), NewCloudWatchMetricsAggregator(clusterGUID), updateFrequency)
}

func newCloudWatchMetricReporter(dis MetricDispatcher, agg MetricAggregator, uf time.Duration) *cloudWatchMetricReporter {
	return &cloudWatchMetricReporter{
		dispatcher:      dis,
		aggregator:      agg,
		updateFrequency: uf,
		timeNowFn:       monotime.Now,
	}
}

func (c *cloudWatchMetricReporter) Start() {
	log.Info("starting CloudWatch metric reporter")
	go c.run()
}

func (c *cloudWatchMetricReporter) Report(mu MetricUpdate) error {
	log.WithField("MetricUpdate", mu).Debugf("feeding metric update for action: %s", mu.ruleID.Action)
	c.aggregator.FeedUpdate(mu)
	return nil
}

func (c *cloudWatchMetricReporter) run() {
	for {
		select {
		case <-jitter.NewTicker(c.updateFrequency, c.updateFrequency/10).C:
			c.dispatcher.Dispatch(c.aggregator.Get(), cwCustomNamespace)
		}
	}
}
