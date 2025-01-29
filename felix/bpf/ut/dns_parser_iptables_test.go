// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package ut_test

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/bpf/bpfdefs"
	"github.com/projectcalico/calico/felix/bpf/bpfmap"
	"github.com/projectcalico/calico/felix/bpf/iptables"
)

func TestBPFDnsParserProgramForIPTables(t *testing.T) {
	RegisterTestingT(t)
	defer iptables.Cleanup()

	err := iptables.LoadDNSParserBPFProgram("debug", bpfmap.DNSMapsToPin())
	Expect(err).NotTo(HaveOccurred())

	pinPath := bpfdefs.DnsObjDir + "/debug/cali_ipt_parse_dns"
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())

	defer os.RemoveAll(pinPath)
	testDNSParser(t, pinPath, true)

	err = iptables.LoadDNSParserBPFProgram("off", bpfmap.DNSMapsToPin())
	Expect(err).NotTo(HaveOccurred())

	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())
	pinPath = bpfdefs.DnsObjDir + "/no_log/cali_ipt_parse_dns"
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())
}

func TestIPSetPinCleanup(t *testing.T) {
	RegisterTestingT(t)
	err := iptables.LoadDNSParserBPFProgram("debug", bpfmap.DNSMapsToPin())
	Expect(err).NotTo(HaveOccurred())
	defer iptables.Cleanup()

	setID := uint64(1234)

	err = iptables.LoadIPSetsPolicyProgram(setID, "debug", 4)
	Expect(err).NotTo(HaveOccurred())
	pinPath := bpfdefs.IPSetMatchProg(setID, 4, "debug")
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())

	err = iptables.CleanupOld("debug")
	Expect(err).NotTo(HaveOccurred())
	// After cleanup, we should just see the parser and not the ipset matcher.
	_, err = os.Stat(pinPath)
	Expect(err).To(HaveOccurred())
	pinPath = bpfdefs.DnsObjDir + "/debug/cali_ipt_parse_dns"
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())

	// Changing the logLevel from debug to off.
	err = iptables.LoadDNSParserBPFProgram("off", bpfmap.DNSMapsToPin())
	Expect(err).NotTo(HaveOccurred())
	err = iptables.LoadIPSetsPolicyProgram(setID, "off", 4)
	Expect(err).NotTo(HaveOccurred())
	pinPath = bpfdefs.IPSetMatchProg(setID, 4, "off")
	_, err = os.Stat(pinPath)
	Expect(err).NotTo(HaveOccurred())

	// Should delete /sys/fs/bpf/dns/debug
	err = iptables.CleanupOld("off")
	Expect(err).NotTo(HaveOccurred())
	_, err = os.Stat(bpfdefs.DnsObjDir + "/debug")
	Expect(err).To(HaveOccurred())
}
