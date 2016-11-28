// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package stats

import (
	"errors"
	"net"
	"time"

	"github.com/tigera/felix-private/go/felix/ipfix"
	"github.com/tigera/libcalico-go-private/lib/backend/model"
)

const RuleTracePathInitLen = 10

var (
	RuleTracePointConflict = errors.New("Conflict in RuleTracePoint")
	RuleTracePointExists   = errors.New("RuleTracePoint Exists")
)

type RuleAction string

const (
	AllowAction    RuleAction = "allow"
	DenyAction     RuleAction = "deny"
	NextTierAction RuleAction = "next-tier"
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

type RuleTracePoint struct {
	TierID   string
	PolicyID string
	Action   RuleAction
	Index    int
}

var EmptyRuleTracePoint = RuleTracePoint{}

// Represents a trace of the rules that a packet hit
type RuleTrace struct {
	path   []RuleTracePoint
	action RuleAction
}

func NewRuleTrace() *RuleTrace {
	return &RuleTrace{
		path: make([]RuleTracePoint, RuleTracePathInitLen),
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
		t.path[tp.Index] = tp
	case t.Len() < tp.Index:
		// Insertion point greater than current length. Grow and then insert.
		newPath := make([]RuleTracePoint, t.Len()+RuleTracePathInitLen)
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
	}
}

func (d *Data) touch() {
	d.updatedAt = time.Now()
	d.ResetAgeTimeout()
}

func (d *Data) AgeTimer() *time.Timer {
	return d.ageTimer
}

func (d *Data) ResetAgeTimeout() {
	// FIXME(doublek): Resetting a timer is a more complex operation. The call to
	// Reset() here will not work according to docs which define the correct way
	// to do this.
	d.ageTimer.Reset(d.ageTimeout)
}

func (d *Data) Action() RuleAction {
	return d.RuleTrace.action
}

func (d *Data) CountersIn() Counter {
	return d.ctrIn
}

func (d *Data) CountersOut() Counter {
	return d.ctrOut
}

func (d *Data) IncreaseCountersIn(packets int, bytes int) {
	d.ctrIn.packets += packets
	d.ctrIn.bytes += bytes
	d.touch()
}

func (d *Data) IncreaseCountersOut(packets int, bytes int) {
	d.ctrOut.packets += packets
	d.ctrOut.bytes += bytes
	d.touch()
}

func (d *Data) SetCountersIn(packets int, bytes int) {
	if d.ctrIn.packets == packets && d.ctrIn.bytes == bytes {
		// Counters are exactly the same. Don't make any changes.
		return
	}
	d.ctrIn.packets = packets
	d.ctrIn.bytes = bytes
	d.touch()
}

func (d *Data) SetCountersOut(packets int, bytes int) {
	if d.ctrOut.packets == packets && d.ctrOut.bytes == bytes {
		// Counters are exactly the same. Don't make any changes.
		return
	}
	d.ctrOut.packets = packets
	d.ctrOut.bytes = bytes
	d.touch()
}

func (d *Data) ResetCounters() {
	d.ctrIn.packets = 0
	d.ctrIn.bytes = 0
	d.ctrOut.packets = 0
	d.ctrOut.bytes = 0
	d.touch()
}

func (d *Data) AddRuleTracePoint(tp RuleTracePoint) error {
	err := d.RuleTrace.addRuleTracePoint(tp)
	if err == nil {
		d.touch()
	}
	return err
}

func (d *Data) ReplaceRuleTracePoint(tp RuleTracePoint) {
	d.RuleTrace.replaceRuleTracePoint(tp)
	d.touch()
}

func (d *Data) ToExportRecord(reason ipfix.FlowEndReasonType) *ipfix.ExportRecord {
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
	}
}

type CounterType string

const (
	AbsoluteCounter CounterType = "absolute"
	DeltaCounter    CounterType = "delta"
)

// TODO(doublek): The current StatUpdate doesn't support deletes. Always
// assumes that it is a add or update.
type StatUpdate struct {
	Tuple      Tuple
	WlEpKey    model.WorkloadEndpointKey
	InPackets  int
	InBytes    int
	OutPackets int
	OutBytes   int
	CtrType    CounterType
	Dir        int
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
