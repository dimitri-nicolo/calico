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

	var src CtL4Src
	var dst CtL4Dst
	if orig.ProtoNum == nfnl.ICMP_PROTO {
		src = CtL4Src{Id: repl.L4Src.Id}
		dst = CtL4Dst{Type: repl.L4Dst.Type, Code: repl.L4Dst.Code}
	} else if orig.ProtoNum == nfnl.TCP_PROTO || orig.ProtoNum == nfnl.UDP_PROTO {
		src = CtL4Src{Port: repl.L4Dst.Port}
		dst = CtL4Dst{Port: repl.L4Src.Port}
	} else {
		src = CtL4Src{All: repl.L4Dst.All}
		dst = CtL4Dst{All: repl.L4Src.All}
	}

	return CtTuple{orig.Src, repl.Src, orig.L3ProtoNum, orig.ProtoNum, orig.Zone, src, dst}, nil
}
