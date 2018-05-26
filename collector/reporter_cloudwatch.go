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
	dispatcher      FlowLogDispatcher
	aggregator      FlowLogAggregator
	retentionTime   time.Duration
	retentionTicker *jitter.Ticker

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

// NewCloudWatchReporter constructs a FlowLogs MetricsReporter using
// a cloudwatch dispatcher and aggregator.
func NewCloudWatchReporter(retentionTime time.Duration) MetricsReporter {
	return newCloudWatchReporter(NewCloudWatchDispatcher(nil),
		NewCloudWatchAggregator(), retentionTime)
}

func newCloudWatchReporter(d FlowLogDispatcher, a FlowLogAggregator, retentionTime time.Duration) *cloudWatchReporter {
	// Set the ticker interval appropriately, we should be checking at least half of the rention time,
	// or the hard-coded check interval (whichever is smaller).
	tickerInterval := retentionTime / 2
	if checkInterval < tickerInterval {
		tickerInterval = checkInterval
	}
	return &cloudWatchReporter{
		dispatcher:      d,
		aggregator:      a,
		retentionTicker: jitter.NewTicker(tickerInterval, tickerInterval/10),
		retentionTime:   retentionTime,
		timeNowFn:       monotime.Now,
	}
}

func (c *cloudWatchReporter) Start() {
	log.Infof("Starting CloudWatchReporter")
	go c.run()
}

func (c *cloudWatchReporter) Report(mu MetricUpdate) error {
	c.aggregator.FeedUpdate(mu)
	return nil
}

func (c *cloudWatchReporter) run() {
	for {
		select {
		case <-c.retentionTicker.C:
			fl := c.aggregator.Get()
			if len(fl) > 0 {
				log.Infof("Dispatching log buffer of size: %d", len(fl))
				c.dispatcher.Dispatch(fl)
			}
		}
	}
}
