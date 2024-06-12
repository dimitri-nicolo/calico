//go:build !windows
// +build !windows

// Copyright (c) 2024 Tigera, Inc. All rights reserved.
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

package dnsresolver

import (
	"encoding/binary"
	"fmt"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/projectcalico/calico/felix/bpf/maps"
)

const (
	DNSPfxKeySize   = 256 + 4
	DNSLpmIpsetsMax = 8
	DNSPfxValueSize = 8 + DNSLpmIpsetsMax*8
)

var DNSPfxMapParams = maps.MapParameters{
	Type:       "lpm_trie",
	KeySize:    DNSPfxKeySize,
	ValueSize:  DNSPfxValueSize,
	MaxEntries: 64 * 1024,
	Name:       "cali_dns_pfx",
	Flags:      unix.BPF_F_NO_PREALLOC,
	Version:    2,
}

func DNSPrefixMap() maps.Map {
	return maps.NewPinnedMap(DNSPfxMapParams)
}

type DNSPfxKey [DNSPfxKeySize]byte

func DNSPfxKeyFromBytes(b []byte) DNSPfxKey {
	var k DNSPfxKey
	copy(k[:], b)
	return k
}

func (k DNSPfxKey) AsBytes() []byte {
	return k[:]
}

func (k DNSPfxKey) PrefixLen() uint32 {
	return binary.LittleEndian.Uint32(k[:4])
}

func (k DNSPfxKey) Domain() string {
	l := int(binary.LittleEndian.Uint32(k[0:4]))
	l /= 8
	r := make([]byte, l)
	for i, b := range k[4 : 4+l] {
		r[l-i-1] = b
	}

	return string(r)
}

func (k DNSPfxKey) String() string {
	return fmt.Sprintf("prefix %d key \"%s\"", k.PrefixLen(), k.Domain())
}

func NewPfxKey(domain string) DNSPfxKey {
	var k DNSPfxKey

	prefixlen := 0

	if domain != "" {
		if domain[0] == '*' {
			domain = domain[1:]
		}
		prefixlen = len(domain) * 8
	}

	binary.LittleEndian.PutUint32(k[:4], uint32(prefixlen))

	bytes := []byte(domain)

	for i, b := range bytes {
		k[4+len(bytes)-i-1] = b
	}

	return k
}

type DNSPfxValue [DNSPfxValueSize]byte

func DNSPfxValueFromBytes(b []byte) DNSPfxValue {
	var k DNSPfxValue
	copy(k[:], b)
	return k
}

func (v DNSPfxValue) AsBytes() []byte {
	return v[:]
}

func (v DNSPfxValue) IDs() []uint64 {
	count := binary.LittleEndian.Uint32(v[:4])

	ret := make([]uint64, 0, count)

	for i := 0; i < int(count); i++ {
		ret = append(ret, binary.BigEndian.Uint64(v[(i+1)*8:(i+2)*8]))
	}

	return ret
}

func NewPfxValue(ipsets ...uint64) DNSPfxValue {
	var v DNSPfxValue

	count := uint32(len(ipsets))
	if count == 0 || count > DNSLpmIpsetsMax {
		log.Fatalf("Too few or too many ipsets %d", count)
	}

	binary.LittleEndian.PutUint32(v[:4], count)

	for i, s := range ipsets {
		binary.BigEndian.PutUint64(v[(i+1)*8:(i+2)*8], s)
	}

	return v
}
