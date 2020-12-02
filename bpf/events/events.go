// Copyright (c) 2020 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"encoding/binary"
	"io"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/perf"
)

// Type defines the type of constants used for determinig the type of an event.
type Type uint16

const (
	// MaxCPUs is the currenty supported max number of CPUs
	MaxCPUs = 512

	// ProcessNameLen max process name length
	ProcessNameLen = 16
	// TypeLostEvents does not carry any other information except the number of lost events.
	TypeLostEvents Type = iota
	//TypeProtoStatsV4 protocol v4 stats
	TypeProtoStatsV4 Type = 1
)

// Event represents the common denominator of all events
type Event interface {
	Type() Type
}

// Source is where do we read the event from
type Source string

const (
	// SourcePerfEvents consumes events using the perf event ring buffer
	SourcePerfEvents Source = "perf-events"
)

type dataReader struct {
	data []byte
	i    int
}

func (r *dataReader) Read(p []byte) (int, error) {
	n := copy(p, r.data[r.i:])
	if n == 0 && len(p) > 0 {
		return 0, io.EOF
	}
	r.i += n
	return n, nil
}

func (r *dataReader) TrimEnd(length int) error {
	if length > len(r.data) {
		return errors.Errorf("TrimEnd cannot extend")
	}

	r.data = r.data[:length]
	return nil
}

func (r *dataReader) TrimHdr() {
	hdrSize := unsafe.Sizeof(eventHdr{})
	r.data = r.data[hdrSize:]
}

func (r *dataReader) Tail() []byte {
	return r.data[r.i:]
}

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
			return nil, errors.WithMessage(err, "failed to get next event")
		}

		if e.LostEvents() != 0 {
			lost := e.LostEvents()
			if len(e.Data()) != 0 {
				// XXX This should not happen, but if it happens, for the sake
				// of simplicity, treat it as another lost event.
				lost++
			}

			return LostEvents(lost), nil
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
	Type uint16
	Len  uint16
}

func parseEvent(raw eventRaw) (Event, error) {

	var hdr eventHdr

	rd := &dataReader{data: raw.Data()}
	if err := binary.Read(rd, binary.LittleEndian, &hdr); err != nil {
		return nil, errors.New("failed to read event header")
	}

	err := rd.TrimEnd(int(hdr.Len))
	if err != nil {
		return nil, err
	}

	rd.TrimHdr()

	switch Type(hdr.Type) {
	case TypeProtoStatsV4:
		return parseProtov4Stats(rd.data)
	default:
		return nil, errors.Errorf("unknown event type: %d", hdr.Type)
	}
}

// LostEvents is an event that reports how many events were missed.
type LostEvents int

// Type returns TypeLostEvents
func (LostEvents) Type() Type {
	return TypeLostEvents
}

type ProtoStatsV4 struct {
	Pid         uint32
	Proto       uint32
	Saddr       uint32
	Daddr       uint32
	Sport       uint16
	Dport       uint16
	TxBytes     uint32
	RxBytes     uint32
	SndBuf      uint32
	RcvBuf      uint32
	ProcessName [ProcessNameLen]byte
}

func (ProtoStatsV4) Type() Type {
	return TypeProtoStatsV4
}

func parseProtov4Stats(raw []byte) (Event, error) {
	var e ProtoStatsV4
	eptr := (unsafe.Pointer)(&e)
	bytes := (*[unsafe.Sizeof(ProtoStatsV4{})]byte)(eptr)
	copy(bytes[:], raw)
	return e, nil
}
