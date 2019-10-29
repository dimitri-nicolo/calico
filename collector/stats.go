// Copyright (c) 2016-2019 Tigera, Inc. All rights reserved.

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

// RuleMatch type is used to indicate whether a rule match from an nflog is newly set, unchanged from the previous
// value, or has been updated. In the latter case the existing entry should be reported and expired.
type RuleMatch byte

const (
	RuleMatchUnchanged RuleMatch = iota
	RuleMatchSet
	RuleMatchIsDifferent
)

// ruleDirToTrafficDir converts the rule direction to the equivalent traffic direction
// (useful for NFLOG based updates where ingress/inbound and egress/outbound are
// tied).
func ruleDirToTrafficDir(r rules.RuleDir) TrafficDirection {
	if r == rules.RuleDirIngress {
		return TrafficDirInbound
	}
	return TrafficDirOutbound
}

const RuleTraceInitLen = 10

// RuleTrace represents the list of rules (i.e, a Trace) that a packet hits.
// The action of a RuleTrace object is the final action that is not a
// next-Tier/pass action. A RuleTrace also contains a endpoint that the rule
// trace applied to,
type RuleTrace struct {
	path []*calc.RuleID

	// The reported path. This is calculated and stored when metrics are reported.
	rulesToReport []*calc.RuleID

	// Counters to store the packets and byte counts for the RuleTrace
	pktsCtr  Counter
	bytesCtr Counter
	dirty    bool

	// Stores the Index of the RuleID that has a RuleAction Allow or Deny.
	verdictIdx int

	// Optimization Note: When initializing a RuleTrace object, the pathArray
	// array is used as the backing array that has been pre-allocated. This avoids
	// one allocation when creating a RuleTrace object. More info on this here:
	// https://github.com/golang/go/wiki/Performance#memory-profiler
	// (Search for "slice array preallocation").
	pathArray [RuleTraceInitLen]*calc.RuleID
}

func (rt *RuleTrace) Init() {
	rt.verdictIdx = -1
	rt.path = rt.pathArray[:]
}

func (t *RuleTrace) String() string {
	rtParts := make([]string, 0)
	for _, tp := range t.Path() {
		rtParts = append(rtParts, fmt.Sprintf("(%s)", tp))
	}
	return fmt.Sprintf(
		"path=[%v], action=%v ctr={packets=%v bytes=%v}",
		strings.Join(rtParts, ", "), t.Action(), t.pktsCtr.Absolute(), t.bytesCtr.Absolute(),
	)
}

func (t *RuleTrace) Len() int {
	return len(t.path)
}

func (t *RuleTrace) Path() []*calc.RuleID {
	if t.rulesToReport != nil {
		return t.rulesToReport
	}
	if t.verdictIdx < 0 {
		return nil
	}

	// The reported rules have not been calculated or have changed. Calculate them now.
	t.rulesToReport = make([]*calc.RuleID, 0, t.verdictIdx)

	// Iterate through the ruleIDs gathered in the nflog path. The IDs will be ordered by staged policies and tiers.
	// e.g.   tier1.SNP1 tier1.SNP2 tier1(EOT) tier2.SNP2 tier2(EOT)
	//     or tier1.SNP1 tier1.NP1  [n/a]      tier2.NP1  [n/a]
	// Both of these represent possible outcomes for the same two tiers. There will be staged matches up to the first
	// enforced policy match, or the end-of-tier action (in which case there will be a hit for each staged policy).
	//
	// We don't add end of tier passes since they are only used for internal bookkeeping.
	for i := 0; i <= t.verdictIdx; i++ {
		r := t.path[i]
		if r == nil || r.IsEndOfTierPass() {
			continue
		}

		endOfTierIndex := func() int {
			for j := i + 1; j <= t.verdictIdx; j++ {
				if t.path[j] != nil && t.path[j].Tier != r.Tier {
					return j - 1
				}
			}
			return t.verdictIdx
		}

		if model.PolicyIsStaged(r.Name) {
			// This is a staged policy. If the rule is an implicit drop then we only include it if the end-of-tier
			// pass action has also been hit.
			if r.IsImplicitDropRule() {
				finalIdx := endOfTierIndex()
				if t.path[finalIdx] == nil || !t.path[finalIdx].IsEndOfTierPass() {
					// This is an implicit drop, but there is no end of tier pass - we do not need to add this entry.
					continue
				}
			}

			// Add the report and then continue to the next entry in the path.
			t.rulesToReport = append(t.rulesToReport, r)
			continue
		}

		// This is an enforced policy, so just add the rule. There should be no more rules this tier, so jump to the end
		// of the tier (we might already be at that index, e.g. if we are processing the verdict).
		t.rulesToReport = append(t.rulesToReport, r)
		i = endOfTierIndex()
	}
	return t.rulesToReport
}

