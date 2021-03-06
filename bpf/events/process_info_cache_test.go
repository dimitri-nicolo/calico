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
	gcInterval         = time.Millisecond
	ttl                = time.Second
	eventuallyTimeout  = 3 * time.Second // 3 times to TTL to avoid any flakes.
	eventuallyInterval = 10 * time.Millisecond
)

var (
	ip1           = ipStrTo16Byte("10.128.0.14")
	ip2           = ipStrTo16Byte("10.128.0.7")
	tuple1        = collector.MakeTuple(ip1, ip2, 6, 40000, 80)
	processEvent1 = events.EventProtoStats{
		Proto:       uint32(6),
		Saddr:       [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 128, 0, 14}, // 10.128.0.14
		Daddr:       [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 128, 0, 7},  // 10.128.0.7
		Sport:       uint16(40000),
		Dport:       uint16(80),
		ProcessName: [events.ProcessNameLen]byte{99, 117, 114, 108, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Pid:         uint32(12345),
	}
	tcpStatsEvent1 = events.EventTcpStats{
		Saddr:             [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 128, 0, 14}, // 10.128.0.14
		Daddr:             [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 128, 0, 7},  // 10.128.0.7
		Sport:             uint16(40000),
		Dport:             uint16(80),
		SendCongestionWnd: 10,
		SmoothRtt:         1234,
		MinRtt:            256,
		Mss:               128,
		TotalRetrans:      2,
		LostOut:           3,
		UnrecoveredRTO:    4,
	}
	processEvent1DifferentProcessName = events.EventProtoStats{
		Proto:       uint32(6),
		Saddr:       [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 128, 0, 14}, // 10.128.0.14
		Daddr:       [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 10, 128, 0, 7},  // 10.128.0.7
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
		pic              collector.ProcessInfoCache
		testProcessChan  chan events.EventProtoStats
		testTcpStatsChan chan events.EventTcpStats
	)

	eventuallyCheckCache := func(tuple collector.Tuple, dir collector.TrafficDirection, expectedProcessInfo collector.ProcessInfo, infoInCache bool) {
		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(tuple, dir)
			return lookupResult{processInfo, ok}
		}, eventuallyTimeout, eventuallyInterval).Should(Equal(lookupResult{expectedProcessInfo, infoInCache}))
	}

	BeforeEach(func() {
		testProcessChan = make(chan events.EventProtoStats, 10)
		testTcpStatsChan = make(chan events.EventTcpStats, 10)
		pic = events.NewBPFProcessInfoCache(testProcessChan, testTcpStatsChan, gcInterval, ttl)
		pic.Start()
	})
	It("Should cache process information", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := collector.ProcessInfo{}

		eventuallyCheckCache(tuple1, collector.TrafficDirOutbound, expectedProcessInfo, false)

		By("Sending a process info event")
		testProcessChan <- processEvent1
		testTcpStatsChan <- tcpStatsEvent1

		//time.Sleep(1 * time.Millisecond)
		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = collector.ProcessInfo{
			Tuple: tuple1,
			ProcessData: collector.ProcessData{
				Name: "curl",
				Pid:  12345,
			},
			TcpStatsData: collector.TcpStatsData{
				SendCongestionWnd: 10,
				SmoothRtt:         1234,
				MinRtt:            256,
				Mss:               128,
				TotalRetrans:      2,
				LostOut:           3,
				UnrecoveredRTO:    4,
				IsDirty:           true,
			},
		}
		eventuallyCheckCache(tuple1, collector.TrafficDirOutbound, expectedProcessInfo, true)

		By("replacing the process info event")
		testProcessChan <- processEvent1DifferentProcessName

		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = collector.ProcessInfo{
			Tuple: tuple1,
			ProcessData: collector.ProcessData{
				Name: "wget",
				Pid:  54321,
			},
			TcpStatsData: collector.TcpStatsData{
				SendCongestionWnd: 10,
				SmoothRtt:         1234,
				MinRtt:            256,
				Mss:               128,
				TotalRetrans:      2,
				LostOut:           3,
				UnrecoveredRTO:    4,
				IsDirty:           true,
			},
		}
		eventuallyCheckCache(tuple1, collector.TrafficDirOutbound, expectedProcessInfo, true)
	})
	It("Should expire cached process information", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := collector.ProcessInfo{}
		eventuallyCheckCache(tuple1, collector.TrafficDirOutbound, expectedProcessInfo, false)

		By("Sending a process info event")
		testProcessChan <- processEvent1

		By("Checking that lookup returns process information")
		expectedProcessInfo = collector.ProcessInfo{
			Tuple: tuple1,
			ProcessData: collector.ProcessData{
				Name: "curl",
				Pid:  12345,
			},
		}
		eventuallyCheckCache(tuple1, collector.TrafficDirOutbound, expectedProcessInfo, true)

		By("Checking that lookup expires process information")
		expectedProcessInfo = collector.ProcessInfo{}

		eventuallyCheckCache(tuple1, collector.TrafficDirOutbound, expectedProcessInfo, false)
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
