// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/health"
)

type FlowLogGetter interface {
	Get() []*FlowLog
}

type FlowLogAggregator interface {
	FlowLogGetter
	IncludeLabels(bool) FlowLogAggregator
	AggregateOver(AggregationKind) FlowLogAggregator
	ForAction(rules.RuleAction) FlowLogAggregator
	FeedUpdate(MetricUpdate) error
}

type FlowLogDispatcher interface {
	Initialize() error
	Dispatch([]*FlowLog) error
}

// cloudWatchReporter implements the MetricsReporter interface.
type cloudWatchReporter struct {
	dispatcher    FlowLogDispatcher
	aggregators   []FlowLogAggregator
	flushInterval time.Duration
	flushTicker   *jitter.Ticker
	hepEnabled    bool

	healthAggregator *health.HealthAggregator

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

const (
	healthName     = "cloud_watch_reporter"
	healthInterval = 10 * time.Second
)

// NewCloudWatchReporter constructs a FlowLogs MetricsReporter using
// a cloudwatch dispatcher and aggregator.
func NewCloudWatchReporter(dispatcher FlowLogDispatcher, flushInterval time.Duration, healthAggregator *health.HealthAggregator, hepEnabled bool) *cloudWatchReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(healthName, &health.HealthReport{Live: true, Ready: true}, healthInterval*2)
	}
	return &cloudWatchReporter{
		dispatcher:       dispatcher,
		flushTicker:      jitter.NewTicker(flushInterval, flushInterval/10),
		flushInterval:    flushInterval,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
		hepEnabled:       hepEnabled,
	}
}

func (c *cloudWatchReporter) AddAggregator(agg FlowLogAggregator) {
	c.aggregators = append(c.aggregators, agg)
}

func (c *cloudWatchReporter) Start() {
	log.Info("Starting CloudWatchReporter")
	go c.run()
}

func (c *cloudWatchReporter) Report(mu MetricUpdate) error {
	if !c.hepEnabled {
		if mu.srcEp != nil && mu.srcEp.IsHostEndpoint() {
			mu.srcEp = nil
		}
		if mu.dstEp != nil && mu.dstEp.IsHostEndpoint() {
			mu.dstEp = nil
		}
	}
	for _, agg := range c.aggregators {
		agg.FeedUpdate(mu)
	}
	return nil
}

func (c *cloudWatchReporter) run() {
	healthTicks := time.NewTicker(healthInterval)
	defer healthTicks.Stop()
	c.reportHealth()
	for {
		// TODO(doublek): Stop and flush cases.
		select {
		case <-c.flushTicker.C:
			// Fetch from different aggregators and then dispatch them to wherever
			// the flow logs need to end up.
			for _, agg := range c.aggregators {
				fl := agg.Get()
				if len(fl) > 0 {
					log.Debugf("Dispatching log buffer of size: %d", len(fl))
					c.dispatcher.Dispatch(fl)
				}
			}
		case <-healthTicks.C:
			// Periodically report current health.
			c.reportHealth()
		}
	}
}

func (c *cloudWatchReporter) canPublishFlowLogs() bool {
	err := c.dispatcher.Initialize()
	if err != nil {
		log.WithError(err).Error("Error when verifying/creating CloudWatch resources.")
		return false
	}
	return true
}

func (c *cloudWatchReporter) reportHealth() {
	readiness := c.canPublishFlowLogs()
	if c.healthAggregator != nil {
		c.healthAggregator.Report(healthName, &health.HealthReport{
			Live:  true,
			Ready: readiness,
		})
	}
}
