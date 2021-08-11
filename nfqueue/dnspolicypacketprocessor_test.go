// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package nfqueue_test

import (
	"net"
	"time"

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

				processor := nfqueue.NewDNSPolicyPacketProcessor(
					nf,
					dnrMarkBit,
					nfqueue.WithPacketDropTimeout(100*time.Millisecond),
					nfqueue.WithPacketReleaseTimeout(20*time.Millisecond),
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

				processor := nfqueue.NewDNSPolicyPacketProcessor(
					nf,
					dnrMarkBit,
					nfqueue.WithPacketDropTimeout(100*time.Millisecond),
					nfqueue.WithPacketReleaseTimeout(20*time.Millisecond),
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

				processor := nfqueue.NewDNSPolicyPacketProcessor(
					nf,
					dnrMarkBit,
					nfqueue.WithPacketDropTimeout(100*time.Millisecond),
					nfqueue.WithPacketReleaseTimeout(20*time.Millisecond),
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

				processor := nfqueue.NewDNSPolicyPacketProcessor(
					nf,
					dnrMarkBit,
					nfqueue.WithPacketDropTimeout(100*time.Millisecond),
					nfqueue.WithPacketReleaseTimeout(20*time.Millisecond),
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
