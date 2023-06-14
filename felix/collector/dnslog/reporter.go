// Copyright (c) 2019-2023 Tigera, Inc. All rights reserved.

package dnslog

import (
	"fmt"
	"time"

	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector/reporter"
	"github.com/projectcalico/calico/felix/jitter"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
)

type aggregatorRef struct {
	a *Aggregator
	d []reporter.LogDispatcher
}

type ReporterInterface interface {
	Start()
	Log(update Update) error
}

type DNSReporter struct {
	dispatchers  map[string]reporter.LogDispatcher
	aggregators  []aggregatorRef
	flushTrigger <-chan time.Time

	healthAggregator *health.HealthAggregator

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration
}

const (
	dnsHealthName     = "DNSReporter"
	dnsHealthInterval = 10 * time.Second
)

// NewReporter constructs a Reporter using a dispatcher and aggregator.
func NewReporter(dispatchers map[string]reporter.LogDispatcher, flushInterval time.Duration, healthAggregator *health.HealthAggregator) *DNSReporter {
	return NewReporterWithShims(dispatchers, jitter.NewTicker(flushInterval, flushInterval/10).Channel(), healthAggregator)
}

func NewReporterWithShims(dispatchers map[string]reporter.LogDispatcher, flushTrigger <-chan time.Time, healthAggregator *health.HealthAggregator) *DNSReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(dnsHealthName, &health.HealthReport{Live: true, Ready: true}, dnsHealthInterval*2)
	}
	return &DNSReporter{
		dispatchers:      dispatchers,
		flushTrigger:     flushTrigger,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
	}
}

func (c *DNSReporter) AddAggregator(agg *Aggregator, dispatchers []string) {
	var ref aggregatorRef
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

func (r *DNSReporter) Start() {
	go r.run()
}

func (r *DNSReporter) Log(update Update) error {
	for _, agg := range r.aggregators {
		if err := agg.a.FeedUpdate(update); err != nil {
			return err
		}
	}
	return nil
}

func (r *DNSReporter) run() {
	healthTicks := time.NewTicker(dnsHealthInterval)
	defer healthTicks.Stop()
	r.reportHealth()
	for {
		log.Debug("DNS reporter loop iteration")

		// TODO(doublek): Stop and flush cases.
		select {
		case <-r.flushTrigger:
			// Fetch from different aggregators and then dispatch them to wherever
			// the flow logs need to end up.
			log.Debug("DNS log flush tick")
			for _, agg := range r.aggregators {
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
			r.reportHealth()
		}
	}
}

func (r *DNSReporter) reportHealth() {
	if r.healthAggregator != nil {
		r.healthAggregator.Report(dnsHealthName, &health.HealthReport{
			Live:  true,
			Ready: r.canPublish(),
		})
	}
}

func (r *DNSReporter) canPublish() bool {
	for name, d := range r.dispatchers {
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
