package nfnetlink

import (
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/tigera/nfnetlink/nfnl"
	"github.com/tigera/nfnetlink/pkt"
	"github.com/vishvananda/netlink/nl"
)

const (
	IPv4Proto = 0x800
	IPv6Proto = 0x86DD
)

const (
	ProtoIcmp = 1
	ProtoTcp  = 6
	ProtoUdp  = 17
)

func NflogSubscribe(ch chan<- NflogPacket, done <-chan struct{}) error {
	sock, err := nl.Subscribe(syscall.NETLINK_NETFILTER)
	if err != nil {
		return err
	}
	// TODO(doublek): Move all this someplace nice.
	nlMsgType := nfnl.NFNL_SUBSYS_ULOG<<8 | nfnl.NFULNL_MSG_CONFIG
	nlMsgFlags := syscall.NLM_F_REQUEST

	req := nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	nfgenmsg := nfnl.NewNfGenMsg(syscall.AF_INET, nfnl.NFNETLINK_V0, 0)
	req.AddData(nfgenmsg)
	nflogcmd := nfnl.NewNflogMsgConfigCmd(nfnl.NFULNL_CFG_CMD_PF_UNBIND)
	nfattr := nl.NewRtAttr(nfnl.NFULA_CFG_CMD, nflogcmd.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	req = nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	nfgenmsg = nfnl.NewNfGenMsg(syscall.AF_INET, nfnl.NFNETLINK_V0, 0)
	req.AddData(nfgenmsg)
	nflogcmd = nfnl.NewNflogMsgConfigCmd(nfnl.NFULNL_CFG_CMD_PF_BIND)
	nfattr = nl.NewRtAttr(nfnl.NFULA_CFG_CMD, nflogcmd.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	req = nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	// TODO(doublek): htons(20) -> 5120. Hardcoded, fix it please.
	nfgenmsg = nfnl.NewNfGenMsg(syscall.AF_INET, nfnl.NFNETLINK_V0, 5120)
	req.AddData(nfgenmsg)
	nflogcmd = nfnl.NewNflogMsgConfigCmd(nfnl.NFULNL_CFG_CMD_BIND)
	nfattr = nl.NewRtAttr(nfnl.NFULA_CFG_CMD, nflogcmd.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	req = nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	// TODO(doublek): htons(20) -> 5120. Hardcoded, fix it please.
	nfgenmsg = nfnl.NewNfGenMsg(syscall.AF_UNSPEC, nfnl.NFNETLINK_V0, 5120)
	req.AddData(nfgenmsg)
	nflogcfg := nfnl.NewNflogMsgConfigMode(0xFF, nfnl.NFULNL_COPY_PACKET)
	nfattr = nl.NewRtAttr(nfnl.NFULA_CFG_MODE, nflogcfg.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	go func() {
		<-done
		sock.Close()
	}()
	go func() {
		defer close(ch)
	Recvloop:
		for {
			var res [][]byte
			fmt.Println("NflogSubscribe: Receiving:")
			msgs, err := sock.Receive()
			if err != nil {
				fmt.Println("NflogSubscribe: ERROR: ", err)
				return
			}
			for _, m := range msgs {
				mType := m.Header.Type
				mFlags := m.Header.Flags
				if mType == syscall.NLMSG_DONE {
					fmt.Println("NLMSG_DONE")
					break
				}
				if mType == syscall.NLMSG_ERROR {
					native := binary.LittleEndian
					err := int32(native.Uint32(m.Data[0:4]))
					fmt.Println("NLMSG_ERROR: ", syscall.Errno(-err))
					continue Recvloop
				}
				res = append(res, m.Data)
				if mFlags&syscall.NLM_F_MULTI == 0 {
					fmt.Println("End of multi part message", mType)
					break
				}
			}
			for _, m := range res {
				msg := nfnl.DeserializeNfGenMsg(m)
				nflogPacket, err := parseNflog(m[msg.Len():])
				if err != nil {
					fmt.Println("Error parsing NFLOG", err)
					continue
				}
				ch <- nflogPacket
			}
		}
	}()

	return nil
}

func parseNflog(m []byte) (NflogPacket, error) {
	nflogPacket := NflogPacket{}
	attrs, err := nfnl.ParseNetfilterAttr(m)
	if err != nil {
		return nflogPacket, err
	}

	for _, attr := range attrs {

		native := binary.BigEndian
		switch attr.Attr.Type {
		case nfnl.NFULA_PACKET_HDR:
			header := NflogPacketHeader{}
			header.HwProtocol = int(native.Uint16(attr.Value[0:2]))
			header.Hook = int(attr.Value[2])
			nflogPacket.Header = header
		case nfnl.NFULA_MARK:
			nflogPacket.Mark = int(native.Uint32(attr.Value[0:4]))
		case nfnl.NFULA_PAYLOAD:
			nflogPacket.Tuple, _ = parsePacketHeader(nflogPacket.Header.HwProtocol, attr.Value)
		case nfnl.NFULA_PREFIX:
			nflogPacket.Prefix = string(attr.Value)
		case nfnl.NFULA_GID:
			nflogPacket.GroupId = int(native.Uint32(attr.Value[0:4]))
		}
	}
	return nflogPacket, nil
}

func parsePacketHeader(hwProtocol int, nflogPayload []byte) (NflogPacketTuple, error) {
	tuple := NflogPacketTuple{}
	switch hwProtocol {
	case IPv4Proto:
		ipHeader := pkt.ParseIPv4Header(nflogPayload)
		tuple.Src = ipHeader.Saddr
		tuple.Dst = ipHeader.Daddr
		tuple.Proto = int(ipHeader.Protocol)
		srcL4, dstL4, _ := parseLayer4Header(int(ipHeader.Protocol), nflogPayload[ipHeader.IHL:])
		tuple.L4Src = srcL4
		tuple.L4Dst = dstL4
	case IPv6Proto:
		fmt.Println("IPv6 Packet")
	}
	return tuple, nil
}

func parseLayer4Header(IPProto int, l4payload []byte) (NflogL4Info, NflogL4Info, error) {
	srcL4Info := NflogL4Info{}
	dstL4Info := NflogL4Info{}
	switch IPProto {
	case ProtoIcmp:
		header := pkt.ParseICMPHeader(l4payload)
		srcL4Info.Id = int(header.Id)
		dstL4Info.Type = int(header.Type)
		dstL4Info.Code = int(header.Code)
	case ProtoTcp:
		header := pkt.ParseTCPHeader(l4payload)
		srcL4Info.Port = int(header.Source)
		dstL4Info.Port = int(header.Dest)
	case ProtoUdp:
		header := pkt.ParseUDPHeader(l4payload)
		srcL4Info.Port = int(header.Source)
		dstL4Info.Port = int(header.Dest)
	}
	return srcL4Info, dstL4Info, nil
}
