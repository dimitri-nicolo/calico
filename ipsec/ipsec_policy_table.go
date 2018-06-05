// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package ipsec

import (
	"reflect"

	"time"

	"net"

	"syscall"

	"fmt"

	"github.com/projectcalico/felix/ip"
	"github.com/projectcalico/libcalico-go/lib/set"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

var (
	gaugeNumIPSecBindings = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_ipsec_bindings_total",
		Help: "Total number of active IPsec bindings.",
	})
	countNumIPSecErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_ipsec_errors",
		Help: "Number of IPsec update failures.",
	})
)

func init() {
	prometheus.MustRegister(gaugeNumIPSecBindings)
	prometheus.MustRegister(countNumIPSecErrors)
}

type PolicySelector struct {
	TrafficSrc ip.V4CIDR
	TrafficDst ip.V4CIDR
	Dir        netlink.Dir
}

func (sel PolicySelector) String() string {
	return fmt.Sprintf("%v -> %v (%v)", sel.TrafficSrc, sel.TrafficDst, sel.Dir)
}

func (sel PolicySelector) Populate(pol *netlink.XfrmPolicy) {
	if sel.TrafficSrc.Prefix() > 0 {
		src := sel.TrafficSrc.ToIPNet()
		pol.Src = &src
	}
	if sel.TrafficDst.Prefix() > 0 {
		dst := sel.TrafficDst.ToIPNet()
		pol.Dst = &dst
	}
	pol.Dir = sel.Dir
}

type PolicyRule struct {
	Action netlink.XfrmPolicyAction

	Mark     uint32
	MarkMask uint32

	TunnelSrc ip.V4Addr
	TunnelDst ip.V4Addr
}

func (r PolicyRule) String() string {
	s := r.Action.String()
	if r.MarkMask != 0 {
		s += fmt.Sprintf(" mask %#x/%#x", r.Mark, r.MarkMask)
	}
	if r.Action != netlink.XFRM_POLICY_BLOCK {
		s += fmt.Sprintf(" tunnel %v -> %v", r.TunnelSrc, r.TunnelDst)
	}
	return s
}

func (r *PolicyRule) Populate(pol *netlink.XfrmPolicy, ourReqID int) {
	if r == nil {
		return
	}
	if r.MarkMask != 0 {
		pol.Mark = &netlink.XfrmMark{
			Value: r.Mark,
			Mask:  r.MarkMask,
		}
	}
	pol.Action = r.Action

	// Note: for a block action, the template doesn't get used.  However, we include it because it allows us
	// to include a ReqID, which we use to match our policies during resync.
	pol.Tmpls = append(pol.Tmpls, netlink.XfrmPolicyTmpl{
		Src:   r.TunnelSrc.AsNetIP(),
		Dst:   r.TunnelDst.AsNetIP(),
		Proto: netlink.XFRM_PROTO_ESP,
		Mode:  netlink.XFRM_MODE_TUNNEL,
		Reqid: ourReqID,
	})
}

func (p *PolicyTable) xfrmPolToOurPol(xfrmPol netlink.XfrmPolicy) (sel PolicySelector, rule *PolicyRule) {
	if len(xfrmPol.Tmpls) == 0 {
		return
	}
	tmpl := xfrmPol.Tmpls[0]
	if tmpl.Reqid != p.ourReqID {
		return
	}

	sel = PolicySelector{
		TrafficSrc: ipNetPtrToCIDR(xfrmPol.Src),
		TrafficDst: ipNetPtrToCIDR(xfrmPol.Dst),
		Dir:        xfrmPol.Dir,
	}
	rule = &PolicyRule{}
	rule.Action = xfrmPol.Action
	if xfrmPol.Mark != nil {
		rule.Mark = xfrmPol.Mark.Value
		rule.MarkMask = xfrmPol.Mark.Mask
	}
	if !tmpl.Src.IsUnspecified() {
		rule.TunnelSrc = ip.FromNetIP(tmpl.Src).(ip.V4Addr)
	}
	if !tmpl.Dst.IsUnspecified() {
		rule.TunnelDst = ip.FromNetIP(tmpl.Dst).(ip.V4Addr)
	}

	return
}

func ipNetPtrToCIDR(ipNet *net.IPNet) (c ip.V4CIDR) {
	if ipNet == nil {
		return
	}
	if ones, _ := ipNet.Mask.Size(); ones == 0 {
		return
	}
	return ip.CIDRFromIPNet(ipNet).(ip.V4CIDR)
}

type PolicyTable struct {
	ourReqID int

	resyncRequired bool

	pendingRuleUpdates map[PolicySelector]*PolicyRule
	pendingDeletions   set.Set

	selectorToRule map[PolicySelector]*PolicyRule

	nlHandleFactory func() (xfrmIface, error)
	nlHndl          xfrmIface

	// Shim for time.Sleep()
	sleep func(time.Duration)
}

func NewPolicyTable(ourReqID int) *PolicyTable {
	return NewPolicyTableWithShims(
		ourReqID,
		newRealNetlinkHandle,
		time.Sleep,
	)
}

