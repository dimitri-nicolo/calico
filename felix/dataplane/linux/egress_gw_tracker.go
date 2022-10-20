// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/jitter"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

var (
	numEGWPollsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "felix_egress_gateway_remote_polls",
		Help: "Number of remote egress gateways that are being actively polled.",
	}, []string{"status"})
)

func init() {
	prometheus.MustRegister(numEGWPollsGauge)
}

type EgressGWTracker struct {
	ipSetIDToGateways map[string]gatewaysByIP
	dirtyEgressIPSet  set.Typed[string]

	nextPollerNonce     int
	pollers             map[egwPollerID]*egwPoller
	pollInterval        time.Duration
	minPollFailureCount int

	healthReportC chan<- EGWHealthReport

	fastRetryTicker *jitter.Ticker
	context         context.Context
}

func NewEgressGWTracker(ctx context.Context, healthReportC chan<- EGWHealthReport, pollInterval time.Duration, pollFailCount int) *EgressGWTracker {
	return &EgressGWTracker{
		ipSetIDToGateways: map[string]gatewaysByIP{},
		dirtyEgressIPSet:  set.New[string](),

		pollers:             map[egwPollerID]*egwPoller{},
		pollInterval:        pollInterval,
		minPollFailureCount: pollFailCount,

		healthReportC: healthReportC,
		context:       ctx,
	}
}

func (m *EgressGWTracker) OnIPSetDeltaUpdate(msg *proto.IPSetDeltaUpdate) {
	gateways, found := m.ipSetIDToGateways[msg.Id]
	if !found {
		log.WithField("msg", msg).Debug("Ignoring IP set delta update (not a set we're tracking)")
		return
	}
	log.Infof("EgressIP set delta update: id=%v removed=%v added=%v", msg.Id, msg.RemovedMembers, msg.AddedMembers)

	// The member string contains cidr,deletionTimestamp, and so we could get the same cidr in membersAdded
	// and in membersRemoved, with different timestamps. For this reason, process the removes before the adds.
	for _, mStr := range msg.RemovedMembers {
		member, err := parseEGWIPSetMember(mStr)
		if err != nil {
			log.WithError(err).Errorf("BUG: error parsing ip set member from member string %s", mStr)
			continue
		}
		m.stopPollerIfRunning(msg.Id, member.addr)
		delete(gateways, member.addr)
	}

	for _, mStr := range msg.AddedMembers {
		member, err := parseEGWIPSetMember(mStr)
		if err != nil {
			log.WithError(err).Errorf("BUG: error parsing ip set member from member string %s", mStr)
			continue
		}
		gateways[member.addr] = member
		if member.healthPort != 0 {
			m.ensurePollerRunning(msg.Id, member.addr, member.healthPort)
		} else {
			m.stopPollerIfRunning(msg.Id, member.addr)
		}
	}
	m.markSetDirty(msg.Id)
}

func (m *EgressGWTracker) OnIPSetUpdate(msg *proto.IPSetUpdate) {
	if msg.Type != proto.IPSetUpdate_EGRESS_IP {
		log.WithField("msg", msg).Debug("Ignore non-EGW IP set update")
		return
	}

	oldGWs := m.ipSetIDToGateways[msg.Id]

	log.Infof("Update whole EgressIP set: msg=%v", msg)
	newGWs := make(gatewaysByIP)
	for _, mStr := range msg.Members {
		member, err := parseEGWIPSetMember(mStr)
		if err != nil {
			log.WithError(err).Errorf("BUG: error parsing details from memberStr: %s", mStr)
			continue
		}
		newGWs[member.addr] = member
		if member.healthPort != 0 {
			m.ensurePollerRunning(msg.Id, member.addr, member.healthPort)
		} else {
			m.stopPollerIfRunning(msg.Id, member.addr)
		}
	}

	for k, oldGW := range oldGWs {
		if _, ok := newGWs[k]; ok {
			continue
		}
		m.stopPollerIfRunning(msg.Id, oldGW.addr)
	}

	m.ipSetIDToGateways[msg.Id] = newGWs
	m.markSetDirty(msg.Id)
}

