package usage

import (
	"math"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

// newReportGenerator creates a report generator that observes the event channels and generates basicLicenseUsageReport objects
// on the reports channel based on the observed events.
func newReportGenerator(events events, stop chan struct{}) reportGenerator {
	return reportGenerator{
		events:     events,
		stopIssued: stop,
		reports:    make(chan basicLicenseUsageReport),
		usage: usageState{
			nodes:                make(map[string]*v1.Node),
			initialSyncCompleted: false,
		},
	}
}

func (r *reportGenerator) startGeneratingReports() {
	log.Info("Starting Usage Reporter")
	r.usage.intervalStart = time.Now()

	for {
		select {
		case nodeUpdate := <-r.nodeUpdates:
			r.updateNodesTracked(nodeUpdate)
			r.recalculateCurrentCounts()
			r.recalculateMinCountsForInterval()
			r.recalculateMaxCountsForInterval()

		case <-r.intervalComplete:
			r.flushReportForInterval()
			r.resetInterval()

		case <-r.initialSyncComplete:
			r.markInitialSyncCompleted()
			r.initializeMinCountsForInterval()
			r.initializeMaxCountsForInterval()

		case <-r.stopIssued:
			log.Info("Stopping Usage Reporter")
			return
		}

		log.Debug("Usage Reporter handled input event")
	}
}

func (r *reportGenerator) markInitialSyncCompleted() {
	r.usage.initialSyncCompleted = true
}

func (r *reportGenerator) flushReportForInterval() {
	intervalEnd := time.Now()
	mustSend[basicLicenseUsageReport](r.reports, basicLicenseUsageReport{
		intervalStart: r.usage.intervalStart,
		intervalEnd:   intervalEnd,
		minCounts:     r.usage.minCounts,
		maxCounts:     r.usage.maxCounts,
		complete:      r.usage.initialSyncCompleted,
	})
}
func (r *reportGenerator) resetInterval() {
	r.usage.intervalStart = time.Now()
	r.usage.minCounts = r.usage.currentCounts
	r.usage.maxCounts = r.usage.currentCounts
}

func (r *reportGenerator) updateNodesTracked(event nodeEvent) {
	if event.old != nil {
		delete(r.usage.nodes, event.old.Name)
	}
	if event.new != nil {
		r.usage.nodes[event.new.Name] = event.new
	}
}

func (r *reportGenerator) recalculateCurrentCounts() {
	var vCPUs int
	for _, node := range r.usage.nodes {
		// CPU capacity has the unit of cores, e.g. 4 cores.
		// Make sure we round this value in case we receive a fractional core.
		vCPUs += int(math.Round(node.Status.Capacity.Cpu().AsApproximateFloat64()))
	}

	r.usage.currentCounts.vCPU = vCPUs
	r.usage.currentCounts.nodes = len(r.usage.nodes)
}

func (r *reportGenerator) initializeMinCountsForInterval() {
	r.usage.minCounts = r.usage.currentCounts
}

func (r *reportGenerator) initializeMaxCountsForInterval() {
	r.usage.maxCounts = r.usage.currentCounts
}

func (r *reportGenerator) recalculateMinCountsForInterval() {
	if !r.usage.initialSyncCompleted {
		// If the initial sync has not been completed, then we have not observed the full state of the cluster yet.
		// Therefore, we are not in any position to assert a minimum.
		return
	}

	if r.usage.currentCounts.vCPU < r.usage.minCounts.vCPU {
		r.usage.minCounts.vCPU = r.usage.currentCounts.vCPU
	}

	if r.usage.currentCounts.nodes < r.usage.minCounts.nodes {
		r.usage.minCounts.nodes = r.usage.currentCounts.nodes
	}
}

func (r *reportGenerator) recalculateMaxCountsForInterval() {
	if !r.usage.initialSyncCompleted {
		// If the initial sync has not been completed, then we have not observed the full state of the cluster yet.
		// Therefore, we are not in any position to assert a maximum.
		return
	}

	if r.usage.currentCounts.vCPU > r.usage.maxCounts.vCPU {
		r.usage.maxCounts.vCPU = r.usage.currentCounts.vCPU
	}

	if r.usage.currentCounts.nodes > r.usage.maxCounts.nodes {
		r.usage.maxCounts.nodes = r.usage.currentCounts.nodes
	}
}

type reportGenerator struct {
	events
	stopIssued chan struct{}

	// The singular output channel.
	reports chan basicLicenseUsageReport

	// State collected while monitoring.
	usage usageState
}

type basicLicenseUsageReport struct {
	intervalStart time.Time
	intervalEnd   time.Time
	minCounts     counts
	maxCounts     counts
	complete      bool
}

type events struct {
	// Updates to v1.Node objects in the cluster.
	nodeUpdates chan nodeEvent

	// Synchronization with the datastore.
	initialSyncComplete chan bool

	// Completions of reporting intervals.
	intervalComplete chan bool
}

type usageState struct {
	nodes                map[string]*v1.Node
	intervalStart        time.Time
	currentCounts        counts
	minCounts            counts
	maxCounts            counts
	initialSyncCompleted bool
}

type counts struct {
	vCPU  int
	nodes int
}
