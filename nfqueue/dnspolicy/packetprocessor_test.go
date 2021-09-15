// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dnspolicy_test

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"

	nfqdnspolicy "github.com/projectcalico/felix/nfqueue/dnspolicy"

	gonfqueue "github.com/florianl/go-nfqueue"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/felix/nfqueue"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	dnrMarkBit = uint32(0x400000)
)

var _ = Describe("DNSPolicyPacketProcessor", func() {
	Context("Packet hasn't expired", func() {
		When("a single packet is sent for processing", func() {
			It("releases the packet after holding it for it's maximum duration", func() {
				packetID := uint32(2)
				packetPayload := newTestIPV4Packet(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, layers.TCPPort(300), layers.TCPPort(600))

				ch := make(chan gonfqueue.Attribute)
				defer close(ch)

				var readOnlyCh <-chan gonfqueue.Attribute
				readOnlyCh = ch

				nf := new(nfqueue.MockNfqueue)
				nf.On("SetVerdict", uint32(2), gonfqueue.NfRepeat).Return(nil)
				nf.On("PacketAttributesChannel").Return(readOnlyCh)

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithPacketDropTimeout(100*time.Millisecond),
					nfqdnspolicy.WithPacketReleaseTimeout(20*time.Millisecond),
				)
				processor.Start()

				now := time.Now()

				ch <- gonfqueue.Attribute{
					Timestamp: &now,
					PacketID:  &packetID,
					Payload:   &packetPayload,
				}
				defer processor.Stop()

				time.Sleep(200 * time.Millisecond)
				nf.AssertExpectations(GinkgoT())
			})
		})

		When("multiple packets are sent for processing", func() {
			It("releases all packets after holding them for the maximum duration", func() {
				numPackets := 10

				// create a channel with a capacity of 10 so we can load up the channel with values
				ch := make(chan gonfqueue.Attribute, numPackets)
				defer close(ch)

				var readOnlyCh <-chan gonfqueue.Attribute
				readOnlyCh = ch

				nf := new(nfqueue.MockNfqueue)
				nf.On("SetVerdict", uint32(2), gonfqueue.NfRepeat).Return(nil)
				nf.On("PacketAttributesChannel").Return(readOnlyCh)

				now := time.Now()
				for i := 1; i <= numPackets; i++ {
					packetID := uint32(i)
					packetPayload := newTestIPV4Packet(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, byte(i)},
						layers.TCPPort(300), layers.TCPPort(600))

					nf.On("SetVerdict", uint32(i), gonfqueue.NfRepeat).Return(nil)

					ch <- gonfqueue.Attribute{
						Timestamp: &now,
						PacketID:  &packetID,
						Payload:   &packetPayload,
					}
				}

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithPacketDropTimeout(100*time.Millisecond),
					nfqdnspolicy.WithPacketReleaseTimeout(20*time.Millisecond),
				)
				processor.Start()

				defer processor.Stop()

				time.Sleep(200 * time.Millisecond)
				nf.AssertExpectations(GinkgoT())
			})
		})
	})

	Context("Dropping expired packets", func() {
		When("when the same packet is sent twice and the second time is after the timeout", func() {
			It("drops the packet on the second attempt", func() {
				packetID := uint32(2)
				packetPayload := newTestIPV4Packet(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, layers.TCPPort(300), layers.TCPPort(600))

				attrChan := make(chan gonfqueue.Attribute)
				defer close(attrChan)

				var readOnlyattrChan <-chan gonfqueue.Attribute
				readOnlyattrChan = attrChan

				nf := new(nfqueue.MockNfqueue)
				nf.On("SetVerdict", uint32(2), gonfqueue.NfRepeat).Return(nil).Once()
				nf.On("SetVerdictWithMark", uint32(2), gonfqueue.NfRepeat, int(dnrMarkBit)).Return(nil).Once()
				nf.On("PacketAttributesChannel").Return(readOnlyattrChan)

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithPacketDropTimeout(100*time.Millisecond),
					nfqdnspolicy.WithPacketReleaseTimeout(20*time.Millisecond),
				)

				processor.Start()
				defer processor.Stop()

				packet := gonfqueue.Attribute{
					PacketID: &packetID,
					Payload:  &packetPayload,
				}

				attrChan <- packet

				time.Sleep(200 * time.Millisecond)

				attrChan <- packet

				time.Sleep(200 * time.Millisecond)

				nf.AssertExpectations(GinkgoT())
			})
		})
		When("when a packet comes through with the dnr bit set", func() {
			It("drops the packet on the second immediately", func() {
				packetID := uint32(2)
				packetPayload := newTestIPV4Packet(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, layers.TCPPort(300), layers.TCPPort(600))

				attrChan := make(chan gonfqueue.Attribute)
				defer close(attrChan)

				var readOnlyattrChan <-chan gonfqueue.Attribute
				readOnlyattrChan = attrChan

				nf := new(nfqueue.MockNfqueue)
				nf.On("SetVerdict", uint32(2), gonfqueue.NfDrop).Return(nil).Once()
				nf.On("PacketAttributesChannel").Return(readOnlyattrChan)

				processor := nfqdnspolicy.NewPacketProcessor(
					nf,
					dnrMarkBit,
					nfqdnspolicy.WithPacketDropTimeout(100*time.Millisecond),
					nfqdnspolicy.WithPacketReleaseTimeout(20*time.Millisecond),
				)

				processor.Start()
				defer processor.Stop()

				mark := dnrMarkBit
				packet := gonfqueue.Attribute{
					PacketID: &packetID,
					Mark:     &mark,
					Payload:  &packetPayload,
				}

				attrChan <- packet

				time.Sleep(200 * time.Millisecond)

				nf.AssertExpectations(GinkgoT())
			})
		})
	})
})

