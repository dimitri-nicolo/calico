// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.

package collector

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gavv/monotime"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

type TrafficDirection string
type RuleDirection string

const (
	TrafficDirInbound  TrafficDirection = "inbound"
	TrafficDirOutbound TrafficDirection = "outbound"
	RuleDirIngress     RuleDirection    = "ingress"
	RuleDirEgress      RuleDirection    = "egress"
	RuleDirUnknown     RuleDirection    = "unknown"
)

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

type RuleAction string

const (
	ActionAllow    RuleAction = "allow"
	ActionDeny     RuleAction = "deny"
	ActionNextTier RuleAction = "pass"
)

const RuleTraceInitLen = 10

var (
	RuleTracePointConflict   = errors.New("Conflict in RuleTracePoint")
	RuleTracePointParseError = errors.New("RuleTracePoint Parse Error")
)

// RuleIDs contains the complete identifiers for a particular rule.
type RuleIDs struct {
	Action    RuleAction
	Tier      string
	Policy    string
	Direction RuleDirection
	Index     string
}

// RuleTracePoint represents a rule and the Tier and a Policy that contains
// it. The `Index` specifies the absolute position of a RuleTracePoint in the
// RuleTrace list. The `EpKey` contains the corresponding workload or host
// endpoint that the Policy applied to.
// Prefix formats are:
// - A/rule Index/profile name
// - A/rule Index/Policy name/Tier name
type RuleTracePoint struct {
	RuleIDs *RuleIDs
	Index   int
	EpKey   interface{}
	Ctr     Counter
}

func NewRuleTracePoint(ruleIDs *RuleIDs, epKey interface{}, tierIndex, numPkts, numBytes int) *RuleTracePoint {
	rtp := &RuleTracePoint{
		RuleIDs: ruleIDs,
		EpKey:   epKey,
		Index:   tierIndex,
	}
	rtp.Ctr.Set(numPkts, numBytes)
	return rtp
}

// Equals compares all but the Ctr field of a RuleTracePoint
func (rtp *RuleTracePoint) Equals(cmpRtp *RuleTracePoint) bool {
	return rtp.RuleIDs == cmpRtp.RuleIDs &&
		rtp.Index == cmpRtp.Index &&
		rtp.EpKey == cmpRtp.EpKey
}

func (rtp *RuleTracePoint) String() string {
	return fmt.Sprintf("tierId='%s' policyId='%s' rule='%s' action='%v' Index=%v Ctr={%v}", rtp.RuleIDs.Tier, rtp.RuleIDs.Policy, rtp.RuleIDs.Index, rtp.RuleIDs.Action, rtp.Index, rtp.Ctr.String())
}

// RuleTrace represents the list of rules (i.e, a Trace) that a packet hits.
// The action of a RuleTrace object is the final RuleTracePoint action that
// is not a next-Tier action. A RuleTrace also contains a workload endpoint,
// which identifies the corresponding endpoint that the rule trace applied to.
type RuleTrace struct {
	path   []*RuleTracePoint
	action RuleAction
	epKey  interface{}
	//TODO: RLB: Do we need this counter?  I think it's always set to the same as the verdict
	// trace point, so why can't we just access that one directly?
	ctr   Counter
	dirty bool

	// Stores the Index of the RuleTracePoint that has a RuleAction Allow or Deny
	verdictIdx int

	// Optimization Note: When initializing a RuleTrace object, the pathArray
	// array is used as the backing array that has been pre-allocated. This avoids
	// one allocation when creating a RuleTrace object. More info on this here:
	// https://github.com/golang/go/wiki/Performance#memory-profiler
	// (Search for "slice array preallocation").
	// RLB: Could we just use a linked list instead - that avoids the need to allocate an array
	pathArray [RuleTraceInitLen]*RuleTracePoint
}

