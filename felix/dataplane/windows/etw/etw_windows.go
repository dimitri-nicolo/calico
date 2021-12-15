// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// This package re-exports the ETW packet as a struct so that it can be shimmed and UTs can run on Linux.
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

var NewEtwOperations = realetw.NewEtwOperations
