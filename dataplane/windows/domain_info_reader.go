// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package windataplane

import (
	log "github.com/sirupsen/logrus"

	fc "github.com/projectcalico/felix/config"
	"github.com/projectcalico/felix/dataplane/common"

	"github.com/projectcalico/felix/dataplane/windows/etw"
)

const (
	windowsPacketETWSession = "tigera-windows-etw-packet"
)

type domainInfoReader struct {
	// Channel that we write to when we want DNS response capture to stop.
	stopChannel chan struct{}

	// Channel on which we receive captured DNS responses (beginning with the IP header) from ETW.
	msgChannel chan *etw.PktEvent

	// Channel on which domainInfoStore receive captured DNS responses (beginning with the IP header).
	storeMsgChannel chan common.DataWithTimestamp

	// Trusted Servers for DNS packet.
	trustedServers []etw.ServerPort

	// ETW operations
	etwOps *etw.EtwOperations
}

func NewDomainInfoReader(trustedServers []fc.ServerPort) *domainInfoReader {
	log.WithField("serverports", trustedServers).Info("Creating Windows domain info reader")
	if len(trustedServers) == 0 {
		log.Fatal("Should have at least one DNS trusted servers.")
	}

	serverPorts := []etw.ServerPort{}

	for _, server := range trustedServers {
		serverPorts = append(serverPorts, etw.ServerPort{
			IP:   server.IP,
			Port: server.Port,
		})
	}

	etwOps, err := etw.NewEtwOperations([]int{etw.PKTMON_EVENT_ID_CAPTURE}, etw.EtwPktProcessor(windowsPacketETWSession))
	if err != nil {
		log.Fatalf("Failed to create ETW operations; %s", err)
	}

	return &domainInfoReader{
		stopChannel:    make(chan struct{}),
		// domainInfoReader forward DNS message to domainInfoStore as soon as it get it.
		// Both domainInfoStore and Windows ETW package caches DNS message so we don't need
		// to cache them here. Set buffer to 10.
		msgChannel:     make(chan *etw.PktEvent, 10),
		trustedServers: serverPorts,
		etwOps:         etwOps,
	}
}

// Start function starts the reader and connect it with domainInfoStore.
func (r *domainInfoReader) Start(msgChan chan common.DataWithTimestamp) {
	log.Info("Starting Windows domain info reader")

	r.storeMsgChannel = msgChan

	r.etwOps.SubscribeToPktMon(r.msgChannel, r.stopChannel, r.trustedServers)

	go r.loop()
}

func (r *domainInfoReader) Stop() {
	r.stopChannel <- struct{}{}
}

func (r *domainInfoReader) loop() {
	for {
		r.loopIteration()
	}
}

func (r *domainInfoReader) loopIteration() {
	select {
	case pktEvent := <-r.msgChannel:
		// Forward to domainInfoStore.
		r.storeMsgChannel <- common.DataWithTimestamp{
			Timestamp: pktEvent.NanoSeconds(),
			Data:      pktEvent.Payload(),
		}
	}
}
