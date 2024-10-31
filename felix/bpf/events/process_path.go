// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package events

import (
	"bytes"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

const (
	PathMax = 128
	MaxArgs = 5
	ArgLen  = 64
)

// EventProcessPath represents the filePath, arguments of a process
type EventProcessPath struct {
	Pid       uint32
	Filename  [PathMax]byte
	Arguments [MaxArgs][ArgLen]byte
}

type ProcessPath struct {
	Pid       int
	Filename  string
	Arguments string
}

func parseEventProcessPath(raw []byte) EventProcessPath {
	var e EventProcessPath
	eptr := (unsafe.Pointer)(&e)
	bytes := (*[unsafe.Sizeof(EventProcessPath{})]byte)(eptr)
	copy(bytes[:], raw)
	return e
}

type EventProcessPathSink struct {
	outChan chan ProcessPath
}

func NewEventProcessPathSink() *EventProcessPathSink {
	return &EventProcessPathSink{
		outChan: make(chan ProcessPath, 1000),
	}
}

func (sink *EventProcessPathSink) HandleEvent(e Event) {
	parsedEvent := parseEventProcessPath(e.Data())
	var arguments string
	for _, arg := range parsedEvent.Arguments {
		argstr := string(bytes.Trim(arg[:], "\x00"))
		if len(argstr) > 0 {
			if arguments == "" {
				arguments = argstr
			} else {
				arguments = arguments + " " + argstr
			}
		}
	}
	filePath := string(bytes.Trim(parsedEvent.Filename[:], "\x00"))
	processData := ProcessPath{
		Pid:       int(parsedEvent.Pid),
		Filename:  filePath,
		Arguments: arguments,
	}
	if log.GetLevel() == log.DebugLevel {
		log.WithField("event", processData).Debug("Received syscall event")
	}
	sink.outChan <- processData
}

func (sink *EventProcessPathSink) EventProcessPathChan() <-chan ProcessPath {
	return sink.outChan
}
