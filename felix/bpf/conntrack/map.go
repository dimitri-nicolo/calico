// +build !windows

// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conntrack

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/bpf"
)

// struct calico_ct_key {
//   uint32_t protocol;
//   __be32 addr_a, addr_b; // NBO
//   uint16_t port_a, port_b; // HBO
// };
const KeySize = 16
const ValueSize = 88
const MaxEntries = 512000

type Key [KeySize]byte

func (k Key) AsBytes() []byte {
	return k[:]
}

func (k Key) Proto() uint8 {
	return uint8(binary.LittleEndian.Uint32(k[:4]))
}

func (k Key) AddrA() net.IP {
	return k[4:8]
}

func (k Key) PortA() uint16 {
	return binary.LittleEndian.Uint16(k[12:14])
}

func (k Key) AddrB() net.IP {
	return k[8:12]
}

func (k Key) PortB() uint16 {
	return binary.LittleEndian.Uint16(k[14:16])
}

func (k Key) String() string {
	return fmt.Sprintf("ConntrackKey{proto=%v %v:%v <-> %v:%v}",
		k.Proto(), k.AddrA(), k.PortA(), k.AddrB(), k.PortB())
}

func NewKey(proto uint8, ipA net.IP, portA uint16, ipB net.IP, portB uint16) Key {
	var k Key
	binary.LittleEndian.PutUint32(k[:4], uint32(proto))
	copy(k[4:8], ipA.To4())
	copy(k[8:12], ipB.To4())
	binary.LittleEndian.PutUint16(k[12:14], portA)
	binary.LittleEndian.PutUint16(k[14:16], portB)
	return k
}

// struct calico_ct_value {
//  __u64 created;
//  __u64 last_seen; // 8
//  __u8 type;     // 16
//  __u8 flags;     // 17
//
//  // Important to use explicit padding, otherwise the compiler can decide
//  // not to zero the padding bytes, which upsets the verifier.  Worse than
//  // that, debug logging often prevents such optimisation resulting in
//  // failures when debug logging is compiled out only :-).
//  __u8 pad0[6];
//  union {
//    // CALI_CT_TYPE_NORMAL and CALI_CT_TYPE_NAT_REV.
//    struct {
//      struct calico_ct_leg a_to_b; // 24
//      struct calico_ct_leg b_to_a; // 56
//
//      // CALI_CT_TYPE_NAT_REV only.
//      __u32 orig_dst;                    // 88
//      __u16 orig_port;                   // 92
//      __u8 pad1[2];                      // 94
//      __u32 tun_ip;                      // 96
//      __u32 pad3;                        // 100
//    };
//
//    // CALI_CT_TYPE_NAT_FWD; key for the CALI_CT_TYPE_NAT_REV entry.
//    struct {
//      struct calico_ct_key nat_rev_key;  // 24
//      __u8 pad2[64];
//    };
//  };
// };

const (
	voCreated  int = 0
	voLastSeen     = 8
	voType         = 16
	voFlags        = 17
	voRevKey       = 24
	voLegAB        = 24
	voLegBA        = 48
	voTunIP        = 72
	voOrigIP       = 76
	voOrigPort     = 80
)

type Value [ValueSize]byte

func (e Value) Created() int64 {
	return int64(binary.LittleEndian.Uint64(e[voCreated : voCreated+8]))
}

func (e Value) LastSeen() int64 {
	return int64(binary.LittleEndian.Uint64(e[voLastSeen : voLastSeen+8]))
}

func (e Value) Type() uint8 {
	return e[voType]
}

func (e Value) Flags() uint8 {
	return e[voFlags]
}

// OrigIP returns the original destination IP, valid only if Type() is TypeNormal or TypeNATReverse
func (e Value) OrigIP() net.IP {
	return e[voOrigIP : voOrigIP+4]
}

// OrigPort returns the original destination port, valid only if Type() is TypeNormal or TypeNATReverse
func (e Value) OrigPort() uint16 {
	return binary.LittleEndian.Uint16(e[voOrigPort : voOrigPort+2])
}

// OrigSPort returns the original source port, valid only if Type() is
// TypeNATReverse and if the value returned is non-zero.
func (e Value) OrigSPort() uint16 {
	return binary.LittleEndian.Uint16(e[voOrigPort+2 : voOrigPort+4])
}

// NATSPort resturns the port to SNAT to, valid only if Type() is TypeNATForward.
func (e Value) NATSPort() uint16 {
	return binary.LittleEndian.Uint16(e[40:42])
}

const (
	TypeNormal uint8 = iota
	TypeNATForward
	TypeNATReverse

	FlagNATOut        uint8 = (1 << 0)
	FlagNATFwdDsr     uint8 = (1 << 1)
	FlagNATNPFwd      uint8 = (1 << 2)
	FlagSkipFIB       uint8 = (1 << 3)
	FlagTrustDNS      uint8 = (1 << 4)
	FlagTrustWorkload uint8 = (1 << 5)
	FlagExtLocal      uint8 = (1 << 6)
)