var _ = Describe("DNSPolicyPacketProcessorWithNfqueueRestarter", func() {
	When("Nfqueue sends a shutdown signal", func() {
		It("recreates the nfqueue connections and continues processing new packets", func() {
			packetID := uint32(2)
			packetPayload := newTestIPV4Packet(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, layers.TCPPort(300), layers.TCPPort(600))

			attrChan1, attrChan2 := make(chan gonfqueue.Attribute), make(chan gonfqueue.Attribute)
			shutdownChan1, shutdownChan2 := make(chan struct{}), make(chan struct{})

			defer close(attrChan1)
			defer close(attrChan2)

			var readOnlyattrChan1, readOnlyattrChan2 <-chan gonfqueue.Attribute
			readOnlyattrChan1 = attrChan1
			readOnlyattrChan2 = attrChan2

			var readOnlyShutdownChan1, readOnlyShutdownChan2 <-chan struct{}
			readOnlyShutdownChan1 = shutdownChan1
			readOnlyShutdownChan2 = shutdownChan2

			var wg1, wg2, wg3 sync.WaitGroup

			nf1 := new(nfqueue.MockNfqueue)

			wg1.Add(2)
			nf1.On("PacketAttributesChannel").Run(func(arguments mock.Arguments) {
				wg1.Done()
			}).Return(readOnlyattrChan1)
			nf1.On("ShutdownNotificationChannel").Run(func(arguments mock.Arguments) {
				wg1.Done()
			}).Return(readOnlyShutdownChan1)

			nf2 := new(nfqueue.MockNfqueue)

			nf2.On("PacketAttributesChannel").Return(readOnlyattrChan2)
			nf2.On("ShutdownNotificationChannel").Return(readOnlyShutdownChan2)
			wg3.Add(1)
			nf2.On("Close").Run(func(args mock.Arguments) {
				wg3.Done()
			}).Return(nil)
			wg2.Add(1)
			nf2.On("SetVerdict", uint32(2), gonfqueue.NfRepeat).Run(func(arguments mock.Arguments) {
				wg2.Done()
			}).Return(nil)

			nfs := make(chan *nfqueue.MockNfqueue, 2)
			nfs <- nf1
			nfs <- nf2

			restarter := nfqdnspolicy.NewPacketProcessorWithNfqueueRestarter(func() (nfqueue.Nfqueue, error) {
				select {
				case nf := <-nfs:
					return nf, nil
				default:
					return nil, fmt.Errorf("failed to create nfqueue")
				}
			}, dnrMarkBit)

			go func() {
				restarter.Start()
			}()

			// TODO make this a timeout wait.
			wg1.Wait()

			close(shutdownChan1)

			attrChan2 <- gonfqueue.Attribute{
				PacketID: &packetID,
				Payload:  &packetPayload,
			}

			wg2.Wait()

			restarter.Stop()

			wg3.Wait()

			nf1.AssertExpectations(GinkgoT())
			nf2.AssertExpectations(GinkgoT())
		})
	})
})

func newTestIPV4Packet(srcIP, dstIP net.IP, srcPort, dstPort layers.TCPPort) []byte {
	buff := gopacket.NewSerializeBuffer()

	err := (&layers.TCP{
		DataOffset: 5,
		SrcPort:    srcPort,
		DstPort:    dstPort,
	}).SerializeTo(buff, gopacket.SerializeOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	err = (&layers.IPv4{
		Version:  4,
		IHL:      5,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}).SerializeTo(buff, gopacket.SerializeOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	return buff.Bytes()
}
