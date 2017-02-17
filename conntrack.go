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
	OriginalTuples []CtTuple
	ReplyTuples    []CtTuple
	Timeout        int
	Mark           int
	Secmark        int
	Status         int
	Use            int
	Id             int
	Zone           int

	OriginalCounters CtCounters
	ReplyCounters    CtCounters

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

func (cte *CtEntry) OriginalTuple() (CtTuple, error) {
	l := len(cte.OriginalTuples)
	if l == 0 {
		return EmptyCtTuple, errors.New("OriginalTuples is empty")
	}
	return cte.OriginalTuples[l-1], nil
}

func (cte *CtEntry) ReplyTuple() (CtTuple, error) {
	l := len(cte.ReplyTuples)
	if l == 0 {
		return EmptyCtTuple, errors.New("ReplyTuples is empty")
	}
	return cte.ReplyTuples[l-1], nil
}

func (cte *CtEntry) OriginalTupleWithoutDNAT() (CtTuple, error) {
	if !cte.IsDNAT() {
		return EmptyCtTuple, errors.New("Entry is not DNAT-ed")
	}
	reply, err := cte.ReplyTuple()
	if err != nil {
		return EmptyCtTuple, err
	}
	original, err := cte.OriginalTuple()
	if err != nil {
		return EmptyCtTuple, err
	}

	var src CtL4Src
	var dst CtL4Dst
	if original.ProtoNum == nfnl.ICMP_PROTO {
		src = CtL4Src{Id: reply.L4Src.Id}
		dst = CtL4Dst{Type: reply.L4Dst.Type, Code: reply.L4Dst.Code}
	} else if original.ProtoNum == nfnl.TCP_PROTO || original.ProtoNum == nfnl.UDP_PROTO {
		src = CtL4Src{Port: reply.L4Dst.Port}
		dst = CtL4Dst{Port: reply.L4Src.Port}
	} else {
		src = CtL4Src{All: reply.L4Dst.All}
		dst = CtL4Dst{All: reply.L4Src.All}
	}

	return CtTuple{original.Src, reply.Src, original.L3ProtoNum, original.ProtoNum, original.Zone, src, dst}, nil
}
