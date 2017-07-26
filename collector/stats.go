// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gavv/monotime"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

type Direction string

const (
	DirIn      Direction = "in"
	DirOut     Direction = "out"
	DirUnknown Direction = "unknown"
)

// Counter stores packet and byte statistics. It also maintains a delta of
// changes made until a `Reset` is called. They can never be set to 0, except
// when creating a new Counter.
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

// DeltaValues returns the packet and byte deltas since the last `Reset()`.
func (c *Counter) DeltaValues() (packets, bytes int) {
	return c.deltaPackets, c.deltaBytes
}

// Set packet and byte values to `packets` and `bytes`. Returns true if value
// was changed and false otherwise.
func (c *Counter) Set(packets int, bytes int) (changed bool) {
	if packets != 0 && bytes != 0 && packets > c.packets && bytes > c.bytes {
		changed = true
		dp := packets - c.packets
		db := bytes - c.bytes
		c.packets = packets
		c.bytes = bytes
		c.deltaPackets += dp
		c.deltaBytes += db
	}
	return
}

// Increase packet and byte values by `packets` and `bytes`. Always returns
// true.
func (c *Counter) Increase(packets int, bytes int) (changed bool) {
	c.Set(c.packets+packets, c.bytes+bytes)
	changed = true
	return
}

// Reset sets the delta packet and byte values to zero. Non-delta packet and
// bytes cannot be reset or set to zero.
func (c *Counter) Reset() {
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
	AllowAction    RuleAction = "allow"
	DenyAction     RuleAction = "deny"
	NextTierAction RuleAction = "pass"
)

var RuleActionToBytes = map[RuleAction][]byte{
	AllowAction:    []byte("allow"),
	DenyAction:     []byte("deny"),
	NextTierAction: []byte("pass"),
}

const RuleTraceInitLen = 10

var (
	RuleTracePointConflict   = errors.New("Conflict in RuleTracePoint")
	RuleTracePointExists     = errors.New("RuleTracePoint Exists")
	RuleTracePointParseError = errors.New("RuleTracePoint Parse Error")
)

var ruleSep = byte('/')

// RuleTracePoint represents a rule and the tier and a policy that contains
// it. The `Index` specifies the absolute position of a RuleTracePoint in the
// RuleTrace list. The `EpKey` contains the corresponding workload or host
// endpoint that the policy applied to.
// Prefix formats are:
// - A/rule index/profile name
// - A/rule index/policy name/tier name
type RuleTracePoint struct {
	prefix    [64]byte
	pfxlen    int
	tierIdx   int
	policyIdx int
	ruleIdx   int
	Action    RuleAction
	Index     int
	EpKey     interface{}
	Ctr       Counter
}

func lookupAction(action byte) RuleAction {
	switch action {
	case 'A':
		return AllowAction
	case 'D':
		return DenyAction
	case 'N':
		return NextTierAction
	default:
		log.Errorf("Unknown action %v", action)
		return NextTierAction
	}
}

func NewRuleTracePoint(prefix [64]byte, prefixLen int, epKey interface{}) (*RuleTracePoint, error) {
	pfxlen := prefixLen
	// Should have at least 2 separators, a action character and a rule (assuming
	// we allow empty policy names).
	if pfxlen < 4 {
		return nil, RuleTracePointParseError
	}
	action := lookupAction(prefix[0])
	ruleIdx := 2
	policySep := bytes.IndexByte(prefix[ruleIdx:], ruleSep)
	if policySep == -1 {
		return nil, RuleTracePointParseError
	}
	policyIdx := ruleIdx + policySep + 1
	tierSep := bytes.IndexByte(prefix[policyIdx:], ruleSep)
	var tierIdx int
	if tierSep == -1 {
		tierIdx = -1
	} else {
		tierIdx = policyIdx + tierSep + 1
	}
	rtp := &RuleTracePoint{}
	rtp.prefix = prefix
	rtp.pfxlen = pfxlen
	rtp.ruleIdx = ruleIdx
	rtp.policyIdx = policyIdx
	rtp.Action = action
	rtp.tierIdx = tierIdx
	rtp.EpKey = epKey
	return rtp, nil
}

func (rtp *RuleTracePoint) TierID() []byte {
	if rtp.tierIdx == -1 {
		return []byte("profile")
	} else {
		return rtp.prefix[rtp.tierIdx:rtp.pfxlen]
	}
}

