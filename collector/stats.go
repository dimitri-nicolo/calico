// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gavv/monotime"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

type TrafficDirection int

const (
	TrafficDirInbound TrafficDirection = iota
	TrafficDirOutbound
)

const (
	TrafficDirInboundStr  = "inbound"
	TrafficDirOutboundStr = "outbound"
)

func (t TrafficDirection) String() string {
	if t == TrafficDirInbound {
		return TrafficDirInboundStr
	}
	return TrafficDirOutboundStr
}

// ruleDirToTrafficDir converts the rule direction to the equivalent traffic direction
// (useful for NFLOG based updates where ingress/inbound and egress/outbound are
// tied).
func ruleDirToTrafficDir(r rules.RuleDir) TrafficDirection {
	if r == rules.RuleDirIngress {
		return TrafficDirInbound
	}
	return TrafficDirOutbound
}

// Counter stores packet and byte statistics. It also maintains a delta of
// changes made until a `ResetDeltas()` is called.
type Counter struct {
	packets      int
	bytes        int
	deltaPackets int
	deltaBytes   int
}

func NewCounter(packets int, bytes int) *Counter {
	return &Counter{packets, bytes, packets, bytes}
}

// Values returns packet and bytes stored in a Counter object.
func (c *Counter) Values() (packets, bytes int) {
	return c.packets, c.bytes
}

// DeltaValues returns the packet and byte deltas since the last `ResetDeltas()`.
func (c *Counter) DeltaValues() (packets, bytes int) {
	return c.deltaPackets, c.deltaBytes
}

// Set packet and byte values to `packets` and `bytes`. Returns true if value
// was changed and false otherwise.
func (c *Counter) Set(packets int, bytes int) bool {
	if packets == c.packets && bytes != c.bytes {
		return false
	}

	dp := packets - c.packets
	db := bytes - c.bytes
	if dp < 0 || db < 0 {
		// There has been a reset event.  Best we can do is assume the counters were
		// reset and therefore our delta counts should be incremented by the new
		// values.
		c.deltaPackets += packets
		c.deltaBytes += bytes
	} else {
		// The counters are higher than before so assuming there has been no intermediate
		// reset event, increment our deltas by the deltas of the new and previous counts.
		c.deltaPackets += dp
		c.deltaBytes += db
	}
	c.packets = packets
	c.bytes = bytes
	return true
}

// Increase packet and byte values by `packets` and `bytes`. Always returns
// true.
func (c *Counter) Increase(packets int, bytes int) (changed bool) {
	c.Set(c.packets+packets, c.bytes+bytes)
	changed = true
	return
}

// Reset does a full reset of delta and absolute values tracked by this counter.
func (c *Counter) Reset() {
	c.packets = 0
	c.bytes = 0
	c.deltaPackets = 0
	c.deltaBytes = 0
}

// ResetDeltas sets the delta packet and byte values to zero. Absolute counters are left
// unchanged.
func (c *Counter) ResetDeltas() {
	c.deltaPackets = 0
	c.deltaBytes = 0
}

func (c *Counter) IsZero() bool {
	return (c.packets == 0 && c.bytes == 0)
}

func (c *Counter) String() string {
	return fmt.Sprintf("packets=%v bytes=%v", c.packets, c.bytes)
}

const RuleTraceInitLen = 10

// RuleTrace represents the list of rules (i.e, a Trace) that a packet hits.
// The action of a RuleTrace object is the final action that is not a
// next-Tier/pass action. A RuleTrace also contains a endpoint that the rule
// trace applied to,
type RuleTrace struct {
	path   []*calc.RuleID
	action rules.RuleAction

	// Counter to store the packets and byte counts for the RuleTrace
	ctr   Counter
	dirty bool

	// Stores the Index of the RuleID that has a RuleAction Allow or Deny
	verdictIdx int

	// Optimization Note: When initializing a RuleTrace object, the pathArray
	// array is used as the backing array that has been pre-allocated. This avoids
	// one allocation when creating a RuleTrace object. More info on this here:
	// https://github.com/golang/go/wiki/Performance#memory-profiler
	// (Search for "slice array preallocation").
	pathArray [RuleTraceInitLen]*calc.RuleID
}

func NewRuleTrace() RuleTrace {
	rt := RuleTrace{verdictIdx: -1}
	rt.path = rt.pathArray[:]
	return rt
}

