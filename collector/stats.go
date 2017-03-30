// Copyright (c) 2016-2017 Tigera, Inc. All rights reserved.

package collector

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

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
		db := packets - c.packets
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

const RuleTraceInitLen = 10

var (
	RuleTracePointConflict = errors.New("Conflict in RuleTracePoint")
	RuleTracePointExists   = errors.New("RuleTracePoint Exists")
)

// RuleTracePoint represents a rule and the tier and a policy that contains
// it. The `Index` specifies the absolute position of a RuleTracePoint in the
// RuleTrace list. The `WlEpKey` contains the corresponding workload endpoint
// that the policy applied to.
type RuleTracePoint struct {
	TierID   string
	PolicyID string
	Rule     string
	Action   RuleAction
	Index    int
	WlEpKey  model.WorkloadEndpointKey
}

func (rtp RuleTracePoint) String() string {
	return fmt.Sprintf("tierId='%v' policyId='%v' rule='%s' action=%v index=%v", rtp.TierID, rtp.PolicyID, rtp.Rule, rtp.Action, rtp.Index)
}

var EmptyRuleTracePoint = RuleTracePoint{}

// RuleTrace represents the list of rules (i.e, a Trace) that a packet hits.
// The action of a RuleTrace object is the final RuleTracePoint action that
// is not a next-tier action. A RuleTrace also contains a workload endpoint,
// which identifies the corresponding endpoint that the rule trace applied to.
type RuleTrace struct {
	path    []RuleTracePoint
	action  RuleAction
	wlEpKey model.WorkloadEndpointKey
}

func NewRuleTrace() *RuleTrace {
	return &RuleTrace{
		path: make([]RuleTracePoint, RuleTraceInitLen),
	}
}

func (t *RuleTrace) String() string {
	rtParts := make([]string, 0)
	for _, tp := range t.path {
		if tp == EmptyRuleTracePoint {
			continue
		}
		rtParts = append(rtParts, fmt.Sprintf("(%v)", tp))
	}
	return fmt.Sprintf("path=[%v], action=%v workloadEndpoint={workload=%v endpoint=%v}",
		strings.Join(rtParts, ", "), t.action, t.wlEpKey.WorkloadID, t.wlEpKey.EndpointID)
}

func (t *RuleTrace) Len() int {
	return len(t.path)
}

func (t *RuleTrace) Path() []RuleTracePoint {
	path := []RuleTracePoint{}
	for _, tp := range t.path {
		if tp == EmptyRuleTracePoint {
			continue
		}
		path = append(path, tp)
	}
	return path
}

func (t *RuleTrace) ToString() string {
	path := t.Path()
	p := path[len(path)-1]
	return fmt.Sprintf("%v/%v/%v/%v", p.TierID, p.PolicyID, p.Rule, p.Action)
}

func (t *RuleTrace) addRuleTracePoint(tp RuleTracePoint) error {
	if tp.Index > t.Len() {
		log.Debug("Got new rule trace: ", tp)
		// Insertion index greater than current length. Grow the path slice as long
		// as necessary.
		newPath := make([]RuleTracePoint, tp.Index)
		copy(newPath, t.path)
		nextSize := (tp.Index / RuleTraceInitLen) * RuleTraceInitLen
		t.path = append(t.path, make([]RuleTracePoint, nextSize)...)
		t.path[tp.Index] = tp
	} else {
		existingTp := t.path[tp.Index]
		switch {
		case existingTp == (RuleTracePoint{}):
			// Position is empty, insert and be done.
			log.Debug("Got new rule trace: ", tp)
			t.path[tp.Index] = tp
		case existingTp == tp:
			// Nothing to do here - maybe a duplicate notification or kernel conntrack
			// expired. Just skip.
		default:
			return RuleTracePointConflict
		}
	}
	if tp.Action != NextTierAction {
		t.action = tp.Action
	}
	t.wlEpKey = tp.WlEpKey
	return nil
}

func (t *RuleTrace) replaceRuleTracePoint(tp RuleTracePoint) {
	log.Debug("Replacing rule trace: ", tp)
	if tp.Action == NextTierAction {
		t.path[tp.Index] = tp
		return
	}
	// New tracepoint is not a next-tier action truncate at this index.
	t.path[tp.Index] = tp
	newPath := make([]RuleTracePoint, t.Len())
	copy(newPath, t.path[:tp.Index+1])
	t.path = newPath
	t.action = tp.Action
	t.wlEpKey = tp.WlEpKey
}