func (m *EgressGWTracker) OnIPSetRemove(msg *proto.IPSetRemove) {
	if _, found := m.ipSetIDToGateways[msg.Id]; !found {
		return
	}
	log.Infof("Remove whole EgressIP set: msg=%v", msg)
	for _, oldGW := range m.ipSetIDToGateways[msg.Id] {
		m.stopPollerIfRunning(msg.Id, oldGW.addr)
	}
	delete(m.ipSetIDToGateways, msg.Id)
	m.markSetDirty(msg.Id)
}

// OnEGWHealthReport is called (on the main dataplane goroutine) when one of our pollers makes a report.
func (m *EgressGWTracker) OnEGWHealthReport(r EGWHealthReport) {
	logCtx := log.WithFields(log.Fields{"egwAddr": r.PollerID.addr, "health": r.Health})
	if p := m.pollers[r.PollerID]; p == nil || p.nonce != r.PollerNonce {
		// This is a message from an old poller.
		logCtx.WithFields(log.Fields{
			"messageNonce":  r.PollerNonce,
			"currentPoller": p,
		}).Info("Received message from a defunct egress gateway poller, ignoring.")
		return
	}
	// If the poller exists, and it has the right nonce then the gateway should exist
	gws := m.ipSetIDToGateways[r.PollerID.setID]
	gw := gws[r.PollerID.addr]
	if gw.healthStatus != r.Health && r.Health == EGWHealthProbeFailed {
		gw.healthFailedAt = time.Now()
	}
	gw.healthStatus = r.Health
	m.markSetDirty(r.PollerID.setID)
	logCtx.Info("Egress gateway health changed.")
}

func (m *EgressGWTracker) markSetDirty(setID string) {
	m.dirtyEgressIPSet.Add(setID)
}

func (m *EgressGWTracker) Dirty() bool {
	return m.dirtyEgressIPSet.Len() > 0
}

func (m *EgressGWTracker) GetAndClearDirtySetIDs() []string {
	s := sortStringSet(m.dirtyEgressIPSet)
	m.dirtyEgressIPSet.Clear()
	return s
}

func (m *EgressGWTracker) GatewaysByID(id string) (gatewaysByIP, bool) {
	gws, exists := m.ipSetIDToGateways[id]
	return gws, exists
}

func (m *EgressGWTracker) AllGatewayIPs() set.Set[ip.Addr] {
	gatewayIPs := set.NewBoxed[ip.Addr]()
	for _, gateways := range m.ipSetIDToGateways {
		for _, g := range gateways {
			gatewayIPs.Add(g.addr)
		}
	}
	return gatewayIPs
}

func (m *EgressGWTracker) ensurePollerRunning(setID string, addr ip.Addr, port uint16) {
	if m.pollInterval == 0 {
		log.Debug("Zero poll interval; disabling poller.")
		return
	}

	id := egwPollerID{
		setID: setID,
		addr:  addr,
	}
	if _, exists := m.pollers[id]; exists {
		return
	}
	ctx, cancel := context.WithCancel(m.context)
	poller := &egwPoller{
		id:                  id,
		nonce:               m.nextPollerNonce,
		cancel:              cancel,
		reportC:             m.healthReportC,
		url:                 fmt.Sprintf("http://%s:%d/readiness", addr, port),
		pollInterval:        m.pollInterval,
		minPollFailureCount: m.minPollFailureCount,
	}
	m.nextPollerNonce += 1
	m.pollers[id] = poller
	poller.Start(ctx)
}

func (m *EgressGWTracker) stopPollerIfRunning(setID string, ip ip.Addr) {
	k := egwPollerID{
		setID: setID,
		addr:  ip,
	}
	if p, exists := m.pollers[k]; !exists {
		return
	} else {
		p.cancel()
	}
	delete(m.pollers, k)
	// Note: we don't wait for the poller to stop here since that could deadlock if it is trying to send a message
	// to this goroutine.  Instead, we filter out messages from defunct pollers based on the per-poller nonce.
}

func (m *EgressGWTracker) AllHealthPortIPSetMembers() []string {
	members := set.New[string]()
	for _, gws := range m.ipSetIDToGateways {
		for _, g := range gws {
			if g.healthPort != 0 {
				members.Add(fmt.Sprintf("%s,tcp:%d", g.addr, g.healthPort))
			}
		}
	}
	return members.Slice()
}