func (e Value) ReverseNATKey() Key {
	var ret Key

	l := len(Key{})
	copy(ret[:l], e[voRevKey:voRevKey+l])

	return ret
}

// AsBytes returns the value as slice of bytes
func (e Value) AsBytes() []byte {
	return e[:]
}

func initValue(v *Value, created, lastSeen time.Duration, typ, flags uint8) {
	binary.LittleEndian.PutUint64(v[voCreated:voCreated+8], uint64(created))
	binary.LittleEndian.PutUint64(v[voLastSeen:voLastSeen+8], uint64(lastSeen))
	v[voType] = typ
	v[voFlags] = flags
}

// NewValueNormal creates a new Value of type TypeNormal based on the given parameters
func NewValueNormal(created, lastSeen time.Duration, flags uint8, legA, legB Leg) Value {
	v := Value{}

	initValue(&v, created, lastSeen, TypeNormal, flags)

	copy(v[voLegAB:voLegAB+legSize], legA.AsBytes())
	copy(v[voLegBA:voLegBA+legSize], legB.AsBytes())

	return v
}

// NewValueNATForward creates a new Value of type TypeNATForward for the given
// arguments and the reverse key
func NewValueNATForward(created, lastSeen time.Duration, flags uint8, revKey Key) Value {
	v := Value{}

	initValue(&v, created, lastSeen, TypeNATForward, flags)

	copy(v[voRevKey:voRevKey+KeySize], revKey.AsBytes())

	return v
}

// NewValueNATReverse creates a new Value of type TypeNATReverse for the given
// arguments and reverse parameters
func NewValueNATReverse(created, lastSeen time.Duration, flags uint8, legA, legB Leg,
	tunnelIP, origIP net.IP, origPort uint16) Value {
	v := Value{}

	initValue(&v, created, lastSeen, TypeNATReverse, flags)

	copy(v[voLegAB:voLegAB+legSize], legA.AsBytes())
	copy(v[voLegBA:voLegBA+legSize], legB.AsBytes())

	copy(v[voOrigIP:voOrigIP+4], origIP.To4())
	binary.LittleEndian.PutUint16(v[voOrigPort:voOrigPort+2], origPort)

	copy(v[voTunIP:voTunIP+4], tunnelIP.To4())

	return v
}

type Leg struct {
	Bytes       uint64
	Packets     uint32
	Seqno       uint32
	SynSeen     bool
	AckSeen     bool
	FinSeen     bool
	RstSeen     bool
	Whitelisted bool
	Opener      bool
	Ifindex     uint32
}

const legSize int = 24

func setBit(bits *uint32, bit uint8, val bool) {
	if val {
		*bits |= (1 << bit)
	}
}

// AsBytes returns Leg serialized as a slice of bytes
func (leg Leg) AsBytes() []byte {
	bytes := make([]byte, 24)

	bits := uint32(0)

	setBit(&bits, 0, leg.SynSeen)
	setBit(&bits, 1, leg.AckSeen)
	setBit(&bits, 2, leg.FinSeen)
	setBit(&bits, 3, leg.RstSeen)
	setBit(&bits, 4, leg.Whitelisted)
	setBit(&bits, 5, leg.Opener)

	binary.LittleEndian.PutUint64(bytes[0:8], leg.Bytes)
	binary.LittleEndian.PutUint32(bytes[8:12], leg.Packets)
	binary.LittleEndian.PutUint32(bytes[12:16], leg.Seqno)
	binary.LittleEndian.PutUint32(bytes[16:20], bits)
	binary.LittleEndian.PutUint32(bytes[20:24], leg.Ifindex)

	return bytes
}

func (leg Leg) Flags() uint32 {
	var flags uint32
	if leg.SynSeen {
		flags |= 1
	}
	if leg.AckSeen {
		flags |= 1 << 1
	}
	if leg.FinSeen {
		flags |= 1 << 2
	}
	if leg.RstSeen {
		flags |= 1 << 3
	}
	if leg.Whitelisted {
		flags |= 1 << 4
	}
	if leg.Opener {
		flags |= 1 << 5
	}
	return flags
}

func bitSet(bits uint32, bit uint8) bool {
	return (bits & (1 << bit)) != 0
}

func readConntrackLeg(b []byte) Leg {
	bits := binary.LittleEndian.Uint32(b[16:20])
	return Leg{
		Bytes:       binary.LittleEndian.Uint64(b[0:8]),
		Packets:     binary.LittleEndian.Uint32(b[8:12]),
		Seqno:       binary.BigEndian.Uint32(b[12:16]),
		SynSeen:     bitSet(bits, 0),
		AckSeen:     bitSet(bits, 1),
		FinSeen:     bitSet(bits, 2),
		RstSeen:     bitSet(bits, 3),
		Whitelisted: bitSet(bits, 4),
		Opener:      bitSet(bits, 5),
		Ifindex:     binary.LittleEndian.Uint32(b[20:24]),
	}
}

type EntryData struct {
	A2B       Leg
	B2A       Leg
	OrigDst   net.IP
	OrigPort  uint16
	OrigSPort uint16
	TunIP     net.IP
}

