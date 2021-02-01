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

// EventProtoStatsV4 represets common stats that we can collect for protocols.
type EventProtoStatsV4 struct {
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

func parseEventProtov4Stats(raw []byte) EventProtoStatsV4 {
	var e EventProtoStatsV4
	eptr := (unsafe.Pointer)(&e)
	bytes := (*[unsafe.Sizeof(EventProtoStatsV4{})]byte)(eptr)
	copy(bytes[:], raw)
	return e
}

type EventProtoStatsV4Sink struct {
	outChan chan EventProtoStatsV4
}

func NewEventProtoStatsV4Sink() *EventProtoStatsV4Sink {
	return &EventProtoStatsV4Sink{
		outChan: make(chan EventProtoStatsV4, 1000),
	}
}

func (sink *EventProtoStatsV4Sink) HandleEvent(e Event) {
	parsedEvent := parseEventProtov4Stats(e.Data())
	if log.GetLevel() == log.DebugLevel {
		log.WithField("event", parsedEvent).Debug("Received Protocol stats")
	}
	sink.outChan <- parsedEvent
}

func (sink *EventProtoStatsV4Sink) EventProtoStatsV4Chan() <-chan EventProtoStatsV4 {
	return sink.outChan
}