func (t *RuleTrace) String() string {
	rtParts := make([]string, 0)
	for _, tp := range t.path {
		if tp == nil {
			continue
		}
		rtParts = append(rtParts, fmt.Sprintf("(%s)", tp))
	}
	return fmt.Sprintf("path=[%v], action=%v ctr={%v} %s", strings.Join(rtParts, ", "), t.Action(), t.ctr.String())
}

func (t *RuleTrace) Len() int {
	return len(t.path)
}

func (t *RuleTrace) Path() []*calc.RuleID {
	// Minor optimization where we only do a rebuild when we don't have the full
	// path.
	rebuild := false
	idx := 0
	for i, ruleID := range t.path {
		if ruleID == nil {
			rebuild = true
			break
		}
		if ruleID.Action == rules.RuleActionAllow || ruleID.Action == rules.RuleActionDeny {
			idx = i
			break
		}
	}
	if !rebuild {
		return t.path[:idx+1]
	}
	path := make([]*calc.RuleID, 0, RuleTraceInitLen)
	for _, tp := range t.path {
		if tp == nil {
			continue
		}
		path = append(path, tp)
	}
	return path
}

func (t *RuleTrace) ToString() string {
	ruleID := t.VerdictRuleID()
	if ruleID == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%v", ruleID.Tier, ruleID.Name, ruleID.Index, ruleID.Action)
}

func (t *RuleTrace) Action() rules.RuleAction {
	ruleID := t.VerdictRuleID()
	if ruleID == nil {
		// We don't know the verdict RuleID yet.
		return rules.RuleActionNextTier
	}
	return ruleID.Action
}

func (t *RuleTrace) Counters() Counter {
	return t.ctr
}

func (t *RuleTrace) IsDirty() bool {
	return t.dirty
}

// VerdictRuleID returns the RuleID that contains either ActionAllow or
// DenyAction in a RuleTrace or nil if we haven't seen either of these yet.
func (t *RuleTrace) VerdictRuleID() *calc.RuleID {
	if t.verdictIdx >= 0 {
		return t.path[t.verdictIdx]
	} else {
		return nil
	}
}

func (t *RuleTrace) ClearDirtyFlag() {
	t.dirty = false
	t.ctr.ResetDeltas()
}

func (t *RuleTrace) addRuleID(rid *calc.RuleID, tierIdx, numPkts, numBytes int) bool {
	if tierIdx >= t.Len() {
		// Insertion Index is beyond than current length. Grow the path slice as long
		// as necessary.
		incSize := (tierIdx / RuleTraceInitLen) * RuleTraceInitLen
		newPath := make([]*calc.RuleID, t.Len()+incSize)
		copy(newPath, t.path)
		t.path = newPath
		t.path[tierIdx] = rid
	} else {
		existingRuleID := t.path[tierIdx]
		switch {
		case existingRuleID == nil:
			// Position is empty, insert and be done.
			t.path[tierIdx] = rid
		case !existingRuleID.Equals(rid):
			return false
		}
	}
	if rid.Action != rules.RuleActionNextTier {
		t.ctr.Increase(numPkts, numBytes)
		t.verdictIdx = tierIdx
	}
	t.dirty = true
	return true
}

func (t *RuleTrace) replaceRuleID(rid *calc.RuleID, tierIdx, numPkts, numBytes int) {
	if rid.Action == rules.RuleActionNextTier {
		t.path[tierIdx] = rid
		return
	}
	// New tracepoint is not a next-Tier action truncate at this Index.
	t.path[tierIdx] = rid
	t.ctr = *NewCounter(numPkts, numBytes)
	t.dirty = true
	t.verdictIdx = tierIdx
}

// ToMetricUpdate converts the RuleTrace to a MetricUpdate used by the reporter.
func (rt *RuleTrace) ToMetricUpdate(ut UpdateType, t Tuple, td TrafficDirection, ctr *Counter, ctrRev *Counter) MetricUpdate {
	var (
		dp, db, dpRev, dbRev int
		isConn               bool
	)
	if ctr != nil && ctrRev != nil {
		dp, db = ctr.DeltaValues()
		dpRev, dbRev = ctrRev.DeltaValues()
		isConn = true
	} else {
		dp, db = rt.ctr.DeltaValues()
		isConn = false
	}
	mu := MetricUpdate{
		updateType:   ut,
		tuple:        t,
		ruleID:       rt.VerdictRuleID(),
		isConnection: isConn,
	}
	switch td {
	case TrafficDirInbound:
		mu.inMetric = MetricValue{dp, db}
		if isConn {
			mu.outMetric = MetricValue{dpRev, dbRev}
		}
	case TrafficDirOutbound:
		mu.outMetric = MetricValue{dp, db}
		if isConn {
			mu.inMetric = MetricValue{dpRev, dbRev}
		}
	}
	return mu
}

