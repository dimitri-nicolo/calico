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
	"bytes"
	"encoding/binary"
	"io"
	"runtime"

	"github.com/pkg/errors"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/bpf/perf"
)

// Type defines the type of constants used for determinig the type of an event.
type Type uint16

const (
	// MaxCPUs is the currenty supported max number of CPUs
	MaxCPUs = 512

	// TypeLostEvents does not carry any other information except thenumber of lost events.
	TypeLostEvents  Type = iota
	TypeTcpv4Events Type = 1
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

func (r *dataReader) TrimEnd(length int) {
	if length > len(r.data) {
		panic("TrimEnd cannot extend")
	}

	r.data = r.data[:length]
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

func parseTcpStats(tcpStats TCPv4Events) {
	// Parse TCP stats and send it to flow collector
}

type eventHdr struct {
	Type uint16
	Len  uint16
}

type eventTcpStats struct {
	Pid         uint32
	Saddr       uint32
	Daddr       uint32
	Sport       uint16
	Dport       uint16
	TxBytes     uint32
	RxBytes     uint32
	SndBuf      uint32
	RcvBuf      uint32
	ProcessName string
}

func parseEvent(raw eventRaw) (Event, error) {

	var hdr eventHdr

	rd := &dataReader{data: raw.Data()}
	if err := binary.Read(rd, binary.LittleEndian, &hdr); err != nil {
		return nil, errors.New("failed to read event header")
	}

	rd.TrimEnd(int(hdr.Len))

	switch Type(hdr.Type) {
	case TypeTcpv4Events:
		var tcpStats eventTcpStats
		tcpStats.Pid = binary.LittleEndian.Uint32(rd.data[0:4])
		tcpStats.Saddr = binary.LittleEndian.Uint32(rd.data[4:8])
		tcpStats.Daddr = binary.LittleEndian.Uint32(rd.data[8:12])
		tcpStats.Sport = binary.LittleEndian.Uint16(rd.data[12:14])
		tcpStats.Dport = binary.LittleEndian.Uint16(rd.data[14:16])
		tcpStats.TxBytes = binary.LittleEndian.Uint32(rd.data[16:20])
		tcpStats.RxBytes = binary.LittleEndian.Uint32(rd.data[20:24])
		tcpStats.SndBuf = binary.LittleEndian.Uint32(rd.data[24:28])
		tcpStats.RcvBuf = binary.LittleEndian.Uint32(rd.data[28:32])
		tcpStats.ProcessName = string(bytes.Trim(rd.data[32:], "\x00"))
		return TCPv4Events(tcpStats), nil
	default:
		return nil, errors.Errorf("unknown event type: %d", hdr.Type)
	}
}

// LostEvents is an event that reports how many events were missed.
type LostEvents int
type TCPv4Events eventTcpStats

// Type returns TypeLostEvents
func (LostEvents) Type() Type {
	return TypeLostEvents
}

func (TCPv4Events) Type() Type {
	return TypeTcpv4Events
}