func (data EntryData) Established() bool {
	return data.A2B.SynSeen && data.A2B.AckSeen && data.B2A.SynSeen && data.B2A.AckSeen
}

func (data EntryData) RSTSeen() bool {
	return data.A2B.RstSeen || data.B2A.RstSeen
}

func (data EntryData) FINsSeen() bool {
	return data.A2B.FinSeen && data.B2A.FinSeen
}

func (data EntryData) FINsSeenDSR() bool {
	return data.A2B.FinSeen || data.B2A.FinSeen
}

func (e Value) Data() EntryData {
	ip := e[voOrigIP : voOrigIP+4]
	tip := e[voTunIP : voTunIP+4]
	return EntryData{
		A2B:       readConntrackLeg(e[voLegAB : voLegAB+legSize]),
		B2A:       readConntrackLeg(e[voLegBA : voLegBA+legSize]),
		OrigDst:   ip,
		OrigPort:  binary.LittleEndian.Uint16(e[voOrigPort : voOrigPort+2]),
		OrigSPort: binary.LittleEndian.Uint16(e[voOrigPort+2 : voOrigPort+4]),
		TunIP:     tip,
	}
}

func (e Value) String() string {
	flags := e.Flags()
	flagsStr := fmt.Sprintf("%v", flags)

	if flags == 0 {
		flagsStr = " <none>"
	} else {
		if flags&FlagNATOut != 0 {
			flagsStr += " nat-out"
		}

		if flags&FlagNATFwdDsr != 0 {
			flagsStr += " fwd-dsr"
		}

		if flags&FlagNATNPFwd != 0 {
			flagsStr += " np-fwd"
		}

		if flags&FlagSkipFIB != 0 {
			flagsStr += " skip-fib"
		}

		if flags&FlagExtLocal != 0 {
			flagsStr += " ext-local"
		}
	}

	ret := fmt.Sprintf("Entry{Type:%d, Created:%d, LastSeen:%d, Flags:%s ",
		e.Type(), e.Created(), e.LastSeen(), flagsStr)

	switch e.Type() {
	case TypeNATForward:
		ret += fmt.Sprintf("REVKey: %s NATSPort: %d", e.ReverseNATKey().String(), e.NATSPort())
	case TypeNormal, TypeNATReverse:
		ret += fmt.Sprintf("Data: %+v", e.Data())
	default:
		ret += "TYPE INVALID"
	}

	return ret + "}"
}

func (e Value) IsForwardDSR() bool {
	return e.Flags()&FlagNATFwdDsr != 0
}

var MapParams = bpf.MapParameters{
	Filename:   "/sys/fs/bpf/tc/globals/cali_v4_ct",
	Type:       "hash",
	KeySize:    KeySize,
	ValueSize:  ValueSize,
	MaxEntries: MaxEntries,
	Name:       "cali_v4_ct",
	Flags:      unix.BPF_F_NO_PREALLOC,
	Version:    3,
}

func Map(mc *bpf.MapContext) bpf.Map {
	return mc.NewPinnedMap(MapParams)
}

const (
	ProtoICMP = 1
	ProtoTCP  = 6
	ProtoUDP  = 17
)

func KeyFromBytes(k []byte) Key {
	var ctKey Key
	if len(k) != len(ctKey) {
		log.Panic("Key has unexpected length")
	}
	copy(ctKey[:], k[:])
	return ctKey
}

func ValueFromBytes(v []byte) Value {
	var ctVal Value
	if len(v) != len(ctVal) {
		log.Panic("Value has unexpected length")
	}
	copy(ctVal[:], v[:])
	return ctVal
}

type MapMem map[Key]Value

// LoadMapMem loads ConntrackMap into memory
func LoadMapMem(m bpf.Map) (MapMem, error) {
	ret := make(MapMem)

	err := m.Iter(func(k, v []byte) bpf.IteratorAction {
		ks := len(Key{})
		vs := len(Value{})

		var key Key
		copy(key[:ks], k[:ks])

		var val Value
		copy(val[:vs], v[:vs])

		ret[key] = val
		return bpf.IterNone
	})

	return ret, err
}

// MapMemIter returns bpf.MapIter that loads the provided MapMem
func MapMemIter(m MapMem) bpf.IterCallback {
	ks := len(Key{})
	vs := len(Value{})

	return func(k, v []byte) bpf.IteratorAction {
		var key Key
		copy(key[:ks], k[:ks])

		var val Value
		copy(val[:vs], v[:vs])

		m[key] = val
		return bpf.IterNone
	}
}

// BytesToKey turns a slice of bytes into a Key
func BytesToKey(bytes []byte) Key {
	var k Key

	copy(k[:], bytes[:])

	return k
}

// StringToKey turns a string into a Key
func StringToKey(str string) Key {
	return BytesToKey([]byte(str))
}

// BytesToValue turns a slice of bytes into a value
func BytesToValue(bytes []byte) Value {
	var v Value

	copy(v[:], bytes)

	return v
}

// StringToValue turns a string into a Value
func StringToValue(str string) Value {
	return BytesToValue([]byte(str))
}
