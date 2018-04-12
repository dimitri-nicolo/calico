package nfnetlink

import (
	"encoding/binary"
	"errors"
	"net"
	"syscall"

	"github.com/tigera/nfnetlink/nfnl"
	"github.com/vishvananda/netlink/nl"
)

type ConntrackEntryHandler func(cte CtEntry)

func ConntrackList(ceh ConntrackEntryHandler) error {
	nlMsgType := nfnl.NFNL_SUBSYS_CTNETLINK<<8 | nfnl.IPCTNL_MSG_CT_GET
	nlMsgFlags := syscall.NLM_F_REQUEST | syscall.NLM_F_DUMP
	// TODO(doublek): Look into how vishvananda/netlink/handle_linux.go to reuse sockets
	req := nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	nfgenmsg := nfnl.NewNfGenMsg(syscall.AF_INET, nfnl.NFNETLINK_V0, 0)
	req.AddData(nfgenmsg)

	msgs, err := req.Execute(syscall.NETLINK_NETFILTER, 0)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		msg := nfnl.DeserializeNfGenMsg(m)
		ctentry, err := conntrackEntryFromNfAttrs(m[msg.Len():], msg.Family)
		if err != nil {
			return err
		}
		ceh(ctentry)

	}
	return nil
}

func conntrackEntryFromNfAttrs(m []byte, family uint8) (CtEntry, error) {
	ctentry := CtEntry{}
	var attrs [nfnl.CTA_MAX]nfnl.NetlinkNetfilterAttr
	err := nfnl.ParseNetfilterAttr(m, attrs[:])
	if err != nil {
		return ctentry, err
	}

	native := binary.BigEndian

	// Start building a Conntrack Entry
	for _, attr := range attrs {
		attrType := int(attr.Attr.Type) & nfnl.NLA_TYPE_MASK
		isNestedAttr := int(attr.Attr.Type)&syscall.NLA_F_NESTED == syscall.NLA_F_NESTED

		switch attrType {
		case nfnl.CTA_TUPLE_ORIG:
			if !isNestedAttr {
				return ctentry, errors.New("Nested attribute value expected")
			}
			parseConntrackTuple(&ctentry.OriginalTuple, attr.Value, family)
		case nfnl.CTA_TUPLE_REPLY:
			if !isNestedAttr {
				return ctentry, errors.New("Nested attribute value expected")
			}
			parseConntrackTuple(&ctentry.ReplyTuple, attr.Value, family)
		case nfnl.CTA_STATUS:
			ctentry.Status = int(native.Uint32(attr.Value[0:4]))
		case nfnl.CTA_TIMEOUT:
			ctentry.Timeout = int(native.Uint32(attr.Value[0:4]))
		case nfnl.CTA_MARK:
			ctentry.Mark = int(native.Uint32(attr.Value[0:4]))
		case nfnl.CTA_COUNTERS_ORIG:
			if !isNestedAttr {
				return ctentry, errors.New("Nested attribute value expected")
			}
			ctentry.OriginalCounters, _ = parseConntrackCounters(attr.Value)
		case nfnl.CTA_COUNTERS_REPLY:
			if !isNestedAttr {
				return ctentry, errors.New("Nested attribute value expected")
			}
			ctentry.ReplyCounters, _ = parseConntrackCounters(attr.Value)
		case nfnl.CTA_ID:
			ctentry.Id = int(native.Uint32(attr.Value[0:4]))
		case nfnl.CTA_ZONE:
			ctentry.Zone = int(native.Uint16(attr.Value[0:2]))
		case nfnl.CTA_USE:
			ctentry.Use = int(native.Uint32(attr.Value[0:4]))
		case nfnl.CTA_SECMARK:
			ctentry.Secmark = int(native.Uint32(attr.Value[0:4]))
		}
	}
	return ctentry, nil
}

