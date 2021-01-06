// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"time"

	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/libcalico-go/lib/health"
)

type L7LogGetter interface {
	Get() []*L7Log
}

type L7LogAggregator interface {
	L7LogGetter
	AggregateOver(L7AggregationKind) L7LogAggregator
	FeedUpdate(L7Update) error
}

type l7AggregatorRef struct {
	a L7LogAggregator
	d []LogDispatcher
}

type L7LogReporterInterface interface {
	Start()
	Log(update L7Update) error
}

type L7LogReporter struct {
	dispatchers  map[string]LogDispatcher
	aggregators  []l7AggregatorRef
	flushTrigger <-chan time.Time

	healthAggregator *health.HealthAggregator

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

const (
	l7HealthName     = "l7_reporter"
	l7HealthInterval = 10 * time.Second
)

func NewL7LogReporter(dispatchers map[string]LogDispatcher, flushInterval time.Duration, healthAggregator *health.HealthAggregator) *L7LogReporter {
	return NewL7LogReporterWithShims(dispatchers, jitter.NewTicker(flushInterval, flushInterval/10).Channel(), healthAggregator)
}

func NewL7LogReporterWithShims(dispatchers map[string]LogDispatcher, flushTrigger <-chan time.Time, healthAggregator *health.HealthAggregator) *L7LogReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(l7HealthName, &health.HealthReport{Live: true, Ready: true}, l7HealthInterval*2)
	}
	return &L7LogReporter{
		dispatchers:      dispatchers,
		flushTrigger:     flushTrigger,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
	}
}

func (c *L7LogReporter) AddAggregator(agg L7LogAggregator, dispatchers []string) {
	var ref l7AggregatorRef
	ref.a = agg
	for _, d := range dispatchers {
		dis, ok := c.dispatchers[d]
		if !ok {
			// This is a code error and is unrecoverable.
			log.Panic(fmt.Sprintf("unknown dispatcher \"%s\"", d))
		}
		ref.d = append(ref.d, dis)
	}
	c.aggregators = append(c.aggregators, ref)
}

func (c *L7LogReporter) Start() {
	go c.run()
}

func (c *L7LogReporter) Log(update L7Update) error {
	for _, agg := range c.aggregators {
		if err := agg.a.FeedUpdate(update); err != nil {
			return err
		}
	}
	return nil
}

func (c *L7LogReporter) run() {
	healthTicks := time.NewTicker(l7HealthInterval)
	defer healthTicks.Stop()
	c.reportHealth()
	for {
		log.Debug("L7 reporter loop iteration")

		// TODO(doublek): Stop and flush cases.
		select {
		case <-c.flushTrigger:
			// Fetch from different aggregators and then dispatch them to wherever
			// the flow logs need to end up.
			log.Debug("L7 log flush tick")
			for _, agg := range c.aggregators {
				fl := agg.a.Get()
				log.Debugf("Flush %v L7 logs", len(fl))
				if len(fl) > 0 {
					for _, d := range agg.d {
						log.WithFields(log.Fields{
							"size":       len(fl),
							"dispatcher": d,
						}).Debug("Dispatching log buffer")
						d.Dispatch(fl)
					}
				}
			}
		case <-healthTicks.C:
			// Periodically report current health.
			c.reportHealth()
		}
	}
}

func (c *L7LogReporter) canPublishL7Logs() bool {
	for name, d := range c.dispatchers {
		err := d.Initialize()
		if err != nil {
			log.WithError(err).
				WithField("name", name).
				Error("dispatcher unable to initialize")
			return false
		}
	}
	return true
}

func (c *L7LogReporter) reportHealth() {
	readiness := c.canPublishL7Logs()
	if c.healthAggregator != nil {
		c.healthAggregator.Report(l7HealthName, &health.HealthReport{
			Live:  true,
			Ready: readiness,
		})
	}
}
