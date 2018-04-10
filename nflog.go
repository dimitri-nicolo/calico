package nfnetlink

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
	Src   [16]byte
	Dst   [16]byte
	Proto int
	L4Src NflogL4Info
	L4Dst NflogL4Info
}

// NflogPrefix stores the "nflog-prefix" of a NFLOG packet.
// NFLOG prefixes are 64 characters long. We keep them as a byte array to save
// the allocation that comes with converting it to a string.
type NflogPrefix struct {
	Prefix  [64]byte
	Len     int
	Packets int
	Bytes   int
}

func (np *NflogPrefix) Equals(cmp *NflogPrefix) bool {
	return np.Prefix == cmp.Prefix
}

type NflogPacket struct {
	Header    NflogPacketHeader
	Mark      int
	Timestamp NflogPacketTimestamp
	Prefix    NflogPrefix
	Gid       int
	Tuple     NflogPacketTuple
	Bytes     int
}

type NflogPacketAggregate struct {
	Tuple    NflogPacketTuple
	Prefixes []NflogPrefix
}
