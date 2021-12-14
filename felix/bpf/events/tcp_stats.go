// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package events

import (
	"encoding/binary"

	log "github.com/sirupsen/logrus"
)

// EventTcpStats represets common stats that we can collect for tcp sockets.
type EventTcpStats struct {
	Saddr             [16]byte
	Daddr             [16]byte
	Sport             uint16
	Dport             uint16
	SendCongestionWnd int
	SmoothRtt         int
	MinRtt            int
	Mss               int
	TotalRetrans      int
	LostOut           int
	UnrecoveredRTO    int
}

func parseEventTcpStats(raw []byte) EventTcpStats {
	e := EventTcpStats{
		Sport:             binary.LittleEndian.Uint16(raw[32:34]),
		Dport:             binary.LittleEndian.Uint16(raw[34:36]),
		SendCongestionWnd: int(binary.LittleEndian.Uint32(raw[36:40])),
		SmoothRtt:         int(binary.LittleEndian.Uint32(raw[40:44])),
		MinRtt:            int(binary.LittleEndian.Uint32(raw[44:48])),
		Mss:               int(binary.LittleEndian.Uint32(raw[48:52])),
		TotalRetrans:      int(binary.LittleEndian.Uint32(raw[52:56])),
		LostOut:           int(binary.LittleEndian.Uint32(raw[56:60])),
		UnrecoveredRTO:    int(binary.LittleEndian.Uint32(raw[60:64])),
	}
	copy(e.Saddr[:], raw[0:16])
	copy(e.Daddr[:], raw[16:32])
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