func (t *RuleTrace) ToString() string {
	ruleID := t.VerdictRuleID()
	if ruleID == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%d/%v", ruleID.Tier, ruleID.Name, ruleID.Index, ruleID.Action)
}

func (t *RuleTrace) Action() rules.RuleAction {
	ruleID := t.VerdictRuleID()
	if ruleID == nil {
		// We don't know the verdict RuleID yet.
		return rules.RuleActionPass
	}
	return ruleID.Action
}

func (t *RuleTrace) IsDirty() bool {
	return t.dirty
}

// FoundVerdict returns true if the verdict rule has been found, that is the rule that contains
// the final allow or deny action.
func (t *RuleTrace) FoundVerdict() bool {
	return t.verdictIdx >= 0
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
	t.pktsCtr.ResetDelta()
	t.bytesCtr.ResetDelta()
}

func (t *RuleTrace) addRuleID(rid *calc.RuleID, matchIdx, numPkts, numBytes int) RuleMatch {
	t.maybeResizePath(matchIdx)

	ru := RuleMatchUnchanged

	if existingRuleID := t.path[matchIdx]; existingRuleID == nil {
		// Position is empty, insert and be done. Start revision at 1 to match the expected initial revisions of any
		// staged policy matches (which will be zero value + 1 i.e. 1)
		t.path[matchIdx] = rid
		ru = RuleMatchSet
	} else if !existingRuleID.Equals(rid) {
		// Position is not empty, and does not match the new value.
		return RuleMatchIsDifferent
	}

	// Set as dirty and increment the match revision number for this tier.
	t.dirty = true

	if !model.PolicyIsStaged(rid.Name) && rid.Action != rules.RuleActionPass {
		// This is a verdict action, so increment counters and set our verdict index.
		t.pktsCtr.Increase(numPkts)
		t.bytesCtr.Increase(numBytes)
		t.verdictIdx = matchIdx
	}

	return ru
}

func (t *RuleTrace) replaceRuleID(rid *calc.RuleID, matchIdx, numPkts, numBytes int) {
	// Set the match rule and increment the match revision number for this tier.
	t.path[matchIdx] = rid
	t.dirty = true

	// Reset the reporting path so that we recalculate it next report.
	t.rulesToReport = nil

	if !model.PolicyIsStaged(rid.Name) && rid.Action != rules.RuleActionPass {
		// This is a verdict action, so reset and set counters and set our verdict index.
		t.pktsCtr.ResetAndSet(numPkts)
		t.bytesCtr.ResetAndSet(numBytes)
		t.verdictIdx = matchIdx
	}
}

