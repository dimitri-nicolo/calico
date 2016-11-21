// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package stats

import (
	"errors"
	"net"
	"time"

	"github.com/tigera/libcalico-go-private/lib/backend/model"
)

const TracePathInitLen = 10

var (
	TracePointConflict = errors.New("Conflict in TracePoint")
	TracePointExists   = errors.New("TracePoint Exists")
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

type TracePoint struct {
	TierID   string
	PolicyID string
	Action   RuleAction
	Index    int
}

func (tp *TracePoint) cmp(other *TracePoint) {
}

// Represents a trace of the rules that a packet hit
type Trace struct {
	path   []TracePoint
	action RuleAction
}

func NewTrace() *Trace {
	return &Trace{
		path: make([]TracePoint, TracePathInitLen),
	}
}

func (t *Trace) Len() int {
	return len(t.path)
}

func (t *Trace) add(tp TracePoint) error {
	existingTp := t.path[tp.Index]
	switch {
	case existingTp == (TracePoint{}):
		// Position is empty, insert and be done.
		t.path[tp.Index] = tp
	case t.Len() < tp.Index:
		// Insertion point greater than current length. Grow and then insert.
		newPath := make([]TracePoint, t.Len()+TracePathInitLen)
		copy(newPath, t.path)
		t.path = newPath
		t.path[tp.Index] = tp
	case existingTp == tp:
		// Nothing to do here - maybe a duplicate notification or kernel conntrack
		// expired. Just skip.
	default:
		return TracePointConflict
	}
	if tp.Action != NextTierAction {
		t.action = tp.Action
	}
	return nil
}

func (t *Trace) replace(tp TracePoint) {
	//existingTp := t.path[tp.Index]
	if tp.Action == NextTierAction {
		t.path[tp.Index] = tp
		return
	}
	// New tracepoint is not a next-tier action truncate at this index.
	t.path[tp.Index] = tp
	newPath := make([]TracePoint, t.Len())
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
	tuple      Tuple
	wlEpKey    model.WorkloadEndpointKey
	ctrIn      Counter
	ctrOut     Counter
	trace      *Trace
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
		tuple:      tuple,
		wlEpKey:    wlEpKey,
		ctrIn:      Counter{packets: inPackets, bytes: inBytes},
		ctrOut:     Counter{packets: inPackets, bytes: inBytes},
		trace:      NewTrace(),
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
	d.ageTimer.Reset(d.ageTimeout)
}

func (d *Data) Trace() *Trace {
	return d.trace
}

func (d *Data) Action() RuleAction {
	return d.trace.action
}

func (d *Data) Tuple() Tuple {
	return d.tuple
}

func (d *Data) UpdateCountersIn(packets int, bytes int) {
	d.ctrIn.packets += packets
	d.ctrIn.bytes += bytes
	d.touch()
}

func (d *Data) UpdateCountersOut(packets int, bytes int) {
	d.ctrOut.packets += packets
	d.ctrOut.bytes += bytes
	d.touch()
}

func (d *Data) SetCountersIn(packets int, bytes int) {
	d.ctrIn.packets = packets
	d.ctrIn.bytes = bytes
	d.touch()
}

func (d *Data) SetCountersOut(packets int, bytes int) {
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

func (d *Data) AddTrace(tp TracePoint) error {
	err := d.trace.add(tp)
	if err == nil {
		d.touch()
	}
	return err
}

func (d *Data) ReplaceTrace(tp TracePoint) {
	d.trace.replace(tp)
	d.touch()
}

// TODO(doublek): The current StatUpdate doesn't support deletes. Always
// assumes that it is a add or update.
type StatUpdate struct {
	Tuple      Tuple
	WlEpKey    model.WorkloadEndpointKey
	InPackets  int
	InBytes    int
	OutPackets int
	OutBytes   int
	Dir        int
	Tp         TracePoint
}

func NewStatUpdate(tuple Tuple,
	wlEpKey model.WorkloadEndpointKey,
	inPackets int,
	inBytes int,
	outPackets int,
	outBytes int,
	tp TracePoint) *StatUpdate {
	return &StatUpdate{
		Tuple:      tuple,
		WlEpKey:    wlEpKey,
		InPackets:  inPackets,
		InBytes:    inBytes,
		OutPackets: outPackets,
		OutBytes:   outBytes,
		Tp:         tp,
	}
}
