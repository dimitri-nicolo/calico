// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package flowlog

import (
	"fmt"
	"sync"
	"time"

	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector/reporter"
	"github.com/projectcalico/calico/felix/collector/types/metric"
	"github.com/projectcalico/calico/felix/jitter"
	logutil "github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
)

type aggregatorRef struct {
	a *Aggregator
	d []reporter.LogDispatcher
}

type flowLogAverage struct {
	totalFlows     int
	lastReportTime time.Time
}

// Reporter implements the MetricsReporter interface.
type FlowLogReporter struct {
	dispatchers           map[string]reporter.LogDispatcher
	aggregators           []aggregatorRef
	flushInterval         time.Duration
	flushTicker           jitter.JitterTicker
	hepEnabled            bool
	displayDebugTraceLogs bool

	healthAggregator *health.HealthAggregator
	logOffset        LogOffset

	// Allow the time function to be mocked for test purposes.
	timeNowFn func() time.Duration

	flowLogAvg            *flowLogAverage
	flushIntervalDuration float64
	flowLogAvgMutex       sync.RWMutex
}

const (
	healthName     = "CloudWatchReporter"
	healthInterval = 10 * time.Second
)

func newFlowLogAverage() *flowLogAverage {
	return &flowLogAverage{
		totalFlows:     0,
		lastReportTime: time.Now(),
	}
}

func (fr *FlowLogReporter) updateFlowLogsAvg(numFlows int) {
	fr.flowLogAvgMutex.Lock()
	defer fr.flowLogAvgMutex.Unlock()
	fr.flowLogAvg.totalFlows += numFlows
}

func (fr *FlowLogReporter) GetAndResetFlowLogsAvgPerMinute() (flowsPerMinute float64) {
	fr.flowLogAvgMutex.Lock()
	defer fr.flowLogAvgMutex.Unlock()

	if fr.flowLogAvg == nil || fr.flowLogAvg.totalFlows == 0 {
		return 0
	}

	currentTime := time.Now()
	elapsedTime := currentTime.Sub(fr.flowLogAvg.lastReportTime)

	if elapsedTime.Seconds() < fr.flushIntervalDuration {
		return 0
	}

	flowsPerMinute = float64(fr.flowLogAvg.totalFlows) / elapsedTime.Minutes()
	fr.resetFlowLogsAvg()
	return flowsPerMinute
}

// resetFlowLogsAvg sets the flowAvg fields in FlowLogsReporter.
// This method isn't safe to be used concurrently and the caller should acquire the
// Report.flowLogAvgMutex before calling this method.
func (fr *FlowLogReporter) resetFlowLogsAvg() {
	fr.flowLogAvg.totalFlows = 0
	fr.flowLogAvg.lastReportTime = time.Now()
}

// NewReporter constructs a FlowLogs MetricsReporter using
// a dispatcher and aggregator.
func NewReporter(dispatchers map[string]reporter.LogDispatcher, flushInterval time.Duration, healthAggregator *health.HealthAggregator, hepEnabled, displayDebugTraceLogs bool, logOffset LogOffset) *FlowLogReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(healthName, &health.HealthReport{Live: true, Ready: true}, healthInterval*2)
	}

	return &FlowLogReporter{
		dispatchers:      dispatchers,
		flushTicker:      jitter.NewTicker(flushInterval, flushInterval/10),
		flushInterval:    flushInterval,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
		hepEnabled:       hepEnabled,
		logOffset:        logOffset,

		// Initialize FlowLogAverage struct
		flowLogAvg:            newFlowLogAverage(),
		flushIntervalDuration: flushInterval.Seconds(),
		flowLogAvgMutex:       sync.RWMutex{},
	}
}

func newReporterTest(dispatchers map[string]reporter.LogDispatcher, healthAggregator *health.HealthAggregator, hepEnabled bool, flushTicker jitter.JitterTicker, logOffset LogOffset) *FlowLogReporter {
	if healthAggregator != nil {
		healthAggregator.RegisterReporter(healthName, &health.HealthReport{Live: true, Ready: true}, healthInterval*2)
	}

	return &FlowLogReporter{
		dispatchers:      dispatchers,
		flushTicker:      flushTicker,
		flushInterval:    time.Millisecond,
		timeNowFn:        monotime.Now,
		healthAggregator: healthAggregator,
		hepEnabled:       hepEnabled,
		logOffset:        logOffset,

		// Initialize FlowLogAverage struct
		flowLogAvg:      newFlowLogAverage(),
		flowLogAvgMutex: sync.RWMutex{},
	}
}

func (c *FlowLogReporter) AddAggregator(agg *Aggregator, dispatchers []string) {
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

func (c *FlowLogReporter) Start() {
	log.Info("Starting FlowLogReporter")
	go c.run()
}

func (c *FlowLogReporter) Report(mu metric.Update) error {
	log.Debug("Flow Logs Report got Metric Update")
	if !c.hepEnabled {
		if mu.SrcEp != nil && mu.SrcEp.IsHostEndpoint() {
			mu.SrcEp = nil
		}
		if mu.DstEp != nil && mu.DstEp.IsHostEndpoint() {
			mu.DstEp = nil
		}
	}

	for _, agg := range c.aggregators {
		agg.a.FeedUpdate(&mu)
	}
	return nil
}

func (fr *FlowLogReporter) run() {
	healthTicks := time.NewTicker(healthInterval)
	defer healthTicks.Stop()
	fr.reportHealth()
	for {
		// TODO(doublek): Stop and flush cases.
		select {
		case <-fr.flushTicker.Done():
			log.Debugf("Stopping flush ticker")
			healthTicks.Stop()
			return
		case <-fr.flushTicker.Channel():
			// Fetch from different aggregators and then dispatch them to wherever
			// the flow logs need to end up.
			log.Debug("Flow log flush tick")
			var offsets = fr.logOffset.Read()
			var isBehind = fr.logOffset.IsBehind(offsets)
			var factor = fr.logOffset.GetIncreaseFactor(offsets)

			for _, agg := range fr.aggregators {
				// Evaluate if the external pipeline is stalled
				// and increase / decrease the aggregation level if needed
				newLevel := fr.estimateLevel(agg, AggregationKind(factor), isBehind)

				// Retrieve values from cache and calibrate the cache to the new aggregation level
				fl := agg.a.GetAndCalibrate(newLevel)
				fr.updateFlowLogsAvg(len(fl))
				if len(fl) > 0 {
					// Dispatch logs
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
			fr.reportHealth()
		}
	}
}

func (c *FlowLogReporter) estimateLevel(
	agg aggregatorRef, factor AggregationKind, isBehind bool,
) AggregationKind {
	logutil.Tracef(c.displayDebugTraceLogs, "Evaluate aggregation level. Logs are marked as behind = %v for level %v",
		isBehind, agg.a.CurrentAggregationLevel())
	var newLevel = agg.a.CurrentAggregationLevel()
	if isBehind {
		newLevel = agg.a.CurrentAggregationLevel() + factor
	} else if agg.a.AggregationLevelChanged() {
		newLevel = agg.a.DefaultAggregationLevel()
	}
	logutil.Tracef(c.displayDebugTraceLogs, "Estimate aggregation level to %d", newLevel)
	return newLevel
}

func (c *FlowLogReporter) reportHealth() {
	if c.healthAggregator != nil {
		c.healthAggregator.Report(healthName, &health.HealthReport{
			Live:  true,
			Ready: c.canPublish(),
		})
	}
}

func (c *FlowLogReporter) canPublish() bool {
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
