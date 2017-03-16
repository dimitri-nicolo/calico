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
	DirIn  Direction = "in"
	DirOut Direction = "out"
)

type Counter struct {
	packets int
	bytes   int
}

func (c Counter) Reset() {
	c.packets = 0
	c.bytes = 0
}

func (c Counter) IsZero() bool {
	return (c.packets == 0 && c.bytes == 0)
}

func (c Counter) String() string {
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

// A RuleTracePoint represents a rule and the tier and a policy that contains
// it. The `Index` specifies the absolute position of a RuleTracePoint in the
// RuleTrace list.
type RuleTracePoint struct {
	TierID   string
	PolicyID string
	Rule     string
	Action   RuleAction
	Index    int
}

func (rtp RuleTracePoint) String() string {
	return fmt.Sprintf("tierId='%v' policyId='%v' rule='%s' action=%v index=%v", rtp.TierID, rtp.PolicyID, rtp.Rule, rtp.Action, rtp.Index)
}

var EmptyRuleTracePoint = RuleTracePoint{}

// Represents the list of rules (i.e, a Trace) that a packet hits. The action
// of a RuleTrace object is the final RuleTracePoint action that is not a
// next-tier action.
type RuleTrace struct {
	path   []RuleTracePoint
	action RuleAction
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
	return fmt.Sprintf("path=[%v], action=%v", strings.Join(rtParts, ", "), t.action)
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
}

// Tuple represents a 5-Tuple value that identifies a connection. This is
// a hashable object and can be used as a map's key.
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

// A Data object contains metadata and statistics such as rule counters and
// age of a connection represented as a Tuple.
// Age Timer Implementation Note: Each Data entry's age is implemented using
// time.Timer. Any actions that modifiy statistics or metadata of a Data entry
// object will extend the life timer of the object. Each method of Data will
// specify if it updates or doesn't update the age timer. When creating a new Data
// object a timeout is specified and this fires when the there have been no updates
// on the object for specified duration.
type Data struct {
	Tuple      Tuple
	WlEpKey    model.WorkloadEndpointKey
	ctrIn      Counter
	ctrOut     Counter
	RuleTrace  *RuleTrace
	createdAt  time.Time
	updatedAt  time.Time
	ageTimeout time.Duration
	ageTimer   *time.Timer
	dirty      bool
}

func NewData(tuple Tuple,
	wlEpKey model.WorkloadEndpointKey,
	inPackets int,
	inBytes int,
	outPackets int,
	outBytes int,
	duration time.Duration) *Data {
	return &Data{
		Tuple:      tuple,
		WlEpKey:    wlEpKey,
		ctrIn:      Counter{packets: inPackets, bytes: inBytes},
		ctrOut:     Counter{packets: outPackets, bytes: outBytes},
		RuleTrace:  NewRuleTrace(),
		createdAt:  time.Now(),
		updatedAt:  time.Now(),
		ageTimeout: duration,
		ageTimer:   time.NewTimer(duration),
		dirty:      true,
	}
}

func (d *Data) String() string {
	return fmt.Sprintf("tuple={%v}, counterIn={%v}, countersOut={%v}, updatedAt=%v ruleTrace={%v} workloadId=%v endpointId=%v",
		&(d.Tuple), d.ctrIn, d.ctrOut, d.updatedAt, d.RuleTrace, d.WlEpKey.WorkloadID, d.WlEpKey.EndpointID)
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
func (d *Data) Action() RuleAction {
	return d.RuleTrace.action
}

func (d *Data) CountersIn() Counter {
	return d.ctrIn
}

func (d *Data) CountersOut() Counter {
	return d.ctrOut
}

func (d *Data) setCountersIn(packets int, bytes int) {
	if packets != d.ctrIn.packets && bytes != d.ctrIn.bytes {
		d.setDirtyFlag()
	}
	d.ctrIn.packets = packets
	d.ctrIn.bytes = bytes
	d.touch()
}

func (d *Data) setCountersOut(packets int, bytes int) {
	if packets != d.ctrOut.packets && bytes != d.ctrOut.bytes {
		d.setDirtyFlag()
	}
	d.ctrOut.packets = packets
	d.ctrOut.bytes = bytes
	d.touch()
}

// Add packets and bytes to the In Counters' values. Use the IncreaseCounters*
// methods when the source of packets/bytes are delta values.
func (d *Data) IncreaseCountersIn(packets int, bytes int) {
	d.setCountersIn(d.ctrIn.packets+packets, d.ctrIn.bytes+bytes)
}

// Add packets and bytes to the Out Counters' values. Use the IncreaseCounters*
// methods when the source of packets/bytes are delta values.
func (d *Data) IncreaseCountersOut(packets int, bytes int) {
	d.setCountersOut(d.ctrOut.packets+packets, d.ctrOut.bytes+bytes)
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCountersIn(packets int, bytes int) {
	d.setCountersIn(packets, bytes)
}

// Set In Counters' values to packets and bytes. Use the SetCounters* methods
// when the source if packets/bytes are absolute values.
func (d *Data) SetCountersOut(packets int, bytes int) {
	d.setCountersOut(packets, bytes)
}

func (d *Data) ResetCounters() {
	d.setCountersIn(0, 0)
	d.setCountersOut(0, 0)
}

func (d *Data) AddRuleTracePoint(tp RuleTracePoint) error {
	err := d.RuleTrace.addRuleTracePoint(tp)
	if err == nil {
		d.touch()
		d.setDirtyFlag()
	}
	return err
}

func (d *Data) ReplaceRuleTracePoint(tp RuleTracePoint) {
	d.RuleTrace.replaceRuleTracePoint(tp)
	d.touch()
	d.setDirtyFlag()
}

type CounterType string

const (
	AbsoluteCounter CounterType = "absolute"
	DeltaCounter    CounterType = "delta"
)

// A StatUpdate represents an statistics update to be made on a `Tuple`.
// All attributes are required. However, when a RuleTracePoint cannot be
// specified, use the `EmptyRuleTracePoint` value to specify this.
// The current StatUpdate doesn't support deletes and all StatUpdate-s are
// either "Add" or "Update" operations.
type StatUpdate struct {
	Tuple      Tuple
	WlEpKey    model.WorkloadEndpointKey
	InPackets  int
	InBytes    int
	OutPackets int
	OutBytes   int
	CtrType    CounterType
	Tp         RuleTracePoint
}

func NewStatUpdate(tuple Tuple,
	wlEpKey model.WorkloadEndpointKey,
	inPackets int,
	inBytes int,
	outPackets int,
	outBytes int,
	ctrType CounterType,
	tp RuleTracePoint) *StatUpdate {
	return &StatUpdate{
		Tuple:      tuple,
		WlEpKey:    wlEpKey,
		InPackets:  inPackets,
		InBytes:    inBytes,
		OutPackets: outPackets,
		OutBytes:   outBytes,
		CtrType:    ctrType,
		Tp:         tp,
	}
}