// gateway stores an IPSet member's cidr and maintenance window.
// If the maintenanceStarted.IsZero() or maintenanceFinished.IsZero() then the member is not terminating.
// Otherwise, it is in the process of terminating, and will be deleted at the given maintenanceFinished timestamp.
type gateway struct {
	addr ip.Addr

	// Start and end time of planned maintenance (i.e. time pod was scheduled for deletion and the deletionTimestamp).
	maintenanceStarted  time.Time
	maintenanceFinished time.Time

	healthPort     uint16
	healthStatus   EGWHealth
	healthFailedAt time.Time
}

type EGWHealth string

const (
	EGWHealthUnknown     EGWHealth = ""
	EGWHealthUp          EGWHealth = "up"
	EGWHealthProbeFailed EGWHealth = "probe-failed"
)

type egwPollerID struct {
	setID string
	addr  ip.Addr
}

type egwPoller struct {
	id                  egwPollerID
	nonce               int
	url                 string
	cancel              func()
	reportC             chan<- EGWHealthReport
	fastRetryC          <-chan time.Time
	pollInterval        time.Duration
	minPollFailureCount int
}

func (p *egwPoller) Start(ctx context.Context) {
	go p.loop(ctx)
}

type EGWHealthReport struct {
	PollerID egwPollerID
	// PollerNonce is a unique value for each poller that we create. By including it in the messages we send mack to
	// the main goroutine, it allows us to tell the difference between messages that were already in a queue from a
	// poller that was just shut down and messages from a fresh poller that has just been created.
	PollerNonce int
	Health      EGWHealth
}

func (p *egwPoller) loop(ctx context.Context) {
	logCtx := log.WithFields(log.Fields{
		"probeURL":    p.url,
		"pollerNonce": p.nonce,
	})
	logCtx.Info("Polling health of remote egress gateway.")
	ticker := jitter.NewTicker(p.pollInterval*95/100, p.pollInterval*10/100)
	defer ticker.Stop()
	lastReportedHealth := EGWHealthUnknown
	lastPromHealth := EGWHealthUnknown
	defer func() {
		if lastPromHealth != EGWHealthUnknown {
			numEGWPollsGauge.WithLabelValues(string(lastPromHealth)).Dec()
		}
	}()
	var numBadReports int
	var reportC chan<- EGWHealthReport

	numEGWPollsGauge.WithLabelValues("total").Inc()
	defer numEGWPollsGauge.WithLabelValues("total").Dec()

	for ctx.Err() == nil {
		egwHealth, err := p.doOneProbe(ctx, logCtx)
		if err != nil { // Only used for context done.
			break
		}

		if egwHealth != lastPromHealth {
			if lastPromHealth != EGWHealthUnknown {
				numEGWPollsGauge.WithLabelValues(string(lastPromHealth)).Dec()
			}
			if egwHealth != EGWHealthUnknown {
				numEGWPollsGauge.WithLabelValues(string(egwHealth)).Inc()
			}
			lastPromHealth = egwHealth
		}

		if egwHealth == EGWHealthUp {
			numBadReports = 0
		} else {
			numBadReports++
		}

		if egwHealth != lastReportedHealth {
			if egwHealth == EGWHealthUp || numBadReports >= p.minPollFailureCount {
				// Got a new report to send, unmask the report channel.
				logCtx.Debug("Unmasking report chan.")
				reportC = p.reportC
			} else {
				logCtx.Debug("Delaying report of egress gateway non-ready until we have multiple failures.")
			}
		} else {
			// Not ready to send; mask the channel, so we don't send anything.
			reportC = nil
		}

		report := EGWHealthReport{
			PollerID:    p.id,
			PollerNonce: p.nonce,
			Health:      egwHealth,
		}

		select {
		case reportC <- report:
			lastReportedHealth = egwHealth
			logCtx.WithField("health", egwHealth).Debug("Sent health report.")
			reportC = nil
		case <-ticker.C:
		case <-ctx.Done():
		}
	}
	logCtx.Info("Stopped polling health of remote egress gateway.")
}

