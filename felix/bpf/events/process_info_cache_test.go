// Copyright (c) 2021-2023 Tigera, Inc. All rights reserved.

package events_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/bpf/events"
	"github.com/projectcalico/calico/felix/collector/dataplane"
	"github.com/projectcalico/calico/felix/collector/types/tuple"
	"github.com/projectcalico/calico/felix/collector/utils"
)

var (
	gcInterval         = time.Millisecond
	ttl                = time.Second
	eventuallyTimeout  = 3 * time.Second // 3 times to TTL to avoid any flakes.
	eventuallyInterval = 10 * time.Millisecond
)

var (
	ip1           = utils.IpStrTo16Byte("10.128.0.14")
	ip2           = utils.IpStrTo16Byte("10.128.0.7")
	tuple1        = tuple.Make(ip1, ip2, 6, 40000, 80)
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
	processPathEvent1 = events.ProcessPath{
		Pid:       12345,
		Filename:  "/usr/bin/curl",
		Arguments: "example.com",
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
	name dataplane.ProcessInfo
	ok   bool
}

var _ = Describe("ProcessInfoCache tests", func() {
	var (
		pic                 dataplane.ProcessInfoCache
		testProcessChan     chan events.EventProtoStats
		testTcpStatsChan    chan events.EventTcpStats
		testProcessPathChan chan events.ProcessPath
	)

	eventuallyCheckCache := func(key tuple.Tuple, dir dataplane.TrafficDirection, expectedProcessInfo dataplane.ProcessInfo, infoInCache bool) {
		Eventually(func() lookupResult {
			processInfo, ok := pic.Lookup(key, dir)
			return lookupResult{processInfo, ok}
		}, eventuallyTimeout, eventuallyInterval).Should(Equal(lookupResult{expectedProcessInfo, infoInCache}))
	}

	BeforeEach(func() {
		testProcessChan = make(chan events.EventProtoStats, 10)
		testTcpStatsChan = make(chan events.EventTcpStats, 10)
		testProcessPathChan = make(chan events.ProcessPath, 10)
		pp := events.NewBPFProcessPathCache(testProcessPathChan, gcInterval, 30*ttl)
		pic = events.NewBPFProcessInfoCache(testProcessChan, testTcpStatsChan, gcInterval, ttl, pp)
		pic.Start()
	})
	It("Should cache process information", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := dataplane.ProcessInfo{}

		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, false)

		By("Sending a process info event")
		testProcessChan <- processEvent1
		testTcpStatsChan <- tcpStatsEvent1

		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = dataplane.ProcessInfo{
			Tuple: tuple1,
			ProcessData: dataplane.ProcessData{
				Name: "curl",
				Pid:  12345,
			},
			TcpStatsData: dataplane.TcpStatsData{
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
		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, true)

		By("replacing the process info event")
		testProcessChan <- processEvent1DifferentProcessName

		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = dataplane.ProcessInfo{
			Tuple: tuple1,
			ProcessData: dataplane.ProcessData{
				Name: "wget",
				Pid:  54321,
			},
			TcpStatsData: dataplane.TcpStatsData{
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
		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, true)
	})
	It("Should cache process path information if available", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := dataplane.ProcessInfo{}

		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, false)

		By("Sending a process info event, path event")
		testProcessPathChan <- processPathEvent1
		time.Sleep(1 * time.Millisecond)
		testProcessChan <- processEvent1
		testTcpStatsChan <- tcpStatsEvent1

		//time.Sleep(1 * time.Millisecond)
		By("Checking that lookup returns process information and is converted correctly")
		expectedProcessInfo = dataplane.ProcessInfo{
			Tuple: tuple1,
			ProcessData: dataplane.ProcessData{
				Name:      "/usr/bin/curl",
				Pid:       12345,
				Arguments: "example.com",
			},
			TcpStatsData: dataplane.TcpStatsData{
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
		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, true)
	})
	It("Should expire cached process information", func() {
		By("Checking that lookup cache doesn't contain the right process info")
		expectedProcessInfo := dataplane.ProcessInfo{}
		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, false)

		By("Sending a process info event")
		testProcessChan <- processEvent1

		By("Checking that lookup returns process information")
		expectedProcessInfo = dataplane.ProcessInfo{
			Tuple: tuple1,
			ProcessData: dataplane.ProcessData{
				Name: "curl",
				Pid:  12345,
			},
		}
		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, true)

		By("Checking that lookup expires process information")
		expectedProcessInfo = dataplane.ProcessInfo{}

		eventuallyCheckCache(tuple1, dataplane.TrafficDirOutbound, expectedProcessInfo, false)
	})
	AfterEach(func() {
		pic.Stop()
	})
})
