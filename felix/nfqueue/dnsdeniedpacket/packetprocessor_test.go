// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package dnsdeniedpacket_test

import (
	"net"
	"sync"
	"time"

	"github.com/projectcalico/calico/felix/timeshim"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/nfqueue"
	nfqdnspolicy "github.com/projectcalico/calico/felix/nfqueue/dnsdeniedpacket"

	"github.com/projectcalico/calico/libcalico-go/lib/set"

	gonfqueue "github.com/florianl/go-nfqueue"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	dnrMarkBit = uint32(0x400000)
)

func waitTimeout(success chan struct{}, waitDuration time.Duration, failMessage string) {
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	select {
	case <-success:
	case <-timer.C:
		Fail(failMessage)
	}
}

var _ = Describe("DNSPolicyPacketProcessor", func() {
	var packetAttributesChan chan gonfqueue.Attribute
	var mockTicker *timeshim.MockTicker
	var mockTime *timeshim.MockInterface
	var nf *nfqueue.MockNfqueue
	var defaultPacketID uint32

	BeforeEach(func() {
		nf = new(nfqueue.MockNfqueue)
		nf.Test(GinkgoT())

		mockTicker = &timeshim.MockTicker{}
		mockTime = &timeshim.MockInterface{}

		defaultPacketID = uint32(2)
		packetAttributesChan = make(chan gonfqueue.Attribute, 100)
	})

	AfterEach(func() {
		close(packetAttributesChan)
		nf.AssertExpectations(GinkgoT())
	})

	When("the packet attribute channel is closed then we call Stop", func() {
		It("produces no errors", func() {
			// Use a different channel then the default one since it would result in the channel getting close again in
			// the AfterEach
			packetAttributesChan2 := make(chan gonfqueue.Attribute, 100)

			var wg sync.WaitGroup
			wg.Add(1)
			nf.On("PacketAttributesChannel").Run(
				func(args mock.Arguments) {
					wg.Done()
				},
			).Return((<-chan gonfqueue.Attribute)(packetAttributesChan2))

			processor := nfqdnspolicy.NewPacketProcessor(nf, dnrMarkBit)
			processor.Start()

			// Wait for PacketAttributesChannel to be called.
			wg.Wait()

			close(packetAttributesChan2)
			processor.Stop()
		})
	})

	// This test tests that the packet is removed from the PacketProcessor.
	When("duplicate updates for the same IP come through", func() {
		It("releases the packet immediately then does nothing for the second update",
			func() {
				dstIP := net.IP{9, 9, 9, 9}
				packetReleaseTimeout := 1 * time.Second
				packetPayload := createPacketPayload(net.IP{8, 8, 8, 8}, dstIP, uint16(300), uint16(600), layers.IPProtocolTCP)

				setVerdictCalled := make(chan struct{})
				defer close(setVerdictCalled)
				nf.On("SetVerdictWithMark", defaultPacketID, gonfqueue.NfRepeat, int(dnrMarkBit)).Run(
					func(arguments mock.Arguments) {
						setVerdictCalled <- struct{}{}
					},
				).Return(nil).Once()
				nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

				tickerChan := make(chan time.Time)

				mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
				mockTicker.On("Stop").Return(true)

				mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
				mockTime.On("Now").Return(func() time.Time { return time.Now() })
				mockTime.On("Since", mock.Anything).Return(time.Duration(0))

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithPacketReleaseTimeout(packetReleaseTimeout),
					nfqdnspolicy.WithReleaseTickerDuration(100*time.Millisecond),
					nfqdnspolicy.WithTimeInterface(mockTime),
				)
				processor.Start()
				defer processor.Stop()

				packetAttributesChan <- gonfqueue.Attribute{
					PacketID: &defaultPacketID,
					Payload:  &packetPayload,
				}

				processor.OnIPSetMemberUpdates(set.From(ip.FromNetIP(dstIP)))
				time.Sleep(5 * time.Millisecond) // ensure we don't tick before the update

				tickerChan <- time.Now()
				Eventually(setVerdictCalled, 1*time.Second).Should(Receive())

				processor.OnIPSetMemberUpdates(set.From(ip.FromNetIP(dstIP)))
				time.Sleep(5 * time.Millisecond) // ensure we don't tick before the update

				tickerChan <- time.Now()

				// Wait to ensure setVerdict is not called again
				Consistently(setVerdictCalled, 1*time.Second).ShouldNot(Receive())
			},
		)
	})

	When("an update containing the destination IP of a packet that's held comes through", func() {
		DescribeTable("releases the packet immediately",
			func(srcIP, dstIP net.IP, srcPort, dstPort uint16, protocol layers.IPProtocol) {
				packetPayload := createPacketPayload(srcIP, dstIP, srcPort, dstPort, protocol)

				setVerdictCalled := make(chan struct{})
				defer close(setVerdictCalled)
				nf.On("SetVerdictWithMark", uint32(2), gonfqueue.NfRepeat, int(dnrMarkBit)).Run(
					func(arguments mock.Arguments) {
						setVerdictCalled <- struct{}{}
					},
				).Return(nil)
				nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

				tickerChan := make(chan time.Time)

				mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
				mockTicker.On("Stop").Return(true)

				mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
				mockTime.On("Now").Return(func() time.Time { return time.Now() })
				mockTime.On("Since", mock.Anything).Return(time.Duration(0))

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithTimeInterface(mockTime),
				)
				processor.Start()
				defer processor.Stop()

				packetAttributesChan <- gonfqueue.Attribute{
					PacketID: &defaultPacketID,
					Payload:  &packetPayload,
				}

				processor.OnIPSetMemberUpdates(set.From(ip.FromNetIP(dstIP)))
				time.Sleep(5 * time.Millisecond) // ensure we don't tick before the update

				tickerChan <- time.Now()
				Eventually(setVerdictCalled, 1*time.Second).Should(Receive())
			},
			Entry(
				"IPV4 TCP",
				net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, uint16(300), uint16(600), layers.IPProtocolTCP,
			),
			Entry(
				"IPV6 TCP",
				net.ParseIP("2607:f8b0:400a:80a::200e"), net.ParseIP("fc00:f853:ccd:e777::1"), uint16(300), uint16(600),
				layers.IPProtocolTCP,
			),
		)
	})

	When("the ipset update comes before the packet and the ip cache timeout has not been reached", func() {
		It("releases the packet immediately", func() {
			dstIP := net.IP{9, 9, 9, 9}
			packetPayload := createPacketPayload(net.IP{8, 8, 8, 8}, dstIP, uint16(300), uint16(600), layers.IPProtocolTCP)

			setVerdictCalled := make(chan struct{})
			defer close(setVerdictCalled)
			nf.On("SetVerdictWithMark", uint32(2), gonfqueue.NfRepeat, int(dnrMarkBit)).Run(func(arguments mock.Arguments) {
				setVerdictCalled <- struct{}{}
			}).Return(nil)
			nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

			tickerChan := make(chan time.Time)

			mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
			mockTicker.On("Stop").Return(true)

			mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
			mockTime.On("Now").Return(func() time.Time { return time.Now() })
			mockTime.On("Since", mock.Anything).Return(time.Duration(0))

			processor := nfqdnspolicy.NewPacketProcessor(
				nf,
				dnrMarkBit,
				nfqdnspolicy.WithTimeInterface(mockTime),
			)
			processor.Start()
			defer processor.Stop()

			// Help ensure that the loop in the packet processor has finished setting up and is just waiting for ips
			// and packets.
			time.Sleep(5 * time.Millisecond)

			processor.OnIPSetMemberUpdates(set.From(ip.FromNetIP(dstIP)))
			time.Sleep(5 * time.Millisecond)

			packetAttributesChan <- gonfqueue.Attribute{
				PacketID: &defaultPacketID,
				Payload:  &packetPayload,
			}

			time.Sleep(5 * time.Millisecond) // ensure we don't tick before the update

			tickerChan <- time.Now()
			Eventually(setVerdictCalled, 1*time.Second).Should(Receive())
		})
	})

	When("the ipset update comes before the packet and the ip cache timeout has been reached", func() {
		It("releases the packet after the timeout period is spent", func() {
			dstIP := net.IP{9, 9, 9, 9}
			packetPayload := createPacketPayload(net.IP{8, 8, 8, 8}, dstIP, uint16(300), uint16(600), layers.IPProtocolTCP)

			ipCacheTimeout := 200 * time.Millisecond
			packetTimeout := 200 * time.Millisecond

			verdictCalled := make(chan struct{})
			defer close(verdictCalled)
			nf.On("SetVerdictWithMark", uint32(2), gonfqueue.NfRepeat, int(dnrMarkBit)).Run(func(arguments mock.Arguments) {
				verdictCalled <- struct{}{}
			}).Return(nil)
			nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

			tickerChan := make(chan time.Time)

			mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
			mockTicker.On("Stop").Return(true)

			mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
			mockTime.On("Now").Return(func() time.Time { return time.Now() })

			processor := nfqdnspolicy.NewPacketProcessor(
				nf,
				dnrMarkBit,
				nfqdnspolicy.WithIPCacheDuration(ipCacheTimeout),
				nfqdnspolicy.WithPacketReleaseTimeout(packetTimeout),
				nfqdnspolicy.WithTimeInterface(mockTime),
			)
			processor.Start()
			defer processor.Stop()

			// Help ensure that the loop in the packet processor has finished setting up and is just waiting for ips
			// and packets.
			time.Sleep(5 * time.Millisecond)

			processor.OnIPSetMemberUpdates(set.From(ip.FromNetIP(dstIP)))
			time.Sleep(5 * time.Millisecond)

			// Since will be called to check the duration on the cached IP so we return a duration that forces it
			// to expire.
			mockTime.On("Since", mock.Anything).Return(func(t time.Time) time.Duration { return ipCacheTimeout }).Once()
			tickerChan <- time.Now()

			packetAttributesChan <- gonfqueue.Attribute{
				PacketID: &defaultPacketID,
				Payload:  &packetPayload,
			}
			time.Sleep(5 * time.Millisecond)

			// Since will now be called to check the duration on the how long the packet has been held for, so we say
			// 0 seconds so we can check that it is not released (since that would mean we still have the cached IP).
			mockTime.On("Since", mock.Anything).Return(time.Duration(0)).Once()
			tickerChan <- time.Now()

			// Now we want to actually time out the packet and see it was released, so we return the packet timeout.
			mockTime.On("Since", mock.Anything).Return(func(t time.Time) time.Duration { return packetTimeout }).Once()
			// This since is for prometheus since we keep stats on latency.
			mockTime.On("Since", mock.Anything).Return(time.Duration(0)).Once()
			tickerChan <- time.Now()

			Eventually(verdictCalled, 500*time.Millisecond).Should(Receive())
		})
	})

	When("no ipset updates come through", func() {
		DescribeTable("releases the packet after the maximum timeout has been reached",
			func(srcIP, dstIP net.IP, srcPort, dstPort uint16, protocol layers.IPProtocol) {
				packetPayload := createPacketPayload(srcIP, dstIP, srcPort, dstPort, protocol)
				releaseTimeout := 500 * time.Millisecond

				setVerdictCalled := make(chan struct{})
				defer close(setVerdictCalled)
				nf.On("SetVerdictWithMark", defaultPacketID, gonfqueue.NfRepeat, int(dnrMarkBit)).Run(func(arguments mock.Arguments) {
					setVerdictCalled <- struct{}{}
				}).Return(nil)
				nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

				tickerChan := make(chan time.Time)

				mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
				mockTicker.On("Stop").Return(true)

				mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
				mockTime.On("Now").Return(func() time.Time { return time.Now() })

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithPacketReleaseTimeout(releaseTimeout),
					nfqdnspolicy.WithTimeInterface(mockTime),
				)
				processor.Start()
				defer processor.Stop()

				packetAttributesChan <- gonfqueue.Attribute{
					PacketID: &defaultPacketID,
					Payload:  &packetPayload,
				}
				time.Sleep(5 * time.Millisecond)

				// Since will now be called to check the duration on the how long the packet has been held for, so we say
				// 0 seconds so we can check that it is not released (since that would mean we still have the cached IP).
				mockTime.On("Since", mock.Anything).Return(time.Duration(0)).Once()
				tickerChan <- time.Now()

				// Now we want to actually time out the packet and see it was released, so we return the packet timeout.
				mockTime.On("Since", mock.Anything).Return(func(t time.Time) time.Duration { return releaseTimeout }).Once()
				// This since is for prometheus since we keep stats on latency.
				mockTime.On("Since", mock.Anything).Return(time.Duration(0)).Once()
				tickerChan <- time.Now()

				Eventually(setVerdictCalled, 500*time.Millisecond).Should(Receive())
			},
			Entry(
				"IPV4 TCP",
				net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, uint16(300), uint16(600), layers.IPProtocolTCP,
			),
			Entry(
				"IPV6 TCP",
				net.ParseIP("2607:f8b0:400a:80a::200e"), net.ParseIP("fc00:f853:ccd:e777::1"), uint16(300), uint16(600),
				layers.IPProtocolTCP,
			),
		)
	})

	When("multiple ipset updates come through that match multiple packets being held", func() {
		It("releases all packets as soon as the updates come through", func() {
			numPackets := 10
			packetReleaseTimeout := 1 * time.Second
			nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

			ips := set.New()
			now := time.Now()

			tickerChan := make(chan time.Time)

			mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
			mockTicker.On("Stop").Return(true)

			mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
			mockTime.On("Now").Return(func() time.Time { return time.Now() })
			mockTime.On("Since", mock.Anything).Return(time.Duration(0))

			var wg sync.WaitGroup
			for i := 1; i <= numPackets; i++ {
				wg.Add(1)
				packetID := uint32(i)

				dstIP := net.IP{9, 9, 9, byte(i)}
				ips.Add(ip.FromNetIP(dstIP))

				packetPayload := createPacketPayload(net.IP{8, 8, 8, 8}, dstIP, 300, 600, layers.IPProtocolTCP)

				nf.On("SetVerdictWithMark", uint32(i), gonfqueue.NfRepeat, int(dnrMarkBit)).Run(
					func(args mock.Arguments) {
						wg.Done()
					},
				).Return(nil)

				packetAttributesChan <- gonfqueue.Attribute{
					Timestamp: &now,
					PacketID:  &packetID,
					Payload:   &packetPayload,
				}
			}

			done := make(chan struct{})
			defer close(done)
			go func() {
				wg.Wait()
				done <- struct{}{}
			}()

			processor := nfqdnspolicy.NewPacketProcessor(
				nf,
				dnrMarkBit,
				nfqdnspolicy.WithPacketReleaseTimeout(packetReleaseTimeout),
				nfqdnspolicy.WithTimeInterface(mockTime),
			)

			processor.Start()
			defer processor.Stop()

			processor.OnIPSetMemberUpdates(ips)
			time.Sleep(5 * time.Millisecond)

			tickerChan <- time.Now()

			Eventually(done, 500*time.Millisecond).Should(Receive())
		})
	})

	When("when a packet comes through with the dnr bit set", func() {
		It("drops the packet on the second immediately", func() {
			packetPayload := createPacketPayload(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, 300, 600, layers.IPProtocolTCP)

			tickerChan := make(chan time.Time)

			mockTicker.On("Chan").Return((<-chan time.Time)(tickerChan))
			mockTicker.On("Stop").Return(true)

			mockTime.On("NewTicker", mock.Anything).Return(mockTicker)
			mockTime.On("Now", mock.Anything).Return(func() time.Time { return time.Now() })

			setVerdictCalled := make(chan struct{})
			defer close(setVerdictCalled)
			nf.On("SetVerdict", defaultPacketID, gonfqueue.NfDrop).Run(
				func(args mock.Arguments) {
					setVerdictCalled <- struct{}{}
				},
			).Return(nil).Once()
			nf.On("PacketAttributesChannel").Return((<-chan gonfqueue.Attribute)(packetAttributesChan))

			processor := nfqdnspolicy.NewPacketProcessor(
				nf,
				dnrMarkBit,
				nfqdnspolicy.WithPacketReleaseTimeout(100*time.Millisecond),
				nfqdnspolicy.WithTimeInterface(mockTime),
			)

			processor.Start()
			defer processor.Stop()

			mark := dnrMarkBit
			packetAttributesChan <- gonfqueue.Attribute{
				PacketID: &defaultPacketID,
				Mark:     &mark,
				Payload:  &packetPayload,
			}

			Eventually(setVerdictCalled, 500*time.Millisecond).Should(Receive())
		})
	})
})

