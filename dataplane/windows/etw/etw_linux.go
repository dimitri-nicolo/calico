// Copyright (c) 2021 Tigera, Inc. All rights reserved.
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

const (
	PKTMON_EVENT_ID_CAPTURE = 0
)

func NewEtwOperations(eventIDs []int, ep EventProcessor) (*EtwOperations, error) {
	return &EtwOperations{}, nil
}
