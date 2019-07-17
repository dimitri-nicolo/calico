// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"time"

	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/jitter"
	"github.com/projectcalico/libcalico-go/lib/health"
)

type DNSLogGetter interface {
	Get() []*DNSLog
}

type DNSLogAggregator interface {
	DNSLogGetter
	IncludeLabels(bool) DNSLogAggregator
	AggregateOver(DNSAggregationKind) DNSLogAggregator
	FeedUpdate(DNSUpdate) error
}

type dnsAggregatorRef struct {
	a DNSLogAggregator
	d []LogDispatcher
}

type DNSLogReporterInterface interface {
	Start()
	Log(update DNSUpdate) error
}

type DNSLogReporter struct {
	dispatchers  map[string]LogDispatcher
	aggregators  []dnsAggregatorRef
	flushTrigger <-chan time.Time

	healthAggregator *health.HealthAggregator

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

const (
	dnsHealthName     = "dns_reporter"
	dnsHealthInterval = 10 * time.Second
)

// NewDNSLogReporter constructs a DNSLogReporter using a dispatcher and aggregator.
func NewDNSLogReporter(dispatchers map[string]LogDispatcher, flushInterval time.Duration, healthAggregator *health.HealthAggregator) *DNSLogReporter {
	return NewDNSLogReporterWithShims(dispatchers, jitter.NewTicker(flushInterval, flushInterval/10).C, healthAggregator)
}

func NewDNSLogReporterWithShims(dispatchers map[string]LogDispatcher, flushTrigger <-chan time.Time, healthAggregator *health.HealthAggregator) *DNSLogReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(dnsHealthName, &health.HealthReport{Live: true, Ready: true}, dnsHealthInterval*2)
	}
	return &DNSLogReporter{
		dispatchers:      dispatchers,
		flushTrigger:     flushTrigger,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
	}
}

func (c *DNSLogReporter) AddAggregator(agg DNSLogAggregator, dispatchers []string) {
	var ref dnsAggregatorRef
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

func (c *DNSLogReporter) Start() {
	go c.run()
}

func (c *DNSLogReporter) Log(update DNSUpdate) error {
	for _, agg := range c.aggregators {
		if err := agg.a.FeedUpdate(update); err != nil {
			return err
		}
	}
	return nil
}

func (c *DNSLogReporter) run() {
	healthTicks := time.NewTicker(dnsHealthInterval)
	defer healthTicks.Stop()
	c.reportHealth()
	for {
		log.Debug("DNS reporter loop iteration")

		// TODO(doublek): Stop and flush cases.
		select {
		case <-c.flushTrigger:
			// Fetch from different aggregators and then dispatch them to wherever
			// the flow logs need to end up.
			log.Debug("DNS log flush tick")
			for _, agg := range c.aggregators {
				fl := agg.a.Get()
				log.Debugf("Flush %v DNS logs", len(fl))
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

func (c *DNSLogReporter) canPublishDNSLogs() bool {
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

func (c *DNSLogReporter) reportHealth() {
	readiness := c.canPublishDNSLogs()
	if c.healthAggregator != nil {
		c.healthAggregator.Report(dnsHealthName, &health.HealthReport{
			Live:  true,
			Ready: readiness,
		})
	}
}
