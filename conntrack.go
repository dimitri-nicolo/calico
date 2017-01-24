package nfnetlink

import (
	"errors"
	"net"

	"github.com/tigera/nfnetlink/nfnl"
)

const (
	DIR_IN = iota
	DIR_OUT
	__DIR_MAX
)

type CtCounters struct {
	Packets int
	Bytes   int
}

type CtL4Src struct {
	Port int // TCP, UDP
	Id   int // ICMP
	All  int // Others
}

type CtL4Dst struct {
	Port int // TCP, UDP
	Type int // ICMP
	Code int // ICMP
	All  int // Others
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

var EmptyCtTuple = CtTuple{}

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

func (cte *CtEntry) IsNAT() bool {
	if cte.Status&nfnl.IPS_NAT_MASK != 0 {
		return true
	}
	return false
}

func (cte *CtEntry) IsSNAT() bool {
	if cte.Status&nfnl.IPS_SRC_NAT != 0 {
		return true
	}
	return false
}

func (cte *CtEntry) IsDNAT() bool {
	if cte.Status&nfnl.IPS_DST_NAT != 0 {
		return true
	}
	return false
}

func (cte *CtEntry) OrigTuple() (CtTuple, error) {
	l := len(cte.OrigTuples)
	if l == 0 {
		return EmptyCtTuple, errors.New("OrigTuples is empty")
	}
	return cte.OrigTuples[l-1], nil
}

func (cte *CtEntry) ReplTuple() (CtTuple, error) {
	l := len(cte.ReplTuples)
	if l == 0 {
		return EmptyCtTuple, errors.New("ReplTuples is empty")
	}
	return cte.ReplTuples[l-1], nil
}

func (cte *CtEntry) OrigTupleWithoutDNAT() (CtTuple, error) {
	if !cte.IsDNAT() {
		return EmptyCtTuple, errors.New("Entry is not DNAT-ed")
	}
	repl, err := cte.ReplTuple()
	if err != nil {
		return EmptyCtTuple, err
	}
	orig, err := cte.OrigTuple()
	if err != nil {
		return EmptyCtTuple, err
	}
	return CtTuple{repl.Dst, repl.Src, orig.L3ProtoNum, orig.ProtoNum, orig.Zone, orig.L4Src, orig.L4Dst}, nil
}
