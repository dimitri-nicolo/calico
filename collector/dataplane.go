// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

package collector

import (
	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/rules"
)

// RuleHit records how many times a rule was hit and how many bytes passed
// through.
type RuleHit struct {
	RuleID *calc.RuleID
	Hits   int
	Bytes  int
}

// PacketInfo is information about a packet we received from the dataplane
type PacketInfo struct {
	Tuple        Tuple
	PreDNATTuple Tuple
	IsDNAT       bool
	Direction    rules.RuleDir
	RuleHits     []RuleHit
}

// PacketInfoReader is an interface for a reader that consumes information
// from dataplane and converts it to the format needed by colelctor
type PacketInfoReader interface {
	Start()
	Chan() <-chan PacketInfo
}

// ConntrackCounters counters for ConntrackInfo
type ConntrackCounters struct {
	Packets int
	Bytes   int
}

// ConntrackInfo is information about a connection from the dataplane.
type ConntrackInfo struct {
	Tuple         Tuple
	PreDNATTuple  Tuple
	IsDNAT        bool
	Expired       bool
	Counters      ConntrackCounters
	ReplyCounters ConntrackCounters
}

// ConntrackInfoReader is an interafce that provides information from conntrack.
type ConntrackInfoReader interface {
	Start()
	Chan() <-chan ConntrackInfo
}
