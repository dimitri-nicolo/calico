// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package events

import (
	"encoding/binary"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/projectcalico/calico/felix/bpf"
	"github.com/projectcalico/calico/felix/bpf/perf"
	"github.com/projectcalico/calico/felix/bpf/state"
)

// Type defines the type of constants used for determining the type of an event.
type Type uint16

const (
	// MaxCPUs is the currenty supported max number of CPUs
	MaxCPUs = 512

	// TypeLostEvents does not carry any other information except the number of lost events.
	TypeLostEvents Type = 0
	// TypeProtoStats protocol v4 stats
	TypeProtoStats Type = 1
	// TypeDNSEvent reports information on DNS packets
	TypeDNSEvent Type = 2
	// TypePolicyVerdict is emitted when a policy program reaches a verdict
	TypePolicyVerdict Type = 3
	// TypeTcpStats reports L4 TCP socket information
	TypeTcpStats Type = 4
	// TypeProcessPath reports process exec path, arguments
	TypeProcessPath Type = 5
)

func (t Type) String() string {
	return strconv.Itoa(int(t))
}

// Event represents the common denominator of all events
type Event struct {
	typ  Type
	data []byte
}

// Type returns the event type
func (e Event) Type() Type {
	return e.typ
}

// Data returns the data of the event as an unparsed byte string
func (e Event) Data() []byte {
	return e.data
}

// Source is where do we read the event from
type Source string

const (
	// SourcePerfEvents consumes events using the perf event ring buffer
	SourcePerfEvents Source = "perf-events"
)

type eventRaw interface {
	CPU() int
	Data() []byte
	LostEvents() int
}

// Events is an interface for consuming events
type Events interface {
	Next() (Event, error)
	Map() bpf.Map
	Close() error
}

// New creates a new Events object to consume events.
func New(mc *bpf.MapContext, src Source) (Events, error) {
	switch src {
	case SourcePerfEvents:
		return newPerfEvents(mc)
	}

	return nil, errors.Errorf("unknown events source: %s", src)
}

type perfEventsReader struct {
	events perf.Perf
	bpfMap bpf.Map

	next func() (Event, error)
}

func newPerfEvents(mc *bpf.MapContext) (Events, error) {
	if runtime.NumCPU() > MaxCPUs {
		return nil, errors.Errorf("more cpus (%d) than the max supported (%d)", runtime.NumCPU(), 128)
	}

	perfMap := perf.Map(mc, "perf_evnt", MaxCPUs)
	if err := perfMap.EnsureExists(); err != nil {
		return nil, err
	}

	perfEvents, err := perf.New(perfMap, 1<<20)
	if err != nil {
		return nil, err
	}

	rd := &perfEventsReader{
		events: perfEvents,
		bpfMap: perfMap,
	}

	rd.next = func() (Event, error) {
		e, err := rd.events.Next()
		if err != nil {
			return Event{}, errors.WithMessage(err, "failed to get next event")
		}

		if e.LostEvents() != 0 {
			lost := e.LostEvents()
			if len(e.Data()) != 0 {
				// XXX This should not happen, but if it happens, for the sake
				// of simplicity, treat it as another lost event.
				lost++
			}

			return Event{}, ErrLostEvents(lost)
		}

		return parseEvent(e)
	}

	return rd, nil
}

func (e *perfEventsReader) Close() error {
	return e.events.Close()
}

func (e *perfEventsReader) Next() (Event, error) {
	return e.next()
}

func (e *perfEventsReader) Map() bpf.Map {
	return e.bpfMap
}

type eventHdr struct {
	Type uint32
	Len  uint32
}

type eventTimestampHdr struct {
	eventHdr
	TimestampNS uint64
}

func parseEvent(raw eventRaw) (Event, error) {

	var hdr eventHdr
	hdrBytes := (*[unsafe.Sizeof(eventHdr{})]byte)((unsafe.Pointer)(&hdr))
	data := raw.Data()
	consumed := copy(hdrBytes[:], data)
	l := len(data)
	if int(hdr.Len) > l {
		return Event{}, errors.Errorf("mismatched length %d vs data length %d", hdr.Len, l)
	}
	return Event{
		typ:  Type(hdr.Type),
		data: data[consumed:hdr.Len],
	}, nil
}

// ErrLostEvents reports how many events were lost
type ErrLostEvents int

func (e ErrLostEvents) Error() string {
	return fmt.Sprintf("%d lost events", e)
}

func (e ErrLostEvents) Num() int {
	return int(e)
}

// ParsePolicyVerdict converts a bpf event data and converts to go structure
func ParsePolicyVerdict(data []byte) PolicyVerdict {
	fl := PolicyVerdict{
		SrcAddr:        net.IP(data[0:4]),
		DstAddr:        net.IP(data[8:12]),
		PostNATDstAddr: net.IP(data[12:16]),
		NATTunSrcAddr:  net.IP(data[16:20]),
		PolicyRC:       state.PolicyResult(binary.LittleEndian.Uint32(data[20:24])),
		SrcPort:        binary.LittleEndian.Uint16(data[24:26]),
		DstPort:        binary.LittleEndian.Uint16(data[28:30]),
		PostNATDstPort: binary.LittleEndian.Uint16(data[30:32]),
		IPProto:        uint8(data[32]),
		IPSize:         binary.BigEndian.Uint16(data[34:36]),
		RulesHit:       binary.LittleEndian.Uint32(data[36:40]),
	}

	if fl.RulesHit > state.MaxRuleIDs {
		fl.RulesHit = state.MaxRuleIDs
	}

	off := 40
	for i := 0; i < int(fl.RulesHit); i++ {
		fl.RuleIDs[i] = binary.LittleEndian.Uint64(data[off : off+8])
		off += 8
	}

	return fl
}

// PolicyVerdict describes the policy verdict event and must match the initial part of
// bpf/state.State after the space reserved for the event header.
type PolicyVerdict struct {
	SrcAddr        net.IP
	DstAddr        net.IP
	PostNATDstAddr net.IP
	NATTunSrcAddr  net.IP
	PolicyRC       state.PolicyResult
	SrcPort        uint16
	DstPort        uint16
	PostNATDstPort uint16
	IPProto        uint8
	pad8           uint8
	IPSize         uint16
	RulesHit       uint32
	RuleIDs        [state.MaxRuleIDs]uint64
}

// Type return TypePolicyVerdict
func (PolicyVerdict) Type() Type {
	return TypePolicyVerdict
}