func ipv4Layer(srcIP, dstIP net.IP, buff gopacket.SerializeBuffer) gopacket.SerializeBuffer {
	defer GinkgoRecover()
	err := (&layers.IPv4{
		Version:  4,
		IHL:      5,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}).SerializeTo(buff, gopacket.SerializeOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	return buff
}

func ipv6Layer(srcIP, dstIP net.IP, nextHeader layers.IPProtocol, buff gopacket.SerializeBuffer) gopacket.SerializeBuffer {
	defer GinkgoRecover()
	err := (&layers.IPv6{
		Length:     20,
		Version:    6,
		SrcIP:      srcIP,
		DstIP:      dstIP,
		NextHeader: nextHeader,
	}).SerializeTo(buff, gopacket.SerializeOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	return buff
}

func tcpLayer(srcPort, dstPort layers.TCPPort, buff gopacket.SerializeBuffer) gopacket.SerializeBuffer {
	defer GinkgoRecover()
	err := (&layers.TCP{
		DataOffset: 5,
		SrcPort:    srcPort,
		DstPort:    dstPort,
	}).SerializeTo(buff, gopacket.SerializeOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	return buff
}

func createPacketPayload(srcIP, dstIP net.IP, srcPort, dstPort uint16, protocol layers.IPProtocol) []byte {
	var buffer gopacket.SerializeBuffer
	if protocol == layers.IPProtocolTCP {
		buffer = tcpLayer(layers.TCPPort(srcPort), layers.TCPPort(dstPort), gopacket.NewSerializeBuffer())
	}

	if len(dstIP) == net.IPv4len {
		buffer = ipv4Layer(srcIP, dstIP, buffer)
	} else {
		buffer = ipv6Layer(srcIP, dstIP, layers.IPProtocolTCP, buffer)
	}

	return buffer.Bytes()
}