func parseConntrackTuple(tuple *CtTuple, value []byte, family uint8) error {
	var attrs [nfnl.CTA_TUPLE_MAX]nfnl.NetlinkNetfilterAttr
	err := nfnl.ParseNetfilterAttr(value, attrs[:])
	if err != nil {
		return err
	}
	tuple.L3ProtoNum = int(family)

	native := binary.BigEndian
	for _, attr := range attrs {
		attrType := uint16(int(attr.Attr.Type) & nfnl.NLA_TYPE_MASK)
		isNestedAttr := int(attr.Attr.Type)&syscall.NLA_F_NESTED == syscall.NLA_F_NESTED

		switch attrType {
		case nfnl.CTA_TUPLE_IP:
			if !isNestedAttr {
				return errors.New("Nested attribute value expected")
			}
			parseTupleIp(tuple, attr.Value)
		case nfnl.CTA_TUPLE_PROTO:
			if !isNestedAttr {
				return errors.New("Nested attribute value expected")
			}
			parseTupleProto(tuple, attr.Value)
		case nfnl.CTA_ZONE:
			tuple.Zone = int(native.Uint16(attr.Value[0:2]))
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func parseTupleIp(tuple *CtTuple, value []byte) error {
	var attrs [nfnl.CTA_IP_MAX]nfnl.NetlinkNetfilterAttr
	err := nfnl.ParseNetfilterAttr(value, attrs[:])
	if err != nil {
		return err
	}
	for _, attr := range attrs {
		switch attr.Attr.Type {
		case nfnl.CTA_IP_V4_SRC, nfnl.CTA_IP_V6_SRC:
			copy(tuple.Src[:], net.IP(attr.Value).To16()[:16])
		case nfnl.CTA_IP_V4_DST, nfnl.CTA_IP_V6_DST:
			copy(tuple.Dst[:], net.IP(attr.Value).To16()[:16])
		}
	}
	return err
}

func parseTupleProto(tuple *CtTuple, value []byte) error {
	var attrs [nfnl.CTA_PROTO_MAX]nfnl.NetlinkNetfilterAttr
	err := nfnl.ParseNetfilterAttr(value, attrs[:])
	if err != nil {
		return err
	}
	native := binary.BigEndian
	for _, attr := range attrs {
		switch attr.Attr.Type {
		case nfnl.CTA_PROTO_NUM:
			tuple.ProtoNum = int(attr.Value[0])
		case nfnl.CTA_PROTO_SRC_PORT:
			tuple.L4Src.Port = int(native.Uint16(attr.Value))
		case nfnl.CTA_PROTO_DST_PORT:
			tuple.L4Dst.Port = int(native.Uint16(attr.Value))
		case nfnl.CTA_PROTO_ICMP_ID:
			tuple.L4Src.Id = int(native.Uint16(attr.Value))
		case nfnl.CTA_PROTO_ICMP_TYPE:
			tuple.L4Dst.Type = int(attr.Value[0])
		case nfnl.CTA_PROTO_ICMP_CODE:
			tuple.L4Dst.Code = int(attr.Value[0])
		}
	}
	return err
}

func parseConntrackCounters(value []byte) (CtCounters, error) {
	counters := CtCounters{}
	var attrs [nfnl.CTA_COUNTERS_MAX]nfnl.NetlinkNetfilterAttr
	err := nfnl.ParseNetfilterAttr(value, attrs[:])
	if err != nil {
		return counters, err
	}
	native := binary.BigEndian
	for _, attr := range attrs {
		switch attr.Attr.Type {
		case nfnl.CTA_COUNTERS_PACKETS:
			counters.Packets = int(native.Uint64(attr.Value[0:8]))
		case nfnl.CTA_COUNTERS_BYTES:
			counters.Bytes = int(native.Uint64(attr.Value[0:8]))
		case nfnl.CTA_COUNTERS32_PACKETS:
			counters.Packets = int(native.Uint32(attr.Value[0:8]))
		case nfnl.CTA_COUNTERS32_BYTES:
			counters.Bytes = int(native.Uint32(attr.Value[0:8]))
		}
	}
	return counters, err
}
