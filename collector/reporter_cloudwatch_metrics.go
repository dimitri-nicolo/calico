// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
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

// StartupLog is looked for by the FVs to check that the reporter has started.
const StartupLog = "starting CloudWatch metric reporter"

func (c *cloudWatchMetricReporter) Start() {
	log.Info(StartupLog)
	go c.run()
}

func (c *cloudWatchMetricReporter) Report(mu MetricUpdate) error {
	lastRuleID := mu.GetLastRuleID()
	if lastRuleID == nil {
		log.WithField("MetricUpdate", mu).Error("no rule id present")
		return fmt.Errorf("invalid metric update")
	}
	log.WithField("MetricUpdate", mu).Debugf("feeding metric update for action: %s", lastRuleID.Action)
	c.aggregator.FeedUpdate(mu)
	return nil
}

func (c *cloudWatchMetricReporter) run() {
	tickerC := jitter.NewTicker(c.updateFrequency, c.updateFrequency/10).C
	for {
		select {
		case <-tickerC:
			c.dispatcher.Dispatch(c.aggregator.Get(), cwCustomNamespace)
		}
	}
}
