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

// This package re-exports the ETW packet as a struct sot that it can be shimmed and UTs can run on Linux.
package etw

import (
	realetw "github.com/tigera/windows-networking/pkg/etw"
)

type EtwOperations = realetw.EtwOperations
type EventProcessor = realetw.EventProcessor
type PktEvent = realetw.PktEvent
type ServerPort = realetw.ServerPort
type EtwPktProcessor = realetw.EtwPktProcessor

const (
	PKTMON_EVENT_ID_CAPTURE = realetw.PKTMON_EVENT_ID_CAPTURE
)

func NewEtwOperations(eventIDs []int, ep EventProcessor) (*EtwOperations, error) {
	return realetw.NewEtwOperations(eventIDs, ep)
}
