// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// Dummy version of the etw API for compilation on Linux.
package etw

type EventProcessor interface {
	SessionName() string
}

type EtwPktProcessor string

func (d EtwPktProcessor) SessionName() string {
	return string(d)
}

type PktEvent struct{}

func (p PktEvent) NanoSeconds() uint64 {
	return 20568700
}

func (p PktEvent) Payload() []byte {
	return []byte{0, 0}
}

type ServerPort struct {
	IP   string
	Port uint16
}

type EtwOperations struct{}

func (t *EtwOperations) SubscribeToPktMon(ch chan<- *PktEvent, done <-chan struct{}, serverPorts []ServerPort) error {
	return nil
}

func (t *EtwOperations) WaitForSessionClose() {
	return
}

const (
	PKTMON_EVENT_ID_CAPTURE = 0
)

func NewEtwOperations(eventIDs []int, ep EventProcessor) (*EtwOperations, error) {
	return &EtwOperations{}, nil
}
