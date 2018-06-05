// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package collector

import (
	"time"

	"github.com/gavv/monotime"
	"github.com/projectcalico/felix/jitter"
	log "github.com/sirupsen/logrus"
)

type FlowLogGetter interface {
	Get() []*string
}

type FlowLogAggregator interface {
	FlowLogGetter
	FeedUpdate(MetricUpdate) error
}

type FlowLogDispatcher interface {
	Dispatch([]*string) error
}

// cloudWatchReporter implements the MetricsReporter interface.
type cloudWatchReporter struct {
	dispatcher    FlowLogDispatcher
	aggregators   []FlowLogAggregator
	flushInterval time.Duration
	flushTicker   *jitter.Ticker

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

// NewCloudWatchReporter constructs a FlowLogs MetricsReporter using
// a cloudwatch dispatcher and aggregator.
func NewCloudWatchReporter(dispatcher FlowLogDispatcher, flushInterval time.Duration) *cloudWatchReporter {
	// Set the ticker interval appropriately, we should be checking at least half of the flush time,
	// or the hard-coded check interval (whichever is smaller).
	tickerInterval := flushInterval / 2
	if checkInterval < tickerInterval {
		tickerInterval = checkInterval
	}
	return &cloudWatchReporter{
		dispatcher:    dispatcher,
		flushTicker:   jitter.NewTicker(tickerInterval, tickerInterval/10),
		flushInterval: flushInterval,
		timeNowFn:     monotime.Now,
	}
}

func (c *cloudWatchReporter) AddAggregator(agg FlowLogAggregator) {
	c.aggregators = append(c.aggregators, agg)
}

func (c *cloudWatchReporter) Start() {
	log.Infof("Starting CloudWatchReporter")
	go c.run()
}

func (c *cloudWatchReporter) Report(mu MetricUpdate) error {
	for _, agg := range c.aggregators {
		agg.FeedUpdate(mu)
	}
	return nil
}

func (c *cloudWatchReporter) run() {
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
		}
	}
}