func (p *egwPoller) doOneProbe(ctx context.Context, logCtx *log.Entry) (EGWHealth, error) {
	logCtx.Debug("Doing poll...")
	// Set the timeout to be 90% of the poll interval.  Originally, I tried setting this to be the full poll interval
	// but then there's a good chance that the ticker will pop before we return.  That leads to a 50:50 chance of
	// looping again instead of sending the report.
	timeout := p.pollInterval * 90 / 100
	toCtx, cancel := context.WithTimeout(ctx, timeout)
	req, err := http.NewRequestWithContext(toCtx, "GET", p.url, nil)
	defer cancel()

	var overallErr error
	if err != nil {
		overallErr = fmt.Errorf("failed to initiate probe: %w", err)
	} else if resp, err := http.DefaultClient.Do(req); err != nil {
		overallErr = fmt.Errorf("failed to connect: %w", err)
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			overallErr = fmt.Errorf("failed to read: %w", err)
		} else if err = resp.Body.Close(); err != nil {
			overallErr = fmt.Errorf("failed to close response body: %w", err)
		} else {
			if log.GetLevel() >= log.DebugLevel {
				logCtx.Debugf("Response from egress gateway readiness probe:\n%s", body)
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				logCtx.Debug("Remote egress gateway readiness probe succeeded (egress gateway reports ready).")
				return EGWHealthUp, nil
			} else {
				overallErr = fmt.Errorf("remote egress gateway reports non-ready")
			}
		}
	}

	if ctx.Err() != nil {
		// Poller is being shut down.
		logCtx.WithField("result", overallErr).Debug("Poller is being stopped, ignoring result of probe.")
		return "", ctx.Err()
	}

	log.WithError(overallErr).Warn("Egress gateway health probe failed.")
	return EGWHealthProbeFailed, nil
}

// gatewaysByIP maps a member's IP to a gateway
type gatewaysByIP map[ip.Addr]*gateway

func (g gatewaysByIP) allIPs() set.Set[ip.Addr] {
	s := set.NewBoxed[ip.Addr]()
	for _, m := range g {
		s.Add(m.addr)
	}
	return s
}

func (g gatewaysByIP) activeGateways() gatewaysByIP {
	active := make(gatewaysByIP)
	now := time.Now()
	for _, m := range g {
		m := m
		if now.After(m.maintenanceStarted) && now.Before(m.maintenanceFinished) {
			continue
		}
		// This classes "unknown" and "up" together, which makes sure that we don't flap at start-of-day.
		// Would be nice to be more precise to avoid using a bad EGW at start of day but that'd delay programming
		// of EGWs until we've polled them all.
		if m.healthPort != 0 && m.healthStatus == EGWHealthProbeFailed {
			continue
		}
		active[m.addr] = m
	}
	return active
}

func (g gatewaysByIP) terminatingGateways() gatewaysByIP {
	terminating := make(gatewaysByIP)
	now := time.Now()
	for _, m := range g {
		if (now.Equal(m.maintenanceStarted) || now.After(m.maintenanceStarted)) &&
			(now.Equal(m.maintenanceFinished) || now.Before(m.maintenanceFinished)) {
			terminating[m.addr] = m
		}
	}
	return terminating
}

func (g gatewaysByIP) failedGateways() gatewaysByIP {
	failed := make(gatewaysByIP)
	for _, m := range g {
		if m.healthPort != 0 && m.healthStatus == EGWHealthProbeFailed {
			failed[m.addr] = m
		}
	}
	return failed
}

func (g gatewaysByIP) filteredByHopIPs(hopIPs []ip.Addr) gatewaysByIP {
	gws := make(gatewaysByIP)
	hopIPsSet := set.FromArrayBoxed(hopIPs)
	for _, m := range g {
		if hopIPsSet.Contains(m.addr) {
			gws[m.addr] = m
		}
	}
	return gws
}

// Finds the latest maintenance window on the supplied egress gateway pods.
func (g gatewaysByIP) latestTerminatingGateway() *gateway {
	var member *gateway
	for _, m := range g.terminatingGateways() {
		if m.maintenanceFinished.IsZero() {
			continue
		}
		if member == nil || m.maintenanceFinished.After(member.maintenanceFinished) {
			member = m
		}
	}
	return member
}