// maybeResizePath may resize the tier array based on the index of the tier.
func (t *RuleTrace) maybeResizePath(matchIdx int) {
	if matchIdx >= t.Len() {
		// Insertion Index is beyond than current length. Grow the path slice as long
		// as necessary.
		incSize := (matchIdx / RuleTraceInitLen) * RuleTraceInitLen
		newPath := make([]*calc.RuleID, t.Len()+incSize)
		copy(newPath, t.path)
		t.path = newPath
	}
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

func (t *Tuple) GetSourcePort() int {
	return t.l4Src
}

func (t *Tuple) SetSourcePort(port int) {
	t.l4Src = port
}

func (t *Tuple) GetDestPort() int {
	return t.l4Dst
}

func (t *Tuple) SourceNet() net.IP {
	return net.IP(t.src[:16])
}

func (t *Tuple) DestNet() net.IP {
	return net.IP(t.dst[:16])
}

// Data contains metadata and statistics such as rule counters and age of a
// connection(Tuple). Each Data object contains:
// - 2 RuleTrace's - Ingress and Egress - each providing information on the
// where the Policy was applied, with additional information on corresponding
// workload endpoint. The EgressRuleTrace and the IngressRuleTrace record the
// policies that this tuple can hit - egress from the workload that is the
// source of this connection and ingress into a workload that terminated this.
// - Connection based counters (e.g, for conntrack packets/bytes and HTTP requests).
type Data struct {
	Tuple Tuple

	origSourceIPs       *boundedSet
	origSourceIPsActive bool

	// Contains endpoint information corresponding to source and
	// destination endpoints. Either of these values can be nil
	// if we don't have information about the endpoint.
	srcEp *calc.EndpointData
	dstEp *calc.EndpointData

	// Indicates if this is a connection
	isConnection bool

	// Connection related counters.
	conntrackPktsCtr         Counter
	conntrackPktsCtrReverse  Counter
	conntrackBytesCtr        Counter
	conntrackBytesCtrReverse Counter
	httpReqAllowedCtr        Counter
	httpReqDeniedCtr         Counter

	// These contain the aggregated counts per tuple per rule.
	IngressRuleTrace RuleTrace
	EgressRuleTrace  RuleTrace

	updatedAt     time.Duration
	ruleUpdatedAt time.Duration

	dirty bool
}

func NewData(tuple Tuple, srcEp, dstEp *calc.EndpointData, maxOriginalIPsSize int) *Data {
	now := monotime.Now()
	d := &Data{
		Tuple:         tuple,
		origSourceIPs: NewBoundedSet(maxOriginalIPsSize),
		updatedAt:     now,
		ruleUpdatedAt: now,
		dirty:         true,
		srcEp:         srcEp,
		dstEp:         dstEp,
	}
	d.IngressRuleTrace.Init()
	d.EgressRuleTrace.Init()
	return d
}

func (d *Data) String() string {
	var (
		srcName, dstName string
		osi              []net.IP
		osiTc            int
	)
	if d.srcEp != nil {
		srcName = endpointName(d.srcEp.Key)
	} else {
		srcName = "<unknown>"
	}
	if d.dstEp != nil {
		dstName = endpointName(d.dstEp.Key)
	} else {
		dstName = "<unknown>"
	}
	if d.origSourceIPs != nil {
		osi = d.origSourceIPs.ToIPSlice()
		osiTc = d.origSourceIPs.TotalCount()
	}
	return fmt.Sprintf(
		"tuple={%v}, srcEp={%v} dstEp={%v} connTrackCtr={packets=%v bytes=%v}, "+
			"connTrackCtrReverse={packets=%v bytes=%v}, httpPkts={allowed=%v, denied=%v}, updatedAt=%v ingressRuleTrace={%v} egressRuleTrace={%v}, "+
			"origSourceIPs={ips=%v totalCount=%v}",
		&(d.Tuple), srcName, dstName, d.conntrackPktsCtr.Absolute(), d.conntrackBytesCtr.Absolute(),
		d.conntrackPktsCtrReverse.Absolute(), d.conntrackBytesCtrReverse.Absolute(), d.httpReqAllowedCtr.Delta(),
		d.httpReqDeniedCtr.Delta(), d.updatedAt, d.IngressRuleTrace, d.EgressRuleTrace, osi, osiTc)
}

func (d *Data) touch() {
	d.updatedAt = monotime.Now()
}

func (d *Data) setDirtyFlag() {
	d.dirty = true
}

func (d *Data) clearConnDirtyFlag() {
	d.dirty = false
	d.httpReqAllowedCtr.ResetDelta()
	d.httpReqDeniedCtr.ResetDelta()
	d.conntrackPktsCtr.ResetDelta()
	d.conntrackBytesCtr.ResetDelta()
	d.conntrackPktsCtrReverse.ResetDelta()
	d.conntrackBytesCtrReverse.ResetDelta()
}

func (d *Data) IsDirty() bool {
	return d.dirty
}

func (d *Data) UpdatedAt() time.Duration {
	return d.updatedAt
}

func (d *Data) RuleUpdatedAt() time.Duration {
	return d.ruleUpdatedAt
}

func (d *Data) DurationSinceLastUpdate() time.Duration {
	return monotime.Since(d.updatedAt)
}

func (d *Data) DurationSinceLastRuleUpdate() time.Duration {
	return monotime.Since(d.ruleUpdatedAt)
}

// Returns the final action of the RuleTrace
func (d *Data) IngressAction() rules.RuleAction {
	return d.IngressRuleTrace.Action()
}

// Returns the final action of the RuleTrace
func (d *Data) EgressAction() rules.RuleAction {
	return d.EgressRuleTrace.Action()
}

func (d *Data) ConntrackPacketsCounter() Counter {
	return d.conntrackPktsCtr
}

func (d *Data) ConntrackBytesCounter() Counter {
	return d.conntrackBytesCtr
}

func (d *Data) ConntrackPacketsCounterReverse() Counter {
	return d.conntrackPktsCtrReverse
}

func (d *Data) ConntrackBytesCounterReverse() Counter {
	return d.conntrackBytesCtrReverse
}

func (d *Data) HTTPRequestsAllowed() Counter {
	return d.httpReqAllowedCtr
}

func (d *Data) HTTPRequestsDenied() Counter {
	return d.httpReqDeniedCtr
}

// Set In Counters' values to packets and bytes. Use the SetConntrackCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetConntrackCounters(packets int, bytes int) {
	if d.conntrackPktsCtr.Set(packets) && d.conntrackBytesCtr.Set(bytes) {
		d.setDirtyFlag()
	}
	d.isConnection = true
	d.touch()
}