func NewRuleTrace() *RuleTrace {
	rt := &RuleTrace{verdictIdx: -1}
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
	var epStr string
	switch t.epKey.(type) {
	case *model.WorkloadEndpointKey:
		epKey := t.epKey.(*model.WorkloadEndpointKey)
		epStr = fmt.Sprintf("workloadEndpoint={workload=%v endpoint=%v}", epKey.WorkloadID, epKey.EndpointID)
	case *model.HostEndpointKey:
		epKey := t.epKey.(*model.HostEndpointKey)
		epStr = fmt.Sprintf("hostEndpoint={endpoint=%v}", epKey.EndpointID)
	}
	return fmt.Sprintf("path=[%v], action=%v ctr={%v} %s", strings.Join(rtParts, ", "), t.action, t.ctr.String(), epStr)
}

func (t *RuleTrace) Len() int {
	return len(t.path)
}

func (t *RuleTrace) Path() []*RuleTracePoint {
	// Minor optimization where we only do a rebuild when we don't have the full
	// path.
	rebuild := false
	idx := 0
	for i, tp := range t.path {
		if tp == nil {
			rebuild = true
			break
		}
		if tp.RuleIDs.Action == ActionAllow || tp.RuleIDs.Action == ActionDeny {
			idx = i
			break
		}
	}
	if !rebuild {
		return t.path[:idx+1]
	}
	path := make([]*RuleTracePoint, 0, RuleTraceInitLen)
	for _, tp := range t.path {
		if tp == nil {
			continue
		}
		path = append(path, tp)
	}
	return path
}

func (t *RuleTrace) ToString() string {
	p := t.VerdictRuleTracePoint()
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%v", p.RuleIDs.Tier, p.RuleIDs.Policy, p.RuleIDs.Index, p.RuleIDs.Action)
}

func (t *RuleTrace) Action() RuleAction {
	return t.action
}

func (t *RuleTrace) Counters() Counter {
	return t.ctr
}

func (t *RuleTrace) IsDirty() bool {
	return t.dirty
}

// VerdictRuleTracePoint returns the RuleTracePoint that contains either
// ActionAllow or DenyAction in a RuleTrace or nil if we haven't seen
// either of these yet.
func (t *RuleTrace) VerdictRuleTracePoint() *RuleTracePoint {
	if t.verdictIdx >= 0 {
		return t.path[t.verdictIdx]
	} else {
		return nil
	}
}

func (t *RuleTrace) ClearDirtyFlag() {
	t.dirty = false
	t.ctr.ResetDeltas()
	p := t.VerdictRuleTracePoint()
	if p != nil {
		p.Ctr.ResetDeltas()
	}
}

func (t *RuleTrace) addRuleTracePoint(tp *RuleTracePoint) error {
	ctr := tp.Ctr
	if tp.Index > t.Len() {
		// Insertion Index greater than current length. Grow the path slice as long
		// as necessary.
		incSize := (tp.Index / RuleTraceInitLen) * RuleTraceInitLen
		newPath := make([]*RuleTracePoint, t.Len()+incSize)
		copy(newPath, t.path)
		t.path = newPath
		t.path[tp.Index] = tp
	} else {
		existingTp := t.path[tp.Index]
		switch {
		case existingTp == nil:
			// Position is empty, insert and be done.
			t.path[tp.Index] = tp
		case existingTp.Equals(tp):
			p, b := tp.Ctr.Values()
			existingTp.Ctr.Increase(p, b)
			ctr = existingTp.Ctr
		default:
			return RuleTracePointConflict
		}
	}
	if tp.RuleIDs.Action != ActionNextTier {
		t.action = tp.RuleIDs.Action
		//TODO: RLB: This (and the replaceRPT below) means the RT counter is always
		// kept in sync with the verdict RTP.  In which case do we need the complexity
		// of this additional counter?  Now say the policy is updated such that the
		// tier index changes for the verdict RTP, but the actual rule is the same - in
		// this case we lose the counts for the same actual rule (just it's in a different
		// location in the hierarchy).
		t.ctr = ctr
		t.epKey = tp.EpKey
		t.verdictIdx = tp.Index
	}
	t.dirty = true
	return nil
}

