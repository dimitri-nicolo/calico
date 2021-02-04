// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package events

import (
	"unsafe"

	log "github.com/sirupsen/logrus"
)

const (
	// ProcessNameLen max process name length
	ProcessNameLen = 16
)

// EventProtoStats represets common stats that we can collect for protocols.
type EventProtoStats struct {
	Pid         uint32
	Proto       uint32
	Saddr       [16]byte
	Daddr       [16]byte
	Sport       uint16
	Dport       uint16
	Bytes       uint32
	SndBuf      uint32
	RcvBuf      uint32
	ProcessName [ProcessNameLen]byte
	IsRx        uint32
}

func parseEventProtov4Stats(raw []byte) EventProtoStats {
	var e EventProtoStats
	eptr := (unsafe.Pointer)(&e)
	bytes := (*[unsafe.Sizeof(EventProtoStats{})]byte)(eptr)
	copy(bytes[:], raw)
	return e
}

type EventProtoStatsSink struct {
	outChan chan EventProtoStats
}

func NewEventProtoStatsSink() *EventProtoStatsSink {
	return &EventProtoStatsSink{
		outChan: make(chan EventProtoStats, 1000),
	}
}

func (sink *EventProtoStatsSink) HandleEvent(e Event) {
	parsedEvent := parseEventProtov4Stats(e.Data())
	if log.GetLevel() == log.DebugLevel {
		log.WithField("event", parsedEvent).Debug("Received Protocol stats")
	}
	sink.outChan <- parsedEvent
}

func (sink *EventProtoStatsSink) EventProtoStatsChan() <-chan EventProtoStats {
	return sink.outChan
}