// Set In Counters' values to packets and bytes. Use the SetConntrackCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetConntrackCountersReverse(packets int, bytes int) {
	if d.conntrackPktsCtrReverse.Set(packets) && d.conntrackBytesCtrReverse.Set(bytes) {
		d.setDirtyFlag()
	}
	d.isConnection = true
	d.touch()
}

// Increment the HTTP Request allowed count.
func (d *Data) IncreaseHTTPRequestAllowedCounter(delta int) {
	if delta == 0 {
		return
	}
	d.httpReqAllowedCtr.Increase(delta)
	d.setDirtyFlag()
	d.touch()
}

// Increment the HTTP Request denied count.
func (d *Data) IncreaseHTTPRequestDeniedCounter(delta int) {
	if delta == 0 {
		return
	}
	d.httpReqDeniedCtr.Increase(delta)
	d.setDirtyFlag()
	d.touch()
}

// ResetConntrackCounters resets the counters associated with the tracked connection for
// the data.
func (d *Data) ResetConntrackCounters() {
	d.isConnection = false
	d.conntrackPktsCtr.Reset()
	d.conntrackBytesCtr.Reset()
	d.conntrackPktsCtrReverse.Reset()
	d.conntrackBytesCtrReverse.Reset()
}

// ResetApplicationCounters resets the counters associated with application layer statistics.
func (d *Data) ResetApplicationCounters() {
	d.httpReqAllowedCtr.Reset()
	d.httpReqDeniedCtr.Reset()
}

func (d *Data) SetSourceEndpointData(sep *calc.EndpointData) {
	d.srcEp = sep
}

func (d *Data) SetDestinationEndpointData(dep *calc.EndpointData) {
	d.dstEp = dep
}

func (d *Data) AddRuleID(ruleID *calc.RuleID, matchIdx, numPkts, numBytes int) RuleMatch {
	var ru RuleMatch
	switch ruleID.Direction {
	case rules.RuleDirIngress:
		ru = d.IngressRuleTrace.addRuleID(ruleID, matchIdx, numPkts, numBytes)
	case rules.RuleDirEgress:
		ru = d.EgressRuleTrace.addRuleID(ruleID, matchIdx, numPkts, numBytes)
	}

	if ru != RuleMatchUnchanged {
		d.touch()
		d.setDirtyFlag()
	}
	return ru
}

