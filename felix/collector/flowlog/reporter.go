// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package flowlog

import (
	"fmt"
	"sync"
	"time"

	"github.com/gavv/monotime"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/collector/types"
	"github.com/projectcalico/calico/felix/collector/types/metric"
	"github.com/projectcalico/calico/felix/jitter"
	logutil "github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
)

const (
	healthName     = "CloudWatchReporter"
	healthInterval = 10 * time.Second
)

type aggregatorRef struct {
	a *Aggregator
	d []types.Reporter
}

type flowLogAverage struct {
	totalFlows     int
	lastReportTime time.Time
}

func newFlowLogAverage() *flowLogAverage {
	return &flowLogAverage{
		totalFlows:     0,
		lastReportTime: time.Now(),
	}
}

// Reporter implements the Reporter interface.
type FlowLogReporter struct {
	dispatchers           map[string]types.Reporter
	aggregators           []aggregatorRef
	flushInterval         time.Duration
	flushTicker           jitter.TickerInterface
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

// NewReporter constructs a FlowLogs MetricsReporter using
// a dispatcher and aggregator.
func NewReporter(dispatchers map[string]types.Reporter, flushInterval time.Duration, healthAggregator *health.HealthAggregator, hepEnabled, displayDebugTraceLogs bool, logOffset LogOffset) *FlowLogReporter {
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

func newReporterTest(
	dispatchers map[string]types.Reporter,
	healthAggregator *health.HealthAggregator,
	hepEnabled bool,
	flushTicker jitter.TickerInterface,
	logOffset LogOffset,
) *FlowLogReporter {
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

func (r *FlowLogReporter) updateFlowLogsAvg(numFlows int) {
	r.flowLogAvgMutex.Lock()
	defer r.flowLogAvgMutex.Unlock()
	r.flowLogAvg.totalFlows += numFlows
}

func (r *FlowLogReporter) GetAndResetFlowLogsAvgPerMinute() (flowsPerMinute float64) {
	r.flowLogAvgMutex.Lock()
	defer r.flowLogAvgMutex.Unlock()

	if r.flowLogAvg == nil || r.flowLogAvg.totalFlows == 0 {
		return 0
	}

	currentTime := time.Now()
	elapsedTime := currentTime.Sub(r.flowLogAvg.lastReportTime)

	if elapsedTime.Seconds() < r.flushIntervalDuration {
		return 0
	}

	flowsPerMinute = float64(r.flowLogAvg.totalFlows) / elapsedTime.Minutes()
	r.resetFlowLogsAvg()
	return flowsPerMinute
}

// resetFlowLogsAvg sets the flowAvg fields in FlowLogsReporter.
// This method isn't safe to be used concurrently and the caller should acquire the
// Report.flowLogAvgMutex before calling this method.
func (r *FlowLogReporter) resetFlowLogsAvg() {
	r.flowLogAvg.totalFlows = 0
	r.flowLogAvg.lastReportTime = time.Now()
}

func (r *FlowLogReporter) AddAggregator(agg *Aggregator, dispatchers []string) {
	var ref aggregatorRef
	ref.a = agg
	for _, d := range dispatchers {
		dis, ok := r.dispatchers[d]
		if !ok {
			// This is a code error and is unrecoverable.
			log.Panic(fmt.Sprintf("unknown dispatcher \"%s\"", d))
		}
		ref.d = append(ref.d, dis)
	}
	r.aggregators = append(r.aggregators, ref)
}

func (r *FlowLogReporter) Start() error {
	log.Info("Starting FlowLogReporter")

	// Try to start the dispatchers now to give early feedback.  We'll retry
	// on the health tick on failure.
	ready := r.maybeStartDispatchers()
	r.reportHealth(ready)

	go r.run()
	return nil
}

func (r *FlowLogReporter) Report(u interface{}) error {
	mu, ok := u.(metric.Update)
	if !ok {
		return fmt.Errorf("invalid metric update")
	}
	log.Debug("Flow Logs Report got Metric Update")
	if !r.hepEnabled {
		if mu.SrcEp != nil && mu.SrcEp.IsHostEndpoint() {
			mu.SrcEp = nil
		}
		if mu.DstEp != nil && mu.DstEp.IsHostEndpoint() {
			mu.DstEp = nil
		}
	}

	for _, agg := range r.aggregators {
		if err := agg.a.FeedUpdate(&mu); err != nil {
			log.WithError(err).Debug("failed to feed metric update")
		}
	}
	return nil
}

func (r *FlowLogReporter) run() {
	healthTicks := time.NewTicker(healthInterval)
	defer healthTicks.Stop()
	for {
		// TODO(doublek): Stop and flush cases.
		select {
		case <-r.flushTicker.Done():
			log.Debugf("Stopping flush ticker")
			healthTicks.Stop()
			return
		case <-r.flushTicker.Channel():
			// Fetch from different aggregators and then dispatch them to wherever
			// the flow logs need to end up.
			log.Debug("Flow log flush tick")
			var offsets = r.logOffset.Read()
			var isBehind = r.logOffset.IsBehind(offsets)
			var factor = r.logOffset.GetIncreaseFactor(offsets)

			for _, agg := range r.aggregators {
				// Evaluate if the external pipeline is stalled
				// and increase / decrease the aggregation level if needed
				newLevel := r.estimateLevel(agg, AggregationKind(factor), isBehind)

				// Retrieve values from cache and calibrate the cache to the new aggregation level
				fl := agg.a.GetAndCalibrate(newLevel)
				r.updateFlowLogsAvg(len(fl))
				if len(fl) > 0 {
					// Dispatch logs
					for _, d := range agg.d {
						if log.IsLevelEnabled(log.DebugLevel) {
							log.WithFields(log.Fields{
								"size":       len(fl),
								"dispatcher": fmt.Sprintf("%T", d),
							}).Debug("Dispatching log buffer")
						}
						if err := d.Report(fl); err != nil {
							log.WithError(err).Debug("failed to dispatch flow log")
						}
					}
				}
			}
		case <-healthTicks.C:
			// Periodically report current health.
			ready := r.maybeStartDispatchers()
			r.reportHealth(ready)
		}
	}
}

func (r *FlowLogReporter) estimateLevel(
	agg aggregatorRef, factor AggregationKind, isBehind bool,
) AggregationKind {
	logutil.Tracef(r.displayDebugTraceLogs, "Evaluate aggregation level. Logs are marked as behind = %v for level %v",
		isBehind, agg.a.CurrentAggregationLevel())
	var newLevel = agg.a.CurrentAggregationLevel()
	if isBehind {
		newLevel = agg.a.CurrentAggregationLevel() + factor
	} else if agg.a.AggregationLevelChanged() {
		newLevel = agg.a.DefaultAggregationLevel()
	}
	logutil.Tracef(r.displayDebugTraceLogs, "Estimate aggregation level to %d", newLevel)
	return newLevel
}

func (r *FlowLogReporter) reportHealth(ready bool) {
	if r.healthAggregator != nil {
		r.healthAggregator.Report(healthName, &health.HealthReport{
			Live:  true,
			Ready: ready,
		})
	}
}

func (r *FlowLogReporter) maybeStartDispatchers() bool {
	for name, d := range r.dispatchers {
		err := d.Start()
		if err != nil {
			log.WithError(err).
				WithField("name", name).
				Error("dispatcher unable to initialize")
			return false
		}
	}
	return true
}
