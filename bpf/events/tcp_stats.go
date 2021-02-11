// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package events

import (
	"unsafe"

	log "github.com/sirupsen/logrus"
)

// EventTcpStats represets common stats that we can collect for tcp sockets.
type EventTcpStats struct {
	Saddr             [16]byte
	Daddr             [16]byte
	Sport             uint16
	Dport             uint16
	SendCongestionWnd uint32
	SmoothRtt         uint32
	MinRtt            uint32
	Ssthresh          uint32
	Mss               uint32
	Ecnf              uint32
	TotalRetrans      uint32
	LostOut           uint32
	IsckRetrans       uint32
}

func parseEventTcpStats(raw []byte) EventTcpStats {
	var e EventTcpStats
	eptr := (unsafe.Pointer)(&e)
	bytes := (*[unsafe.Sizeof(EventTcpStats{})]byte)(eptr)
	copy(bytes[:], raw)
	return e
}

type EventTcpStatsSink struct {
	outChan chan EventTcpStats
}

func NewEventTcpStatsSink() *EventTcpStatsSink {
	return &EventTcpStatsSink{
		outChan: make(chan EventTcpStats, 1000),
	}
}

func (sink *EventTcpStatsSink) HandleEvent(e Event) {
	parsedEvent := parseEventTcpStats(e.Data())
	if log.GetLevel() == log.DebugLevel {
		log.WithField("event", parsedEvent).Debug("Received TCP stats")
	}
	sink.outChan <- parsedEvent
}

func (sink *EventTcpStatsSink) EventTcpStatsChan() <-chan EventTcpStats {
	return sink.outChan
}