func (t *RuleTrace) replaceRuleTracePoint(tp *RuleTracePoint) {
	if tp.RuleIDs.Action == ActionNextTier {
		t.path[tp.Index] = tp
		return
	}
	// New tracepoint is not a next-Tier action truncate at this Index.
	t.path[tp.Index] = tp
	newPath := make([]*RuleTracePoint, t.Len())
	copy(newPath, t.path[:tp.Index+1])
	t.path = newPath
	t.action = tp.RuleIDs.Action
	t.ctr = tp.Ctr
	t.dirty = true
	t.epKey = tp.EpKey
	t.verdictIdx = tp.Index
}

// ToMetricUpdate converts the RuleTrace to a MetricUpdate used by the reporter.
func (rt *RuleTrace) ToMetricUpdate(t Tuple) *MetricUpdate {
	p, b := rt.ctr.Values()
	dp, db := rt.ctr.DeltaValues()
	return &MetricUpdate{
		tuple:        t,
		trafficDir:   TrafficDirOutbound,
		ruleIDs:      rt.VerdictRuleTracePoint().RuleIDs,
		packets:      p,
		bytes:        b,
		deltaPackets: dp,
		deltaBytes:   db,
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

	// These are the counts from conntrack
	connTrackCtr        Counter
	connTrackCtrReverse Counter

	// These contain the aggregated counts per tuple per rule.
	//TODO: RLB: I don't think these need to be pointers, we can just embed the struct directly
	//which saves on a couple of allocations.
	IngressRuleTrace *RuleTrace
	EgressRuleTrace  *RuleTrace

	createdAt  time.Duration
	updatedAt  time.Duration
	ageTimeout time.Duration
	dirty      bool
}

func NewData(tuple Tuple, duration time.Duration) *Data {
	now := monotime.Now()
	return &Data{
		Tuple:            tuple,
		IngressRuleTrace: NewRuleTrace(),
		EgressRuleTrace:  NewRuleTrace(),
		createdAt:        now,
		updatedAt:        now,
		ageTimeout:       duration,
		dirty:            true,
	}
}

func (d *Data) String() string {
	return fmt.Sprintf("tuple={%v}, connTrackCtr={%v}, connTrackCtrReverse={%v}, updatedAt=%v ingressRuleTrace={%v} egressRuleTrace={%v}",
		&(d.Tuple), d.connTrackCtr.String(), d.connTrackCtrReverse.String(), d.updatedAt, d.IngressRuleTrace, d.EgressRuleTrace)
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
func (d *Data) IngressAction() RuleAction {
	return d.IngressRuleTrace.action
}

// Returns the final action of the RuleTrace
func (d *Data) EgressAction() RuleAction {
	return d.EgressRuleTrace.action
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
	d.touch()
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCountersReverse(packets int, bytes int) {
	changed := d.connTrackCtrReverse.Set(packets, bytes)
	if changed {
		d.setDirtyFlag()
	}
	d.touch()
}

// ResetConntrackCounters resets the counters associated with the tracked connection for
// the data.
func (d *Data) ResetConntrackCounters() {
	d.connTrackCtr.Reset()
	d.connTrackCtrReverse.Reset()
}

func (d *Data) AddRuleTracePoint(tp *RuleTracePoint) error {
	var err error
	switch tp.RuleIDs.Direction {
	case RuleDirIngress:
		err = d.IngressRuleTrace.addRuleTracePoint(tp)
	case RuleDirEgress:
		err = d.EgressRuleTrace.addRuleTracePoint(tp)
	default:
		err = fmt.Errorf("unknown rule Direction: %v", tp.RuleIDs.Direction)
	}

	if err == nil {
		d.touch()
		d.setDirtyFlag()
	}
	return err
}

func (d *Data) ReplaceRuleTracePoint(tp *RuleTracePoint) error {
	var err error
	switch tp.RuleIDs.Direction {
	case RuleDirIngress:
		d.IngressRuleTrace.replaceRuleTracePoint(tp)
	case RuleDirEgress:
		d.EgressRuleTrace.replaceRuleTracePoint(tp)
	default:
		err = fmt.Errorf("unknown rule Direction: %v", tp.RuleIDs.Direction)
	}

	if err == nil {
		d.touch()
		d.setDirtyFlag()
	}
	return err
}
