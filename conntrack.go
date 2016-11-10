package nfnetlink

import (
	"net"
)

const (
	DIR_IN = iota
	DIR_OUT
	__DIR_MAX
)

type CtCounters struct {
	Packets uint64
	Bytes   uint64
}

type CtL4Src struct {
	Port uint16 // TCP, UDP
	Id   uint16 // ICMP
	All  uint16 // Others
}

type CtL4Dst struct {
	Port uint16 // TCP, UDP
	Type uint8  // ICMP
	Code uint8  // ICMP
	All  uint16 // Others
}

// TODO(doublek): Methods to increment and reset packet counters

type CtTuple struct {
	Src        net.IP
	Dst        net.IP
	L3ProtoNum int
	ProtoNum   int
	Zone       int
	L4Src      CtL4Src
	L4Dst      CtL4Dst
}

type CtNat struct {
	MinIp net.IP
	MaxIP net.IP
	L4Min CtL4Src
	L4Max CtL4Src
}

// TODO(doublek): Methods to compare conntrackTuple's

type CtEntry struct {
	OrigTuples []CtTuple
	ReplTuples []CtTuple
	Timeout    int
	Mark       int
	Secmark    int
	Status     int
	Use        int
	Id         int
	Zone       int

	OrigCounters CtCounters
	ReplCounters CtCounters

	//Snat		CtNat
	//Dnat		CtNat
}