// Tuple represents a 5-Tuple value that identifies a connection/flow of packets
// with an implicit notion of Direction that comes with the use of a source and
// destination. This is a hashable object and can be used as a map's key.
type Tuple struct {
	src   [16]byte
	dst   [16]byte
	proto int
	l4Src int
	l4Dst int
}

func NewTuple(src [16]byte, dst [16]byte, proto int, l4Src int, l4Dst int) *Tuple {
	t := &Tuple{
		src:   src,
		dst:   dst,
		proto: proto,
		l4Src: l4Src,
		l4Dst: l4Dst,
	}
	return t
}

func (t *Tuple) String() string {
	return fmt.Sprintf("src=%v dst=%v proto=%v sport=%v dport=%v", net.IP(t.src[:16]).String(), net.IP(t.dst[:16]).String(), t.proto, t.l4Src, t.l4Dst)
}

// Data contains metadata and statistics such as rule counters and age of a
// connection(Tuple). Each Data object contains:
// - 2 RuleTrace's - Ingress and Egress - each providing information on the
// where the Policy was applied, with additional information on corresponding
// workload endpoint. The EgressRuleTrace and the IngressRuleTrace record the
// policies that this tuple can hit - egress from the workload that is the
// source of this connection and ingress into a workload that terminated this.
// - 2 counters - connTrackCtr for the originating Direction of the connection
// and connTrackCtrReverse for the reverse/reply of this connection.
type Data struct {
	Tuple Tuple
	key   model.Key

	// Indicates if this is a connection
	isConnection bool

	// These are the counts from conntrack
	connTrackCtr        Counter
	connTrackCtrReverse Counter

	// These contain the aggregated counts per tuple per rule.
	IngressRuleTrace RuleTrace
	EgressRuleTrace  RuleTrace

	createdAt  time.Duration
	updatedAt  time.Duration
	ageTimeout time.Duration
	dirty      bool
}

func NewData(tuple Tuple, key model.Key, duration time.Duration) *Data {
	now := monotime.Now()
	return &Data{
		Tuple:            tuple,
		key:              key,
		IngressRuleTrace: NewRuleTrace(),
		EgressRuleTrace:  NewRuleTrace(),
		createdAt:        now,
		updatedAt:        now,
		ageTimeout:       duration,
		dirty:            true,
	}
}

func (d *Data) String() string {
	return fmt.Sprintf("tuple={%v}, ep={%v} connTrackCtr={%v}, connTrackCtrReverse={%v}, updatedAt=%v ingressRuleTrace={%v} egressRuleTrace={%v}",
		&(d.Tuple), endpointName(d.key), d.connTrackCtr.String(), d.connTrackCtrReverse.String(), d.updatedAt, d.IngressRuleTrace, d.EgressRuleTrace)
}

func (d *Data) touch() {
	d.updatedAt = monotime.Now()
}

func (d *Data) setDirtyFlag() {
	d.dirty = true
}

func (d *Data) clearDirtyFlag() {
	d.dirty = false
	d.connTrackCtr.ResetDeltas()
	d.connTrackCtrReverse.ResetDeltas()
	d.IngressRuleTrace.ClearDirtyFlag()
	d.EgressRuleTrace.ClearDirtyFlag()
}

func (d *Data) IsDirty() bool {
	return d.dirty
}

func (d *Data) CreatedAt() time.Duration {
	return d.createdAt
}

func (d *Data) UpdatedAt() time.Duration {
	return d.updatedAt
}

func (d *Data) DurationSinceLastUpdate() time.Duration {
	return monotime.Since(d.updatedAt)
}

func (d *Data) DurationSinceCreate() time.Duration {
	return monotime.Since(d.createdAt)
}

// Returns the final action of the RuleTrace
func (d *Data) IngressAction() rules.RuleAction {
	return d.IngressRuleTrace.Action()
}

// Returns the final action of the RuleTrace
func (d *Data) EgressAction() rules.RuleAction {
	return d.EgressRuleTrace.Action()
}

func (d *Data) Counters() Counter {
	return d.connTrackCtr
}

func (d *Data) CountersReverse() Counter {
	return d.connTrackCtrReverse
}

// Add packets and bytes to the Counters' values. Use the IncreaseCounters*
// methods when the source of packets/bytes are delta values.
func (d *Data) IncreaseCounters(packets int, bytes int) {
	d.connTrackCtr.Increase(packets, bytes)
	d.setDirtyFlag()
	d.touch()
}