func (d *Data) ReplaceRuleID(ruleID *calc.RuleID, matchIdx, numPkts, numBytes int) {
	switch ruleID.Direction {
	case rules.RuleDirIngress:
		d.IngressRuleTrace.replaceRuleID(ruleID, matchIdx, numPkts, numBytes)
	case rules.RuleDirEgress:
		d.EgressRuleTrace.replaceRuleID(ruleID, matchIdx, numPkts, numBytes)
	}

	d.touch()
	d.setDirtyFlag()
}

func (d *Data) AddOriginalSourceIPs(bs *boundedSet) {
	d.origSourceIPs.Combine(bs)
	d.origSourceIPsActive = true
	d.isConnection = true
	d.touch()
	d.setDirtyFlag()
}

func (d *Data) OriginalSourceIps() []net.IP {
	return d.origSourceIPs.ToIPSlice()
}

func (d *Data) IncreaseNumUniqueOriginalSourceIPs(deltaNum int) {
	d.origSourceIPs.IncreaseTotalCount(deltaNum)
	d.isConnection = true
	d.touch()
	d.setDirtyFlag()
}

func (d *Data) NumUniqueOriginalSourceIPs() int {
	return d.origSourceIPs.TotalCount()
}

func (d *Data) Report(c chan<- MetricUpdate, expired bool) {
	ut := UpdateTypeReport
	if expired {
		ut = UpdateTypeExpire
	}
	// For connections and non-connections, we only send ingress and egress updates if:
	// -  There is something to report, i.e.
	//    -  flow is expired, or
	//    -  associated stats are dirty
	// -  The policy verdict rule has been determined. Note that for connections the policy verdict may be "Deny" due
	//    to DropActionOverride setting (e.g. if set to ALLOW, then we'll get connection stats, but the metrics will
	//    indicate Denied).
	// Only clear the associated stats and dirty flag once the metrics are reported.
	if d.isConnection {
		// Report connection stats.
		if expired || d.IsDirty() {
			// Track if we need to send a separate expire metric update. This is required when we are only
			// reporting Original IP metric updates and want to send a corresponding expiration metric update.
			// When they are correlated with regular metric updates and connection metrics, we don't need to
			// send this.
			sendOrigSourceIPsExpire := true
			if d.EgressRuleTrace.FoundVerdict() {
				c <- d.metricUpdateEgressConn(ut)
			}
			if d.IngressRuleTrace.FoundVerdict() {
				sendOrigSourceIPsExpire = false
				c <- d.metricUpdateIngressConn(ut)
			}

			// We may receive HTTP Request data after we've flushed the connection counters.
			if (expired && d.origSourceIPsActive && sendOrigSourceIPsExpire) || d.NumUniqueOriginalSourceIPs() != 0 {
				d.origSourceIPsActive = !expired
				c <- d.metricUpdateOrigSourceIPs(ut)
			}

			// Clear the connection dirty flag once the stats have been reported. Note that we also clear the
			// rule trace stats here too since any data stored in them has been superceded by the connection
			// stats.
			d.clearConnDirtyFlag()
			d.EgressRuleTrace.ClearDirtyFlag()
			d.IngressRuleTrace.ClearDirtyFlag()
		}
	} else {
		// Report rule trace stats.
		if (expired || d.EgressRuleTrace.IsDirty()) && d.EgressRuleTrace.FoundVerdict() {
			c <- d.metricUpdateEgressNoConn(ut)
			d.EgressRuleTrace.ClearDirtyFlag()
		}
		if (expired || d.IngressRuleTrace.IsDirty()) && d.IngressRuleTrace.FoundVerdict() {
			c <- d.metricUpdateIngressNoConn(ut)
			d.IngressRuleTrace.ClearDirtyFlag()
		}

		// We do not need to clear the connection stats here. Connection stats are fully reset if the Data moves
		// from a connection to non-connection state.
	}
}