func newRealNetlinkHandle() (xfrmIface, error) {
	return netlink.NewHandle(syscall.NETLINK_XFRM)
}

func NewPolicyTableWithShims(ourReqID int, nlHandleFactory func() (xfrmIface, error), sleep func(time.Duration)) *PolicyTable {
	return &PolicyTable{
		ourReqID:           ourReqID,
		resyncRequired:     true,
		pendingRuleUpdates: map[PolicySelector]*PolicyRule{},
		pendingDeletions:   set.New(),
		selectorToRule:     map[PolicySelector]*PolicyRule{},
		nlHandleFactory:    nlHandleFactory,
		sleep:              sleep,
	}
}

var blockRule = PolicyRule{Action: netlink.XFRM_POLICY_BLOCK}

func (p *PolicyTable) SetRule(sel PolicySelector, rule *PolicyRule) {
	debug := log.GetLevel() >= log.DebugLevel
	// Clear out any pending state and then recalculate.
	p.pendingDeletions.Discard(sel)
	delete(p.pendingRuleUpdates, sel)

	if reflect.DeepEqual(p.selectorToRule[sel], rule) {
		// Rule is the same as what we think is in the dataplane already, ignore.
		if debug {
			log.WithFields(log.Fields{
				"sel":  sel,
				"rule": *rule,
			}).Debug("Ignoring no-op update to IPsec rule")
		}
		return
	}

	// Queue up the change.
	if debug {
		log.WithFields(log.Fields{
			"sel":  sel,
			"rule": *rule,
		}).Debug("Queueing update of IPsec rule")
	}
	p.pendingRuleUpdates[sel] = rule
}

func (p *PolicyTable) DeleteRule(sel PolicySelector) {
	// Clear out any pending state and then recalculate.
	p.pendingDeletions.Discard(sel)
	delete(p.pendingRuleUpdates, sel)

	debug := log.GetLevel() >= log.DebugLevel
	if _, ok := p.selectorToRule[sel]; !ok {
		// Rule was never programmed to the dataplane. Ignore.
		if debug {
			log.WithField("sel", sel).Debug("Ignoring no-op delete of IPsec rule")
		}
		return
	}

	// Queue up the change.
	if debug {
		log.WithField("sel", sel).Debug("Queueing delete of IPsec rule")
	}
	p.pendingDeletions.Add(sel)
}

func (p *PolicyTable) Apply() {
	success := false
	retryDelay := 1 * time.Millisecond
	backOff := func() {
		p.sleep(retryDelay)
		retryDelay *= 2
	}
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		if attempt > 0 {
			log.Info("Retrying after an IPsec binding update failure...")
		}
		if p.resyncRequired {
			// Compare our in-memory state against the dataplane and queue up
			// modifications to fix any inconsistencies.
			log.Info("Resyncing IPsec bindings with dataplane.")
			var numProblems int
			numProblems, err = p.tryResync()
			if err != nil {
				log.WithError(err).Warning("Failed to resync with dataplane")
				backOff()
				continue
			}
			if numProblems > 0 {
				log.WithField("numProblems", numProblems).Info(
					"Found inconsistencies in dataplane")
			}
			p.resyncRequired = false
		}

		if err = p.tryUpdates(); err != nil {
			log.WithError(err).Warning("Failed to update IPsec bindings. Marking dataplane for resync.")
			p.resyncRequired = true
			countNumIPSecErrors.Inc()
			backOff()
			continue
		}

		success = true
		break
	}
	if !success {
		p.dumpStateToLog()
		log.WithError(err).Panic("Failed to update IPsec bindings after multiple retries.")
	}
	gaugeNumIPSecBindings.Set(float64(len(p.selectorToRule)))
}

