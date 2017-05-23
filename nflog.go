package nfnetlink

import (
	"net"
)

type NflogPacketHeader struct {
	HwProtocol int
	Hook       int
}

type NflogPacketTimestamp struct {
	Sec  uint64
	Usec uint64
}

type NflogL4Info struct {
	Port int
	Id   int
	Type int
	Code int
}

type NflogPacketTuple struct {
	Src   net.IP
	Dst   net.IP
	Proto int
	L4Src NflogL4Info
	L4Dst NflogL4Info
}

type NflogPacket struct {
	Header    *NflogPacketHeader
	Mark      int
	Timestamp *NflogPacketTimestamp
	Prefix    string
	Gid       int
	Tuple     *NflogPacketTuple
	Bytes     int
}