// metricUpdateIngressConn creates a metric update for Inbound connection traffic
func (d *Data) metricUpdateIngressConn(ut UpdateType) MetricUpdate {
	return MetricUpdate{
		updateType:   ut,
		tuple:        d.Tuple,
		srcEp:        d.srcEp,
		dstEp:        d.dstEp,
		ruleIDs:      d.IngressRuleTrace.Path(),
		isConnection: d.isConnection,
		inMetric: MetricValue{
			deltaPackets:             d.conntrackPktsCtr.Delta(),
			deltaBytes:               d.conntrackBytesCtr.Delta(),
			deltaAllowedHTTPRequests: d.httpReqAllowedCtr.Delta(),
			deltaDeniedHTTPRequests:  d.httpReqDeniedCtr.Delta(),
		},
		outMetric: MetricValue{
			deltaPackets: d.conntrackPktsCtrReverse.Delta(),
			deltaBytes:   d.conntrackBytesCtrReverse.Delta(),
		},
	}
}

// metricUpdateEgressConn creates a metric update for Outbound connection traffic
func (d *Data) metricUpdateEgressConn(ut UpdateType) MetricUpdate {
	return MetricUpdate{
		updateType:   ut,
		tuple:        d.Tuple,
		srcEp:        d.srcEp,
		dstEp:        d.dstEp,
		ruleIDs:      d.EgressRuleTrace.Path(),
		isConnection: d.isConnection,
		inMetric: MetricValue{
			deltaPackets: d.conntrackPktsCtrReverse.Delta(),
			deltaBytes:   d.conntrackBytesCtrReverse.Delta(),
		},
		outMetric: MetricValue{
			deltaPackets: d.conntrackPktsCtr.Delta(),
			deltaBytes:   d.conntrackBytesCtr.Delta(),
		},
	}
}

// metricUpdateIngressNoConn creates a metric update for Inbound non-connection traffic
func (d *Data) metricUpdateIngressNoConn(ut UpdateType) MetricUpdate {
	return MetricUpdate{
		updateType:   ut,
		tuple:        d.Tuple,
		srcEp:        d.srcEp,
		dstEp:        d.dstEp,
		ruleIDs:      d.IngressRuleTrace.Path(),
		isConnection: d.isConnection,
		inMetric: MetricValue{
			deltaPackets: d.IngressRuleTrace.pktsCtr.Delta(),
			deltaBytes:   d.IngressRuleTrace.bytesCtr.Delta(),
		},
	}
}

// metricUpdateEgressNoConn creates a metric update for Outbound non-connection traffic
func (d *Data) metricUpdateEgressNoConn(ut UpdateType) MetricUpdate {
	return MetricUpdate{
		updateType:   ut,
		tuple:        d.Tuple,
		srcEp:        d.srcEp,
		dstEp:        d.dstEp,
		ruleIDs:      d.EgressRuleTrace.Path(),
		isConnection: d.isConnection,
		outMetric: MetricValue{
			deltaPackets: d.EgressRuleTrace.pktsCtr.Delta(),
			deltaBytes:   d.EgressRuleTrace.bytesCtr.Delta(),
		},
	}
}

// metricUpdateOrigSourceIPs creates a metric update for HTTP Data (original source ips).
func (d *Data) metricUpdateOrigSourceIPs(ut UpdateType) MetricUpdate {
	// We send Original Source IP updates as standalone metric updates.
	// If however we can't find out the rule trace then we also include
	// an unknown rule ID that the rest of the  metric pipeline uses to
	// extract action and direction.
	var unknownRuleID *calc.RuleID
	if !d.IngressRuleTrace.FoundVerdict() {
		unknownRuleID = calc.NewRuleID(calc.UnknownStr, calc.UnknownStr, calc.UnknownStr, calc.RuleIDIndexUnknown, rules.RuleDirIngress, rules.RuleActionAllow)
	}
	mu := MetricUpdate{
		updateType:    ut,
		tuple:         d.Tuple,
		srcEp:         d.srcEp,
		dstEp:         d.dstEp,
		origSourceIPs: d.origSourceIPs.Copy(),
		ruleIDs:       d.IngressRuleTrace.Path(),
		unknownRuleID: unknownRuleID,
		isConnection:  d.isConnection,
	}
	d.origSourceIPs.Reset()
	return mu
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
