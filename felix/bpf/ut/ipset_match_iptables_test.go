// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package ut_test

import (
	"net"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/bpf/bpfdefs"
	"github.com/projectcalico/calico/felix/bpf/ipsets"
	"github.com/projectcalico/calico/felix/bpf/iptables"
	"github.com/projectcalico/calico/felix/ip"
)

func TestMatchBPFIpsetsProgramForIPTablesV6(t *testing.T) {
	RegisterTestingT(t)
	err := iptables.CreateDNSObjPinDir("debug")
	Expect(err).NotTo(HaveOccurred())

	defer iptables.Cleanup()
	setID := uint64(1234)

	err = iptables.LoadIPSetsPolicyProgram(setID, "debug", 6)
	Expect(err).NotTo(HaveOccurred())
	pinPath := bpfdefs.IPSetMatchProg(setID, 6, "debug")
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())

	defer os.RemoveAll(pinPath)
	err = ipsMapV6.Update(ipsets.MakeBPFIPSetEntryV6(setID, ip.CIDRFromStringNoErr("9100::51/128").(ip.V6CIDR), 0, 0).AsBytes(), ipsets.DummyValue)
	Expect(err).NotTo(HaveOccurred())
	err = ipsMapV6.Update(ipsets.MakeBPFIPSetEntryV6(setID, ip.CIDRFromStringNoErr("9100::52/128").(ip.V6CIDR), 0, 0).AsBytes(), ipsets.DummyValue)
	Expect(err).NotTo(HaveOccurred())

	ipHdr := *ipv6Default
	ipHdr.DstIP = net.IP([]byte{0x91, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 81})
	_, _, _, _, pktBytes, err := testPacketV6(ethDefault, &ipHdr, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	res, err := bpftoolProgRun(pinPath, pktBytes, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(res.Retval).To(Equal(1))

	ipHdr.DstIP = net.IP([]byte{0x91, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 82})
	_, _, _, _, pktBytes, err = testPacketV6(ethDefault, &ipHdr, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	res, err = bpftoolProgRun(pinPath, pktBytes, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(res.Retval).To(Equal(1))

	ipHdr.DstIP = net.IP([]byte{0x91, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 83})
	_, _, _, _, pktBytes, err = testPacketV6(ethDefault, &ipHdr, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	res, err = bpftoolProgRun(pinPath, pktBytes, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(res.Retval).To(Equal(0))
}

func TestMatchBPFIpsetsProgramForIPTables(t *testing.T) {
	RegisterTestingT(t)
	err := iptables.CreateDNSObjPinDir("debug")
	Expect(err).NotTo(HaveOccurred())

	defer iptables.Cleanup()
	setID := uint64(1234)

	err = iptables.LoadIPSetsPolicyProgram(setID, "debug", 4)
	Expect(err).NotTo(HaveOccurred())
	pinPath := bpfdefs.IPSetMatchProg(setID, 4, "debug")
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())

	defer os.RemoveAll(pinPath)
	err = ipsMap.Update(ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("91.189.91.81/32").(ip.V4CIDR), 0, 0).AsBytes(), ipsets.DummyValue)
	Expect(err).NotTo(HaveOccurred())
	err = ipsMap.Update(ipsets.MakeBPFIPSetEntry(setID, ip.CIDRFromStringNoErr("91.189.91.82/32").(ip.V4CIDR), 0, 0).AsBytes(), ipsets.DummyValue)
	Expect(err).NotTo(HaveOccurred())

	ipHdr := *ipv4Default
	ipHdr.DstIP = net.IPv4(91, 189, 91, 81)
	_, _, _, _, pktBytes, err := testPacketV4(ethDefault, &ipHdr, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	res, err := bpftoolProgRun(pinPath, pktBytes, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(res.Retval).To(Equal(1))

	ipHdr.DstIP = net.IPv4(91, 189, 91, 82)
	_, _, _, _, pktBytes, err = testPacketV4(ethDefault, &ipHdr, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	res, err = bpftoolProgRun(pinPath, pktBytes, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(res.Retval).To(Equal(1))

	ipHdr.DstIP = net.IPv4(91, 189, 91, 83)
	_, _, _, _, pktBytes, err = testPacketV4(ethDefault, &ipHdr, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	res, err = bpftoolProgRun(pinPath, pktBytes, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(res.Retval).To(Equal(0))
}