func (p *PolicyTable) tryResync() (numProblems int, err error) {
	log.Info("IPsec resync: starting")
	defer log.Info("IPsec resync: finished")

	xfrmPols, err := p.nl().XfrmPolicyList(netlink.FAMILY_V4)
	if err != nil {
		p.closeNL()
		return 1, err
	}

	expectedState := p.selectorToRule
	actualState := map[PolicySelector]*PolicyRule{}

	// Look up the log level so we can avoid doing expensive log.WithField/Debug calls in the tight loop.
	debug := log.GetLevel() >= log.DebugLevel

	loggedDelete := false
	loggedRepair := false
	for _, xfrmPol := range xfrmPols {
		if debug {
			log.WithField("policy", xfrmPol).Debug("IPsec resync: examining dataplane policy")
		}
		sel, pol := p.xfrmPolToOurPol(xfrmPol)
		if pol == nil {
			if debug {
				log.Debug("IPsec resync: Not one of our policies")
			}
			continue // Not one of our policies
		}
		actualState[sel] = pol
		if expectedPol, ok := expectedState[sel]; !ok {
			// Policy exists in dataplane but not in our expected state.
			if _, ok := p.pendingRuleUpdates[sel]; ok || p.pendingDeletions.Contains(sel) {
				// We've already got an update queued up, which will replace the unexpected policy.
				if debug {
					log.WithField("policy", xfrmPol).Debug(
						"IPsec resync: found unexpected policy but it's already queued for update/deletion")
				}
				continue
			}
			// Queue up a deletion to bring us back into sync.
			if debug || !loggedDelete {
				log.WithField("selector", sel).Warn(
					"IPsec resync: queueing deletion of unexpected policy (skipping further logs of this type).")
				loggedDelete = true
			}
			numProblems++
			p.pendingDeletions.Add(sel)
		} else {
			// Mark this endpoint as seen.
			delete(expectedState, sel)
			// Policy exists in dataplane and our expected state, check whether they match.
			if reflect.DeepEqual(expectedPol, pol) {
				actualState[sel] = expectedPol
				if debug {
					log.WithField("policy", xfrmPol).Debug("IPsec resync: policy matches our state")
				}
				continue // match, nothing to do
			}
			if _, ok := p.pendingRuleUpdates[sel]; ok || p.pendingDeletions.Contains(sel) {
				// We've already got an update queued up that will replace the incorrect policy.
				continue
			}
			// Queue up a repair to bring us back into sync.

			if debug || !loggedRepair {
				log.WithField("policy", xfrmPol).Warn(
					"IPsec resync: found incorrect policy in dataplane, queueing a repair " +
						"(skipping further logs of this type).")
				loggedRepair = true
			}
			numProblems++
			p.pendingRuleUpdates[sel] = expectedPol
		}
	}

	loggedReplace := false
	for sel, pol := range expectedState {
		if _, ok := p.pendingRuleUpdates[sel]; ok {
			// We've already got an update queued up, which will replace the incorrect policy.
			continue
		}
		if p.pendingDeletions.Contains(sel) {
			p.pendingDeletions.Discard(sel)
			continue
		}
		// Expected policy was missing from the dataplane, queue up a repair.
		if debug || !loggedReplace {
			log.WithFields(log.Fields{"sel": sel, "rule": pol}).Warn(
				"IPsec resync: found policy missing from dataplane, queueing a replacement " +
					"(suppressing any further logs)")
			loggedReplace = true
		}
		numProblems++
		p.pendingRuleUpdates[sel] = pol
	}

	p.selectorToRule = actualState

	return
}

func (p *PolicyTable) tryUpdates() (err error) {
	debug := log.GetLevel() >= log.DebugLevel

	if p.pendingDeletions.Len() > 0 {
		log.WithField("numUpdates", p.pendingDeletions.Len()).Info("Applying IPsec policy deletions")
	}
	var lastErr error
	p.pendingDeletions.Iter(func(item interface{}) error {
		sel := item.(PolicySelector)
		xPol := netlink.XfrmPolicy{}
		sel.Populate(&xPol)
		p.selectorToRule[sel].Populate(&xPol, p.ourReqID)
		if debug {
			log.WithFields(log.Fields{"sel": sel, "policy": xPol}).Debug("Deleting rule")
		}
		err := p.nl().XfrmPolicyDel(&xPol)
		if err != nil {
			log.WithError(err).WithField("policy", xPol).Error("Failed to remove IPsec xfrm policy")
			lastErr = err
			p.closeNL()
			return nil
		}
		delete(p.selectorToRule, sel)
		return set.RemoveItem
	})

	if len(p.pendingRuleUpdates) > 0 {
		log.WithField("numUpdates", len(p.pendingRuleUpdates)).Info("Applying IPsec policy updates")
	}
	for sel, rule := range p.pendingRuleUpdates {
		xPol := netlink.XfrmPolicy{}
		sel.Populate(&xPol)
		rule.Populate(&xPol, p.ourReqID)
		if debug {
			log.WithFields(log.Fields{"sel": sel, "rule": rule, "policy": xPol}).Debug(
				"Updating rule")
		}
		err := p.nl().XfrmPolicyUpdate(&xPol)
		if err != nil {
			log.WithError(err).WithField("policy", xPol).Error("Failed to update IPsec xfrm policy")
			lastErr = err
			p.closeNL()
			continue
		}
		p.selectorToRule[sel] = rule
		delete(p.pendingRuleUpdates, sel)
	}

	return lastErr
}

func (p *PolicyTable) dumpStateToLog() {

}

func (p *PolicyTable) closeNL() {
	if p.nlHndl == nil {
		return
	}
	p.nlHndl.Delete()
	p.nlHndl = nil
}

func (p *PolicyTable) nl() xfrmIface {
	if p.nlHndl == nil {
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			p.nlHndl, err = p.nlHandleFactory()
			if err == nil {
				break
			}
			p.sleep(100 * time.Millisecond)
		}
		if p.nlHndl == nil {
			log.WithError(err).Panic("Failed to connect to netlink")
		}
	}
	return p.nlHndl
}

type xfrmIface interface {
	XfrmPolicyList(family int) ([]netlink.XfrmPolicy, error)
	XfrmPolicyUpdate(policy *netlink.XfrmPolicy) error
	XfrmPolicyDel(policy *netlink.XfrmPolicy) error
	Delete()
}
