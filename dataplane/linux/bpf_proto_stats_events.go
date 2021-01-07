// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package intdataplane

import (
	"unsafe"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/felix/bpf/events"
)

const (
	// ProcessNameLen max process name length
	ProcessNameLen = 16
)

// EventProtoStatsV4 represets common stats that we can collect for protocols.
type EventProtoStatsV4 struct {
	Pid         uint32
	Proto       uint32
	Saddr       uint32
	Daddr       uint32
	Sport       uint16
	Dport       uint16
	Bytes       uint32
	SndBuf      uint32
	RcvBuf      uint32
	ProcessName [ProcessNameLen]byte
	IsRx        uint32
}

func parseEventProtov4Stats(raw []byte) EventProtoStatsV4 {
	var e EventProtoStatsV4
	eptr := (unsafe.Pointer)(&e)
	bytes := (*[unsafe.Sizeof(EventProtoStatsV4{})]byte)(eptr)
	copy(bytes[:], raw)
	return e
}

func eventProtoStatsV4Sink(e events.Event) {
	// XXX PLace here whatever should happen with these events.
	log.WithField("event", parseEventProtov4Stats(e.Data())).Debug("Received Protocol stats")
}