// Tuple represents a 5-Tuple value that identifies a connection/flow of packets
// with an implicit notion of direction that comes with the use of a source and
// destination. This is a hashable object and can be used as a map's key.
type Tuple struct {
	src   string
	dst   string
	proto int
	l4Src int
	l4Dst int
}

func NewTuple(src net.IP, dst net.IP, proto int, l4Src int, l4Dst int) *Tuple {
	return &Tuple{
		src:   src.String(),
		dst:   dst.String(),
		proto: proto,
		l4Src: l4Src,
		l4Dst: l4Dst,
	}
}

func (t *Tuple) String() string {
	return fmt.Sprintf("src=%v dst=%v proto=%v sport=%v dport=%v", t.src, t.dst, t.proto, t.l4Src, t.l4Dst)
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
// Age Timer Implementation Note: Each Data entry's age is implemented using
// time.Timer. Any actions that modifiy statistics or metadata of a Data entry
// object will extend the life timer of the object. Each method of Data will
// specify if it updates or doesn't update the age timer. When creating a new Data
// object a timeout is specified and this fires when the there have been no updates
// on the object for specified duration.
type Data struct {
	Tuple            Tuple
	ctr              Counter
	ctrReverse       Counter
	IngressRuleTrace *RuleTrace
	EgressRuleTrace  *RuleTrace
	createdAt        time.Time
	updatedAt        time.Time
	ageTimeout       time.Duration
	ageTimer         *time.Timer
	dirty            bool
}

func NewData(tuple Tuple,
	packets int,
	bytes int,
	reversePackets int,
	reverseBytes int,
	duration time.Duration) *Data {
	return &Data{
		Tuple:            tuple,
		ctr:              *NewCounter(packets, bytes),
		ctrReverse:       *NewCounter(reversePackets, reverseBytes),
		IngressRuleTrace: NewRuleTrace(),
		EgressRuleTrace:  NewRuleTrace(),
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
		ageTimeout:       duration,
		ageTimer:         time.NewTimer(duration),
		dirty:            true,
	}
}

func (d *Data) String() string {
	return fmt.Sprintf("tuple={%v}, counters={%v}, countersReverse={%v}, updatedAt=%v ingressRuleTrace={%v} egressRuleTrace={%v}",
		&(d.Tuple), d.ctr.String(), d.ctrReverse.String(), d.updatedAt, d.IngressRuleTrace, d.EgressRuleTrace)
}

func (d *Data) touch() {
	d.updatedAt = time.Now()
	d.resetAgeTimeout()
}

func (d *Data) setDirtyFlag() {
	d.dirty = true
}

func (d *Data) clearDirtyFlag() {
	d.dirty = false
	d.ctr.Reset()
	d.ctrReverse.Reset()
}

func (d *Data) IsDirty() bool {
	return d.dirty
}

func (d *Data) resetAgeTimeout() {
	// FIXME(doublek): Resetting a timer is a more complex operation. The call to
	// Reset() here will not work according to docs which define the correct way
	// to do this.
	d.ageTimer.Reset(d.ageTimeout)
}

// Return the internal Timer object. Use the Timers internal channel to detect
// when the object's age expires.
func (d *Data) AgeTimer() *time.Timer {
	return d.ageTimer
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

func (d *Data) AddRuleTracePoint(tp RuleTracePoint, dir Direction) error {
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

func (d *Data) ReplaceRuleTracePoint(tp RuleTracePoint, dir Direction) {
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

// StatUpdate represents an statistics update to be made on a `Tuple`.
// All attributes are required. However, when a RuleTracePoint cannot be
// specified, use the `EmptyRuleTracePoint` value to specify this.
// Specify if the Packet and Byte counts included in the update are either
// AbsoluteCounter or DeltaCounter using the CtrType field.
// The current StatUpdate doesn't support deletes and all StatUpdate-s are
// either "Add" or "Update" operations.
type StatUpdate struct {
	Tuple          Tuple
	Packets        int
	Bytes          int
	ReversePackets int
	ReverseBytes   int
	CtrType        CounterType
	Dir            Direction
	Tp             RuleTracePoint
}

func NewStatUpdate(tuple Tuple,
	packets int,
	bytes int,
	reversePackets int,
	reverseBytes int,
	ctrType CounterType,
	dir Direction,
	tp RuleTracePoint) *StatUpdate {
	return &StatUpdate{
		Tuple:          tuple,
		Packets:        packets,
		Bytes:          bytes,
		ReversePackets: reversePackets,
		ReverseBytes:   reverseBytes,
		CtrType:        ctrType,
		Dir:            dir,
		Tp:             tp,
	}
}