// Add packets and bytes to the Reverse Counters' values. Use the IncreaseCounters*
// methods when the source of packets/bytes are delta values.
func (d *Data) IncreaseCountersReverse(packets int, bytes int) {
	d.connTrackCtrReverse.Increase(packets, bytes)
	d.setDirtyFlag()
	d.touch()
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCounters(packets int, bytes int) {
	changed := d.connTrackCtr.Set(packets, bytes)
	if changed {
		d.setDirtyFlag()
	}
	d.isConnection = true
	d.touch()
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCountersReverse(packets int, bytes int) {
	changed := d.connTrackCtrReverse.Set(packets, bytes)
	if changed {
		d.setDirtyFlag()
	}
	d.isConnection = true
	d.touch()
}

// ResetConntrackCounters resets the counters associated with the tracked connection for
// the data.
func (d *Data) ResetConntrackCounters() {
	d.isConnection = false
	d.connTrackCtr.Reset()
	d.connTrackCtrReverse.Reset()
}

func (d *Data) AddRuleID(ruleID *calc.RuleID, tierIdx, numPkts, numBytes int) bool {
	var ok bool
	switch ruleID.Direction {
	case rules.RuleDirIngress:
		ok = d.IngressRuleTrace.addRuleID(ruleID, tierIdx, numPkts, numBytes)
	case rules.RuleDirEgress:
		ok = d.EgressRuleTrace.addRuleID(ruleID, tierIdx, numPkts, numBytes)
	}

	if ok {
		d.touch()
		d.setDirtyFlag()
	}
	return ok
}

func (d *Data) ReplaceRuleID(ruleID *calc.RuleID, tierIdx, numPkts, numBytes int) bool {
	switch ruleID.Direction {
	case rules.RuleDirIngress:
		d.IngressRuleTrace.replaceRuleID(ruleID, tierIdx, numPkts, numBytes)
	case rules.RuleDirEgress:
		d.EgressRuleTrace.replaceRuleID(ruleID, tierIdx, numPkts, numBytes)
	}

	d.touch()
	d.setDirtyFlag()
	return true
}

func (d *Data) Report(c chan<- MetricUpdate, expired bool) {
	ut := UpdateTypeReport
	if expired {
		ut = UpdateTypeExpire
	}
	if ((d.EgressRuleTrace.Action() == rules.RuleActionDeny || d.EgressRuleTrace.Action() == rules.RuleActionAllow) && (expired || d.EgressRuleTrace.IsDirty())) ||
		(!expired && d.EgressRuleTrace.Action() == rules.RuleActionAllow && d.isConnection && d.IsDirty()) {
		if d.isConnection {
			c <- d.EgressRuleTrace.ToMetricUpdate(ut, d.Tuple, TrafficDirOutbound, &d.connTrackCtr, &d.connTrackCtrReverse)
		} else {
			c <- d.EgressRuleTrace.ToMetricUpdate(ut, d.Tuple, TrafficDirOutbound, nil, nil)
		}
	}
	if ((d.IngressRuleTrace.Action() == rules.RuleActionDeny || d.IngressRuleTrace.Action() == rules.RuleActionAllow) && (expired || d.IngressRuleTrace.IsDirty())) ||
		(!expired && d.IngressRuleTrace.Action() == rules.RuleActionAllow && d.isConnection && d.IsDirty()) {
		if d.isConnection {
			c <- d.IngressRuleTrace.ToMetricUpdate(ut, d.Tuple, TrafficDirInbound, &d.connTrackCtr, &d.connTrackCtrReverse)
		} else {
			c <- d.IngressRuleTrace.ToMetricUpdate(ut, d.Tuple, TrafficDirInbound, nil, nil)
		}
	}

	// Metrics have been reported, so acknowledge the stored data by resetting the dirty
	// flag and resetting the delta counts.
	d.clearDirtyFlag()
}

// endpointName is a convenience function to return a printable name for an endpoint.
func endpointName(key model.Key) (name string) {
	switch k := key.(type) {
	case model.WorkloadEndpointKey:
		name = workloadEndpointName(k)
	case model.HostEndpointKey:
		name = hostEndpointName(k)
	}
	return
}

func workloadEndpointName(wep model.WorkloadEndpointKey) string {
	return "WEP(" + wep.Hostname + "/" + wep.OrchestratorID + "/" + wep.WorkloadID + "/" + wep.EndpointID + ")"
}

func hostEndpointName(hep model.HostEndpointKey) string {
	return "HEP(" + hep.Hostname + "/" + hep.EndpointID + ")"
}
