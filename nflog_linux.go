package nfnetlink

import (
	"encoding/binary"
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

	// Channel to pass raw netlink messages for further processing. We keep it at
	// twice the size of the processor's outgoing channel so that reading netlink
	// messages from the socket can be buffered until they can be consumed.
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
					pktAggr, seen := aggregate[nflogPacket.Tuple]
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
						aggregate[nflogPacket.Tuple] = pktAggr
					}
				}
			case <-sendTicker.C:
				for t, pktAddr := range aggregate {
					// Don't block when trying to send to slow receivers.
					// In case of slow receivers, simply continue aggregating and
					// retry sending next time around.
					select {
					case ch <- pktAddr:
						delete(aggregate, t)
					default:
					}
				}
			}
		}
	}()
	return nil
}

func parseNflog(m []byte) (NflogPacket, error) {
	nflogPacket := NflogPacket{}
	var attrs [nfnl.NFULA_MAX]nfnl.NetlinkNetfilterAttr
	err := nfnl.ParseNetfilterAttr(m, attrs[:])
	if err != nil {
		return nflogPacket, err
	}

	for _, attr := range attrs {

		native := binary.BigEndian
		switch attr.Attr.Type {
		case nfnl.NFULA_PACKET_HDR:
			nflogPacket.Header.HwProtocol = int(native.Uint16(attr.Value[0:2]))
			nflogPacket.Header.Hook = int(attr.Value[2])
		case nfnl.NFULA_MARK:
			nflogPacket.Mark = int(native.Uint32(attr.Value[0:4]))
		case nfnl.NFULA_PAYLOAD:
			parsePacketHeader(&nflogPacket.Tuple, nflogPacket.Header.HwProtocol, attr.Value)
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
	return nflogPacket, nil
}

func parsePacketHeader(tuple *NflogPacketTuple, hwProtocol int, nflogPayload []byte) error {
	switch hwProtocol {
	case IPv4Proto:
		ipHeader := pkt.ParseIPv4Header(nflogPayload)
		copy(tuple.Src[:], ipHeader.Saddr.To16()[:16])
		copy(tuple.Dst[:], ipHeader.Daddr.To16()[:16])
		tuple.Proto = int(ipHeader.Protocol)
		parseLayer4Header(tuple, nflogPayload[ipHeader.IHL:])
	case IPv6Proto:
		ipHeader := pkt.ParseIPv6Header(nflogPayload)
		copy(tuple.Src[:], ipHeader.Saddr.To16()[:16])
		copy(tuple.Dst[:], ipHeader.Daddr.To16()[:16])
		tuple.Proto = int(ipHeader.NextHeader)
		parseLayer4Header(tuple, nflogPayload[pkt.IPv6HeaderLen:])
	}
	return nil
}

func parseLayer4Header(tuple *NflogPacketTuple, l4payload []byte) error {
	switch tuple.Proto {
	case ProtoIcmp:
		header := pkt.ParseICMPHeader(l4payload)
		tuple.L4Src.Id = int(header.Id)
		tuple.L4Dst.Type = int(header.Type)
		tuple.L4Dst.Code = int(header.Code)
	case ProtoTcp:
		header := pkt.ParseTCPHeader(l4payload)
		tuple.L4Src.Port = int(header.Source)
		tuple.L4Dst.Port = int(header.Dest)
	case ProtoUdp:
		header := pkt.ParseUDPHeader(l4payload)
		tuple.L4Src.Port = int(header.Source)
		tuple.L4Dst.Port = int(header.Dest)
	}
	return nil
}
