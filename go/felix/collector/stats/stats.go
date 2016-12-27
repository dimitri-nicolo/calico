// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package stats

import (
	"errors"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/ipfix"
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

type RuleAction string

const (
	AllowAction    RuleAction = "allow"
	DenyAction     RuleAction = "deny"
	NextTierAction RuleAction = "next-tier"
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

func (t *RuleTrace) Len() int {
	return len(t.path)
}

func (t *RuleTrace) addRuleTracePoint(tp RuleTracePoint) error {
	existingTp := t.path[tp.Index]
	switch {
	case existingTp == (RuleTracePoint{}):
		// Position is empty, insert and be done.
		log.Debug("Got new rule trace: ", tp)
		t.path[tp.Index] = tp
	case t.Len() < tp.Index:
		// Insertion point greater than current length. Grow and then insert.
		log.Debug("Got new rule trace: ", tp)
		newPath := make([]RuleTracePoint, t.Len()+RuleTraceInitLen)
		copy(newPath, t.path)
		t.path = newPath
		t.path[tp.Index] = tp
	case existingTp == tp:
		// Nothing to do here - maybe a duplicate notification or kernel conntrack
		// expired. Just skip.
	default:
		return RuleTracePointConflict
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
	copy(newPath, t.path[:tp.Index])
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

func (d *Data) ToExportRecord(reason ipfix.FlowEndReasonType) *ipfix.ExportRecord {
	d.clearDirtyFlag()
	rtRecs := []ipfix.RuleTraceRecord{}
	for _, tp := range d.RuleTrace.path {
		if tp == EmptyRuleTracePoint {
			continue
		}
		rtRecs = append(rtRecs, ipfix.RuleTraceRecord{
			TierID:     tp.TierID,
			PolicyID:   tp.PolicyID,
			Rule:       tp.Rule,
			RuleAction: string(tp.Action),
			RuleIndex:  tp.Index,
		})
	}
	return &ipfix.ExportRecord{
		FlowStart:               d.createdAt,
		FlowEnd:                 time.Now(),
		OctetTotalCount:         d.ctrIn.bytes,
		ReverseOctetTotalCount:  d.ctrOut.bytes,
		PacketTotalCount:        d.ctrIn.packets,
		ReversePacketTotalCount: d.ctrOut.packets,

		SourceIPv4Address:      net.ParseIP(d.Tuple.src),
		DestinationIPv4Address: net.ParseIP(d.Tuple.dst),

		SourceTransportPort:      d.Tuple.l4Src,
		DestinationTransportPort: d.Tuple.l4Dst,
		ProtocolIdentifier:       d.Tuple.proto,
		FlowEndReason:            reason,

		RuleTrace: rtRecs,
	}
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
