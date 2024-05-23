// Copyright (c) 2023 Tigera, Inc. All rights reserved.
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

package ut_test

import (
	"fmt"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/bpf/dnsresolver"
	"github.com/projectcalico/calico/felix/bpf/ipsets"
	"github.com/projectcalico/calico/felix/bpf/maps"
	"github.com/projectcalico/calico/felix/ip"
)

func TestDNSParser(t *testing.T) {
	RegisterTestingT(t)

	// DNS response to archive.ubuntu.com with multiple A aswers
	pktBytes := []byte{
		26, 97, 165, 211, 168, 175, 246, 111, 42, 69, 108, 168, 8, 0, 69, 0, 0,
		234, 220, 104, 64, 0, 125, 17, 79, 45, 10, 100, 0, 10, 192, 168, 6, 87,
		0, 53, 225, 200, 0, 214, 38, 108, 205, 111, 129, 128, 0, 1, 0, 5, 0, 0,
		0, 0, 7, 97, 114, 99, 104, 105, 118, 101, 6, 117, 98, 117, 110, 116,
		117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 7, 97, 114, 99, 104, 105, 118, 101,
		6, 117, 98, 117, 110, 116, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 0, 0, 0,
		25, 0, 4, 91, 189, 91, 83, 7, 97, 114, 99, 104, 105, 118, 101, 6, 117,
		98, 117, 110, 116, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 0, 0, 0, 25, 0,
		4, 185, 125, 190, 36, 7, 97, 114, 99, 104, 105, 118, 101, 6, 117, 98,
		117, 110, 116, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 0, 0, 0, 25, 0, 4,
		91, 189, 91, 82, 7, 97, 114, 99, 104, 105, 118, 101, 6, 117, 98, 117,
		110, 116, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 0, 0, 0, 25, 0, 4, 185,
		125, 190, 39, 7, 97, 114, 99, 104, 105, 118, 101, 6, 117, 98, 117, 110,
		116, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 0, 0, 0, 25, 0, 4, 91, 189,
		91, 81,
	}

	dnsPfxMap := dnsresolver.DNSPrefixMap()
	dnsPfxMap.EnsureExists()
	defer dnsPfxMap.Close()

	dnsPfxMap.Update(dnsresolver.NewPfxKey("ubuntu.com").AsBytes(), dnsresolver.NewPfxValue(123).AsBytes())
	dnsPfxMap.Update(dnsresolver.NewPfxKey("*.ubuntu.com").AsBytes(), dnsresolver.NewPfxValue(1234).AsBytes())
	dnsPfxMap.Update(dnsresolver.NewPfxKey("archive.ubuntu.com").AsBytes(), dnsresolver.NewPfxValue(1, 2, 3).AsBytes())

	runBpfUnitTest(t, "dns_parser_test.c", func(bpfrun bpfProgRunFn) {
		res, err := bpfrun(pktBytes)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Retval).To(Equal(resTC_ACT_UNSPEC))

		pktR := gopacket.NewPacket(res.dataOut, layers.LayerTypeEthernet, gopacket.Default)
		fmt.Printf("pktR = %+v\n", pktR)

	})

	ipsMap.Iter(func(k, v []byte) maps.IteratorAction {
		fmt.Println(ipsets.IPSetEntryFromBytes(k))
		return maps.IterNone
	})

	for _, setID := range []uint64{1, 2, 3} {
		_, err := ipsMap.Get(
			ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("91.189.91.81/32").(ip.V4CIDR), 0, 0).AsBytes())
		Expect(err).NotTo(HaveOccurred())
		_, err = ipsMap.Get(
			ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("91.189.91.82/32").(ip.V4CIDR), 0, 0).AsBytes())
		Expect(err).NotTo(HaveOccurred())
		_, err = ipsMap.Get(
			ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("91.189.91.83/32").(ip.V4CIDR), 0, 0).AsBytes())
		Expect(err).NotTo(HaveOccurred())
		_, err = ipsMap.Get(
			ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("185.125.190.36/32").(ip.V4CIDR), 0, 0).AsBytes())
		Expect(err).NotTo(HaveOccurred())
		_, err = ipsMap.Get(
			ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("185.125.190.39/32").(ip.V4CIDR), 0, 0).AsBytes())
		Expect(err).NotTo(HaveOccurred())
	}

	// DNS response to zpravy.idnes.cz with CNAME and A answer
	pktBytes = []byte{
		26, 97, 165, 211, 168, 175, 246, 111, 42, 69, 108, 168, 8, 0, 69, 0, 0,
		130, 84, 86, 64, 0, 125, 17, 215, 167, 10, 100, 0, 10, 192, 168, 6, 87,
		0, 53, 207, 225, 0, 110, 61, 11, 191, 48, 129, 128, 0, 1, 0, 2, 0, 0, 0,
		0, 6, 122, 112, 114, 97, 118, 121, 5, 105, 100, 110, 101, 115, 2, 99,
		122, 0, 0, 1, 0, 1, 6, 122, 112, 114, 97, 118, 121, 5, 105, 100, 110,
		101, 115, 2, 99, 122, 0, 0, 5, 0, 1, 0, 0, 0, 30, 0, 14, 3, 99, 49, 52,
		5, 105, 100, 110, 101, 115, 2, 99, 122, 0, 3, 99, 49, 52, 5, 105, 100,
		110, 101, 115, 2, 99, 122, 0, 0, 1, 0, 1, 0, 0, 0, 30, 0, 4, 185, 17,
		117, 45}

	dnsPfxMap.Update(dnsresolver.NewPfxKey("*.idnes.cz").AsBytes(), dnsresolver.NewPfxValue(666, 3).AsBytes())

	runBpfUnitTest(t, "dns_parser_test.c", func(bpfrun bpfProgRunFn) {
		res, err := bpfrun(pktBytes)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Retval).To(Equal(resTC_ACT_UNSPEC))

		pktR := gopacket.NewPacket(res.dataOut, layers.LayerTypeEthernet, gopacket.Default)
		fmt.Printf("pktR = %+v\n", pktR)

	})

	ipsMap.Iter(func(k, v []byte) maps.IteratorAction {
		fmt.Println(ipsets.IPSetEntryFromBytes(k))
		return maps.IterNone
	})

	_, err := ipsMap.Get(
		ipsets.MakeBPFIPSetEntry(3, ip.CIDRFromStringNoErr("185.17.117.45/32").(ip.V4CIDR), 0, 0).AsBytes())
	Expect(err).NotTo(HaveOccurred())
	_, err = ipsMap.Get(
		ipsets.MakeBPFIPSetEntry(666, ip.CIDRFromStringNoErr("185.17.117.45/32").(ip.V4CIDR), 0, 0).AsBytes())
	Expect(err).NotTo(HaveOccurred())
}
