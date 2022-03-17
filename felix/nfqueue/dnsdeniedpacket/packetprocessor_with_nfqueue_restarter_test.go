// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package dnsdeniedpacket_test

import (
	"fmt"
	"net"
	"sync"

	. "github.com/onsi/ginkgo"

	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/calico/felix/nfqueue"
	nfqdnspolicy "github.com/projectcalico/calico/felix/nfqueue/dnsdeniedpacket"

	gonfqueue "github.com/florianl/go-nfqueue"

	"github.com/google/gopacket/layers"
)

var _ = Describe("DNSPolicyPacketProcessorWithNfqueueRestarter", func() {
	When("Nfqueue sends a shutdown signal", func() {
		It("recreates the nfqueue connections and continues processing new packets", func() {
			packetID := uint32(2)
			packetPayload := createPacketPayload(net.IP{8, 8, 8, 8}, net.IP{9, 9, 9, 9}, 300, 600, layers.IPProtocolTCP)

			attrChan1, attrChan2 := make(chan gonfqueue.Attribute), make(chan gonfqueue.Attribute)
			shutdownChan1, shutdownChan2 := make(chan struct{}), make(chan struct{})

			defer close(attrChan1)
			defer close(attrChan2)

			var readOnlyAttrChan1, readOnlyAttrChan2 <-chan gonfqueue.Attribute
			readOnlyAttrChan1 = attrChan1
			readOnlyAttrChan2 = attrChan2

			var readOnlyShutdownChan1, readOnlyShutdownChan2 <-chan struct{}
			readOnlyShutdownChan1 = shutdownChan1
			readOnlyShutdownChan2 = shutdownChan2

			var wg1, wg2, wg3 sync.WaitGroup

			nf1, nf2 := new(nfqueue.MockNfqueue), new(nfqueue.MockNfqueue)
			nf1.Test(GinkgoT())
			nf2.Test(GinkgoT())

			wg1.Add(2)
			nf1.On("PacketAttributesChannel").Run(func(args mock.Arguments) { wg1.Done() }).Return(readOnlyAttrChan1)
			nf1.On("ShutdownNotificationChannel").Run(func(args mock.Arguments) { wg1.Done() }).Return(readOnlyShutdownChan1)

			nf2.On("PacketAttributesChannel").Return(readOnlyAttrChan2)
			nf2.On("ShutdownNotificationChannel").Return(readOnlyShutdownChan2)
			wg3.Add(1)
			nf2.On("Close").Run(func(args mock.Arguments) { wg3.Done() }).Return(nil)

			wg2.Add(1)
			nf2.On("SetVerdictWithMark", uint32(2), gonfqueue.NfRepeat, int(dnrMarkBit)).Run(func(arguments mock.Arguments) {
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
