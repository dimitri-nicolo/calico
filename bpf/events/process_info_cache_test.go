// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package events_test

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/bpf/events"
	"github.com/projectcalico/felix/collector"
)

var (
	gcInterval = time.Millisecond
	ttl        = time.Second
)

var (
	ip1           = ipStrTo16Byte("10.128.0.14")
	ip2           = ipStrTo16Byte("10.128.0.7")
	tuple1        = collector.MakeTuple(ip1, ip2, 6, 40000, 80)
	processEvent1 = events.EventProtoStatsV4{
		Proto:       uint32(6),
		Saddr:       uint32(176160782), // 10.128.0.14
		Daddr:       uint32(176160775), // 10.128.0.7
		Sport:       uint16(40000),
		Dport:       uint16(80),
		ProcessName: [events.ProcessNameLen]byte{99, 117, 114, 108, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Pid:         uint32(12345),
	}

	processEvent1DifferentProcessName = events.EventProtoStatsV4{
		Proto:       uint32(6),
		Saddr:       uint32(176160782), // 10.128.0.14
		Daddr:       uint32(176160775), // 10.128.0.7
		Sport:       uint16(40000),
		Dport:       uint16(80),
		ProcessName: [events.ProcessNameLen]byte{119, 103, 101, 116, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Pid:         uint32(54321),
	}
)

type lookupResult struct {
	name collector.ProcessInfo
	ok   bool
}

var _ = Describe("ProcessInfoCache tests", func() {
	var (
		pic      collector.ProcessInfoCache
		testChan chan events.EventProtoStatsV4
	)
	BeforeEach(func() {
		testChan = make(chan events.EventProtoStatsV4, 10)
		pic = events.NewBPFProcessInfoCache(testChan, gcInterval, ttl)
		pic.Start()
	})
	It("Should cache process information", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := collector.ProcessInfo{}

		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple1, collector.TrafficDirOutbound)
			return lookupResult{processInfo, ok}
		}).Should(Equal(lookupResult{expectedProcessInfo, false}))

		By("Sending a process info event")
		testChan <- processEvent1

		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = collector.ProcessInfo{
			Tuple: tuple1,
			ProcessData: collector.ProcessData{
				Name: "curl",
				Pid:  12345,
			},
		}
		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple1, collector.TrafficDirOutbound)
			return lookupResult{processInfo, ok}
		}).Should(Equal(lookupResult{expectedProcessInfo, true}))

		By("replacing the process info event")
		testChan <- processEvent1DifferentProcessName

		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = collector.ProcessInfo{
			Tuple: tuple1,
			ProcessData: collector.ProcessData{
				Name: "wget",
				Pid:  54321,
			},
		}
		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple1, collector.TrafficDirOutbound)
			return lookupResult{processInfo, ok}
		}).Should(Equal(lookupResult{expectedProcessInfo, true}))
	})
	It("Should expire cached process information", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := collector.ProcessInfo{}

		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple1, collector.TrafficDirOutbound)
			return lookupResult{processInfo, ok}
		}).Should(Equal(lookupResult{expectedProcessInfo, false}))

		By("Sending a process info event")
		testChan <- processEvent1

		By("Checking that lookup returns process information")
		expectedProcessInfo = collector.ProcessInfo{
			Tuple: tuple1,
			ProcessData: collector.ProcessData{
				Name: "curl",
				Pid:  12345,
			},
		}
		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple1, collector.TrafficDirOutbound)
			return lookupResult{processInfo, ok}
		}).Should(Equal(lookupResult{expectedProcessInfo, true}))

		By("Checking that lookup expires process information")
		expectedProcessInfo = collector.ProcessInfo{}

		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple1, collector.TrafficDirOutbound)
			return lookupResult{processInfo, ok}
		}).Should(Equal(lookupResult{expectedProcessInfo, false}))
	})
	AfterEach(func() {
		pic.Stop()
	})
})

func ipStrTo16Byte(ipStr string) [16]byte {
	addr := net.ParseIP(ipStr)
	return ipTo16Byte(addr)
}

func ipTo16Byte(addr net.IP) [16]byte {
	var addrB [16]byte
	copy(addrB[:], addr.To16()[:16])
	return addrB
}
