// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"github.com/gavv/monotime"
	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/health"
	log "github.com/sirupsen/logrus"
)

type FlowLogGetter interface {
	Get() []*string
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
	Dispatch([]*string) error
}

// cloudWatchReporter implements the MetricsReporter interface.
type cloudWatchReporter struct {
	dispatcher    FlowLogDispatcher
	aggregators   []FlowLogAggregator
	flushInterval time.Duration
	flushTicker   *jitter.Ticker

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
func NewCloudWatchReporter(dispatcher FlowLogDispatcher, flushInterval time.Duration, healthAggregator *health.HealthAggregator) *cloudWatchReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(healthName, &health.HealthReport{Live: true, Ready: true}, healthInterval*2)
	}
	return &cloudWatchReporter{
		dispatcher:       dispatcher,
		flushTicker:      jitter.NewTicker(flushInterval, flushInterval/10),
		flushInterval:    flushInterval,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
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
	// We only produce Flow logs when we know that at least one of the endpoints
	// is a WorkloadEndpoint. Otherwise skip processing.
	if mu.srcEp != nil && mu.dstEp != nil {
		if mu.srcEp.IsHostEndpoint() && mu.dstEp.IsHostEndpoint() {
			log.Debugf("Skipping HEP only update: %v", mu)
			return nil
		}
	}
	for _, agg := range c.aggregators {
		agg.FeedUpdate(mu)
	}
	return nil
}

func (c *cloudWatchReporter) run() {
	healthTicks := time.NewTicker(healthInterval).C
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
		case <-healthTicks:
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
