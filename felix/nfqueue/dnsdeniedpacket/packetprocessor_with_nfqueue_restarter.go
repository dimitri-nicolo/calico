// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package dnsdeniedpacket

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/nfqueue"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

type PacketProcessorWithNfqueueRestarter interface {
	PacketProcessor

	DebugKillCurrentNfqueueConnection() error
}

// packetProcessorWithNfqueueRestarter is an implementation of the PacketProcessor interface and is an extension of
// the packetProcessor implementation of said interface. The purpose of this implementation is to handle
type packetProcessorWithNfqueueRestarter struct {
	newNfqueue      func() (nfqueue.Nfqueue, error)
	loopKeepAliveWG sync.WaitGroup
	dnrMark         uint32
	options         []Option
	done            chan struct{}
	closeOnce       sync.Once

	ipsetMemberUpdates       chan set.Set
	debugKillNfqueueConnChan chan chan error
}

func NewPacketProcessorWithNfqueueRestarter(
	newNfqueue func() (nfqueue.Nfqueue, error),
	dnrMark uint32,
	options ...Option) PacketProcessorWithNfqueueRestarter {
	return &packetProcessorWithNfqueueRestarter{
		newNfqueue:               newNfqueue,
		dnrMark:                  dnrMark,
		options:                  options,
		done:                     make(chan struct{}),
		debugKillNfqueueConnChan: make(chan chan error),
		ipsetMemberUpdates:       make(chan set.Set),
	}
}

// Start kicks off the internal loop to start the packet processor and open the nfqueue connection, and restart if
// something should go wrong. If the loop fails to start (without being gracefully closed) then it panics.
func (restarter *packetProcessorWithNfqueueRestarter) Start() {
	restarter.loopKeepAliveWG.Add(1)
	go func() {
		if err := restarter.loopKeepAlive(); err != nil {
			log.WithError(err).Error("failed to start nfqueue dns policy packet processor")
			// We panic here because if loopKeepAlive returns unexpectedly then there will be nothing listening for
			// nfqueue packets. With nothing listening, the default for the iptables nfqueue rule is to drop the packets
			// immediately. This is fine in terms of us doing the correct thing with the packet (as it would have been
			// dropped anyways), however, we will not have a flow log for this dropped packet as the deny verdict is
			// sent to felix after the nfqueue rule.
			panic("failed to start nfqueue dns policy packet processor")
		}
	}()
}

// loopKeepAlive loops creating and cleaning up the packet process. If it detects the nfqueue connection has gone down
// it stops the current PacketProcessor, recreates the nfqueue connection, then recreates a new PacketProcessor, giving
// it the new nfqueue.
func (restarter *packetProcessorWithNfqueueRestarter) loopKeepAlive() error {
	defer restarter.loopKeepAliveWG.Done()

done:
	for {
		nf, err := restarter.newNfqueue()
		if err != nil {
			return err
		}

		processor := NewPacketProcessor(nf, restarter.dnrMark, restarter.options...)
		processor.Start()

	loop:
		for {
			select {
			case <-nf.ShutdownNotificationChannel():
				processor.Stop()
				break loop
			case <-restarter.done:
				processor.Stop()
				if err := nf.Close(); err != nil {
					log.WithError(err).Warning("an error occurred while closing nfqueue.")
				}
				break done
			case updates := <-restarter.ipsetMemberUpdates:
				processor.OnIPSetMemberUpdates(updates)
			case errChan := <-restarter.debugKillNfqueueConnChan:
				errChan <- nf.DebugKillConnection()
				close(errChan)
			}
		}

		log.Info("Recreating NFQUEUE connection...")
	}

	return nil
}

// OnIPSetMemberUpdates accepts a set of IPs which passes through to the underlying packetProcessor.
//
// Note that OnIPSetMemberUpdates must not be called after Stop() has been called on the packetProcessor.
func (restarter *packetProcessorWithNfqueueRestarter) OnIPSetMemberUpdates(ips set.Set) {
	restarter.ipsetMemberUpdates <- ips
}

func (restarter *packetProcessorWithNfqueueRestarter) Stop() {
	log.Info("Stopping PacketProcessor.")
	restarter.closeOnce.Do(func() {
		close(restarter.done)

		restarter.loopKeepAliveWG.Wait()

		close(restarter.debugKillNfqueueConnChan)
		close(restarter.ipsetMemberUpdates)
	})
}

// DebugKillCurrentNfqueueConnection calls DebugKillConnection on the currently use nfqueue instance, destroying the
// underlying connection. This is used for testing purposes only.
//
// In general, DO NOT USE THIS FUNCTION.
func (restarter *packetProcessorWithNfqueueRestarter) DebugKillCurrentNfqueueConnection() error {
	errChan := make(chan error)
	restarter.debugKillNfqueueConnChan <- errChan

	return <-errChan
}
