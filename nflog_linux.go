package nfnetlink

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
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

const AggregationDuration = time.Duration(10) * time.Millisecond

func NflogSubscribe(groupNum int, bufSize int, ch chan<- *NflogPacketAggregate, done <-chan struct{}) error {
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
	nfgenmsg = nfnl.NewNfGenMsg(syscall.AF_INET, nfnl.NFNETLINK_V0, groupNum)
	req.AddData(nfgenmsg)
	nflogcmd = nfnl.NewNflogMsgConfigCmd(nfnl.NFULNL_CFG_CMD_BIND)
	nfattr = nl.NewRtAttr(nfnl.NFULA_CFG_CMD, nflogcmd.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	req = nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	nfgenmsg = nfnl.NewNfGenMsg(syscall.AF_UNSPEC, nfnl.NFNETLINK_V0, groupNum)
	req.AddData(nfgenmsg)
	nflogcfg := nfnl.NewNflogMsgConfigMode(0xFF, nfnl.NFULNL_COPY_PACKET)
	nfattr = nl.NewRtAttr(nfnl.NFULA_CFG_MODE, nflogcfg.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	req = nl.NewNetlinkRequest(nlMsgType, nlMsgFlags)
	nfgenmsg = nfnl.NewNfGenMsg(syscall.AF_UNSPEC, nfnl.NFNETLINK_V0, groupNum)
	req.AddData(nfgenmsg)
	nflogbufsiz := nfnl.NewNflogMsgConfigBufSiz(bufSize)
	nfattr = nl.NewRtAttr(nfnl.NFULA_CFG_NLBUFSIZ, nflogbufsiz.Serialize())
	req.AddData(nfattr)
	if err := sock.Send(req); err != nil {
		return err
	}

	go func() {
		<-done
		sock.Close()
	}()

	// Channel to pass raw Nflog packets for further processing. We keep it at
	// twice the size of the processers outgoing channel so that reading NFLOG
	// packets from the socket can be buffered until they can be consumed.
	resChan := make(chan [][]byte, 2*cap(ch))
	// Start a goroutine for receiving netlink messages from the kernel.
	go func() {
		logCtx := log.WithFields(log.Fields{
			"groupNum": groupNum,
		})
	Recvloop:
		for {
			var res [][]byte
			msgs, err := sock.Receive()
			if err != nil {
				switch err := err.(type) {
				case syscall.Errno:
					if err.Temporary() || err == syscall.ENOBUFS {
						logCtx.Warnf("NflogSubscribe Receive: %v", err)
						continue
					}
				default:
					logCtx.Fatalf("NflogSubscribe Receive: %v", err)
				}
			}
			for _, m := range msgs {
				mType := m.Header.Type
				if mType == syscall.NLMSG_DONE {
					break
				}
				if mType == syscall.NLMSG_ERROR {
					native := binary.LittleEndian
					err := int32(native.Uint32(m.Data[0:4]))
					logCtx.Warnf("NLMSG_ERROR: %v", syscall.Errno(-err))
					continue Recvloop
				}
				res = append(res, m.Data)
			}
			resChan <- res
		}
	}()
	// Start another goroutine for parsing netlink messages into nflog objects
	go func() {
		defer close(ch)
		logCtx := log.WithFields(log.Fields{
			"groupNum": groupNum,
		})
		// We batch NFLOG objects and send them to the subscriber every
		// "AggregationDuration" time interval.
		sendTicker := time.NewTicker(AggregationDuration)
		// Batching is done like so:
		// For each NflogPacketTuple if it's a prefix we've already seen we update
		// packet and byte counters on exising NflogPrefix and discard the parsed
		// packet.
		aggregate := make(map[NflogPacketTuple]*NflogPacketAggregate)
		for {
			select {
			case res := <-resChan:
				for _, m := range res {
					msg := nfnl.DeserializeNfGenMsg(m)
					nflogPacket, err := parseNflog(m[msg.Len():])
					if err != nil {
						logCtx.Warnf("Error parsing NFLOG %v", err)
						continue
					}
					var pktAggr *NflogPacketAggregate
					updatePrefix := true
					pktAggr, seen := aggregate[*nflogPacket.Tuple]
					if seen {
						for i, prefix := range pktAggr.Prefixes {
							if prefix.Equals(&nflogPacket.Prefix) {
								prefix.Packets++
								prefix.Bytes += nflogPacket.Bytes
								pktAggr.Prefixes[i] = prefix
								updatePrefix = false
								break
							}
						}
						// We reached here, so we didn't find a prefix. Appending this prefix
						// is handled below.
					} else {
						pktAggr = &NflogPacketAggregate{
							Tuple: nflogPacket.Tuple,
						}
					}
					if updatePrefix {
						pktAggr.Prefixes = append(pktAggr.Prefixes, nflogPacket.Prefix)
						aggregate[*nflogPacket.Tuple] = pktAggr
					}
				}
			case <-sendTicker.C:
				for t, pktAddr := range aggregate {
					ch <- pktAddr
					delete(aggregate, t)
				}
			}
		}
	}()
	return nil
}

func parseNflog(m []byte) (*NflogPacket, error) {
	nflogPacket := &NflogPacket{}
	attrs, err := nfnl.ParseNetfilterAttr(m)
	if err != nil {
		// TODO(doublek): Exported AttrPool is a bit ugly, need to keep this local to
		// parseNflog.
		nfnl.AttrPool.Put(attrs)
		return nflogPacket, err
	}

	for _, attr := range attrs {

		native := binary.BigEndian
		switch attr.Attr.Type {
		case nfnl.NFULA_PACKET_HDR:
			header := &NflogPacketHeader{}
			header.HwProtocol = int(native.Uint16(attr.Value[0:2]))
			header.Hook = int(attr.Value[2])
			nflogPacket.Header = header
		case nfnl.NFULA_MARK:
			nflogPacket.Mark = int(native.Uint32(attr.Value[0:4]))
		case nfnl.NFULA_PAYLOAD:
			nflogPacket.Tuple, _ = parsePacketHeader(nflogPacket.Header.HwProtocol, attr.Value)
			nflogPacket.Bytes = len(attr.Value)
		case nfnl.NFULA_PREFIX:
			p := NflogPrefix{Len: len(attr.Value) - 1}
			copy(p.Prefix[:], attr.Value[:len(attr.Value)-1])
			nflogPacket.Prefix = p
		case nfnl.NFULA_GID:
			nflogPacket.Gid = int(native.Uint32(attr.Value[0:4]))
		}
	}
	nflogPacket.Prefix.Packets = 1
	nflogPacket.Prefix.Bytes = nflogPacket.Bytes
	nfnl.AttrPool.Put(attrs)
	return nflogPacket, nil
}

func parsePacketHeader(hwProtocol int, nflogPayload []byte) (*NflogPacketTuple, error) {
	tuple := &NflogPacketTuple{}
	switch hwProtocol {
	case IPv4Proto:
		ipHeader := pkt.ParseIPv4Header(nflogPayload)
		copy(tuple.Src[:], ipHeader.Saddr.To16()[:16])
		copy(tuple.Dst[:], ipHeader.Daddr.To16()[:16])
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