func (rtp *RuleTracePoint) PolicyID() []byte {
	var t int
	if rtp.tierIdx == -1 {
		t = rtp.pfxlen
	} else {
		t = rtp.tierIdx - 1
	}
	return rtp.prefix[rtp.policyIdx:t]
}

func (rtp *RuleTracePoint) Rule() []byte {
	return rtp.prefix[rtp.ruleIdx : rtp.policyIdx-1]
}

// Equals compares all but the Ctr field of a RuleTracePoint
func (rtp *RuleTracePoint) Equals(cmpRtp *RuleTracePoint) bool {
	return rtp.prefix == cmpRtp.prefix &&
		rtp.Action == cmpRtp.Action &&
		rtp.Index == cmpRtp.Index &&
		rtp.EpKey == cmpRtp.EpKey
}

func (rtp *RuleTracePoint) String() string {
	return fmt.Sprintf("tierId='%s' policyId='%s' rule='%s' action=%v index=%v ctr={%v}", rtp.TierID(), rtp.PolicyID(), rtp.Rule(), rtp.Action, rtp.Index, rtp.Ctr.String())
}

var EmptyRuleTracePoint = RuleTracePoint{}

// RuleTrace represents the list of rules (i.e, a Trace) that a packet hits.
// The action of a RuleTrace object is the final RuleTracePoint action that
// is not a next-tier action. A RuleTrace also contains a workload endpoint,
// which identifies the corresponding endpoint that the rule trace applied to.
type RuleTrace struct {
	path   []*RuleTracePoint
	action RuleAction
	epKey  interface{}
	ctr    Counter
	dirty  bool

	// Stores the index of the RuleTracePoint that has a RuleAction Allow or Deny
	verdictIdx int

	// Optimization Note: When initializing a RuleTrace object, the pathArray
	// array is used as the backing array that has been pre-allocated. This avoids
	// one allocation when creating a RuleTrace object. More info on this here:
	// https://github.com/golang/go/wiki/Performance#memory-profiler
	// (Search for "slice array preallocation").
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
	// Minor optimiztion where we only do a rebuild when we don't have the full
	// path.
	rebuild := false
	idx := 0
	for i, tp := range t.path {
		if tp == nil {
			rebuild = true
			break
		}
		if tp.Action == AllowAction || tp.Action == DenyAction {
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
	return fmt.Sprintf("%s/%s/%s/%v", p.TierID(), p.PolicyID(), p.Rule(), p.Action)
}

func (t *RuleTrace) ConcatBytes(buf *bytes.Buffer) {
	p := t.VerdictRuleTracePoint()
	buf.Write(p.TierID())
	buf.Write([]byte("/"))
	buf.Write(p.PolicyID())
	buf.Write([]byte("/"))
	buf.Write(p.Rule())
	buf.Write([]byte("/"))
	buf.Write(RuleActionToBytes[p.Action])
	return
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
// AllowAction or DenyAction in a RuleTrace or nil if we haven't seen
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
	t.ctr.Reset()
	p := t.VerdictRuleTracePoint()
	if p != nil {
		p.Ctr.Reset()
	}
}

func (t *RuleTrace) addRuleTracePoint(tp *RuleTracePoint) error {
	var ctr Counter
	ctr = tp.Ctr
	if tp.Index > t.Len() {
		// Insertion index greater than current length. Grow the path slice as long
		// as necessary.
		newPath := make([]*RuleTracePoint, tp.Index)
		copy(newPath, t.path)
		nextSize := (tp.Index / RuleTraceInitLen) * RuleTraceInitLen
		t.path = append(t.path, make([]*RuleTracePoint, nextSize)...)
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
	if tp.Action != NextTierAction {
		t.action = tp.Action
		t.ctr = ctr
		t.epKey = tp.EpKey
		t.verdictIdx = tp.Index
	}
	t.dirty = true
	return nil
}

func (t *RuleTrace) replaceRuleTracePoint(tp *RuleTracePoint) {
	if tp.Action == NextTierAction {
		t.path[tp.Index] = tp
		return
	}
	// New tracepoint is not a next-tier action truncate at this index.
	t.path[tp.Index] = tp
	newPath := make([]*RuleTracePoint, t.Len())
	copy(newPath, t.path[:tp.Index+1])
	t.path = newPath
	t.action = tp.Action
	t.ctr = tp.Ctr
	t.dirty = true
	t.epKey = tp.EpKey
	t.verdictIdx = tp.Index
}

// Tuple represents a 5-Tuple value that identifies a connection/flow of packets
// with an implicit notion of direction that comes with the use of a source and
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
// where the policy was applied, with additional information on corresponding
// workload endpoint. The EgressRuleTrace and the IngressRuleTrace record the
// policies that this tuple can hit - egress from the workload that is the
// source of this connection and ingress into a workload that terminated this.
// - 2 counters - ctr for the direction of the connection and ctrReverse for
// the reverse/reply of this connection.
type Data struct {
	Tuple            Tuple
	ctr              Counter
	ctrReverse       Counter
	IngressRuleTrace *RuleTrace
	EgressRuleTrace  *RuleTrace
	createdAt        time.Duration
	updatedAt        time.Duration
	ageTimeout       time.Duration
	dirty            bool
}

func NewData(tuple Tuple, duration time.Duration) *Data {
	now := monotime.Now()
	return &Data{
		Tuple:            tuple,
		ctr:              *NewCounter(0, 0),
		ctrReverse:       *NewCounter(0, 0),
		IngressRuleTrace: NewRuleTrace(),
		EgressRuleTrace:  NewRuleTrace(),
		createdAt:        now,
		updatedAt:        now,
		ageTimeout:       duration,
		dirty:            true,
	}
}

func (d *Data) String() string {
	return fmt.Sprintf("tuple={%v}, counters={%v}, countersReverse={%v}, updatedAt=%v ingressRuleTrace={%v} egressRuleTrace={%v}",
		&(d.Tuple), d.ctr.String(), d.ctrReverse.String(), d.updatedAt, d.IngressRuleTrace, d.EgressRuleTrace)
}

func (d *Data) touch() {
	d.updatedAt = monotime.Now()
}

func (d *Data) setDirtyFlag() {
	d.dirty = true
}

func (d *Data) clearDirtyFlag() {
	d.dirty = false
	d.ctr.Reset()
	d.ctrReverse.Reset()
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
	return d.ctr
}

func (d *Data) CountersReverse() Counter {
	return d.ctrReverse
}

// Add packets and bytes to the Counters' values. Use the IncreaseCounters*
// methods when the source of packets/bytes are delta values.
func (d *Data) IncreaseCounters(packets int, bytes int) {
	d.ctr.Increase(packets, bytes)
	d.setDirtyFlag()
	d.touch()
}

// Add packets and bytes to the Reverse Counters' values. Use the IncreaseCounters*
// methods when the source of packets/bytes are delta values.
func (d *Data) IncreaseCountersReverse(packets int, bytes int) {
	d.ctrReverse.Increase(packets, bytes)
	d.setDirtyFlag()
	d.touch()
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCounters(packets int, bytes int) {
	changed := d.ctr.Set(packets, bytes)
	if changed {
		d.setDirtyFlag()
	}
	d.touch()
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCountersReverse(packets int, bytes int) {
	changed := d.ctrReverse.Set(packets, bytes)
	if changed {
		d.setDirtyFlag()
	}
	d.touch()
}

func (d *Data) ResetCounters() {
	d.ctr = *NewCounter(0, 0)
	d.ctrReverse = *NewCounter(0, 0)
}

func (d *Data) AddRuleTracePoint(tp *RuleTracePoint, dir Direction) error {
	var err error
	if dir == DirIn {
		err = d.IngressRuleTrace.addRuleTracePoint(tp)
	} else {
		err = d.EgressRuleTrace.addRuleTracePoint(tp)
	}
	if err == nil {
		d.touch()
		d.setDirtyFlag()
	}
	return err
}

func (d *Data) ReplaceRuleTracePoint(tp *RuleTracePoint, dir Direction) {
	if dir == DirIn {
		d.IngressRuleTrace.replaceRuleTracePoint(tp)
	} else {
		d.EgressRuleTrace.replaceRuleTracePoint(tp)
	}
	d.touch()
	d.setDirtyFlag()
}

type CounterType string

const (
	AbsoluteCounter CounterType = "absolute"
	DeltaCounter    CounterType = "delta"
)
