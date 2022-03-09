// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package dnsresponsepacket

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/calico/felix/dataplane/common"
	"github.com/projectcalico/calico/felix/nfqueue"
	cprometheus "github.com/projectcalico/calico/libcalico-go/lib/prometheus"
)

var (
	prometheusNfqueueQueuedLatency = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_packet_nfqueue_monitor_queued_latency",
		Help: "Summary of the length of time DNS response packets were in the nfqueue queue before they were received in userspace",
	})

	prometheusPacketHoldTime = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_packet_nfqueue_monitor_hold_time",
		Help: "Summary of the length of time the DNS response packets were held in userspace",
	})

	prometheusPacketsHeld = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_dns_packet_nfqueue_monitor_num_unreleased_packets",
		Help: "Gauge of the number of DNS response packets to release currently in memory",
	})

	prometheusNfqueueShutdownCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_packet_nfqueue_monitor_shutdown_count",
		Help: "Count of how many times nfqueue was shutdown due to a fatal error",
	})

	prometheusNfqueueVerdictFailCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_packet_nfqueue_monitor_verdict_failed",
		Help: "Count of how many times that the packet processor has failed to set the verdict",
	})

	prometheusPacketsSeen = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_packet_nfqueue_monitor_packets_in",
		Help: "Count of how many DNS response packets have been seen",
	})

	prometheusReleasedProgrammed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_packet_nfqueue_monitor_packets_released_programmed",
		Help: "Count of how many DNS response packets have been released after programming",
	})

	prometheusReleasedTimeout = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_packet_nfqueue_monitor_packets_released_timeout",
		Help: "Count of how many DNS response packets have been released due to timeout",
	})

	prometheusDroppedConnClosed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_packet_nfqueue_monitor_packets_released_conn_closed",
		Help: "Count of how many DNS response packets in userspace have been dropped due to an NFQUEUE connection close",
	})
)

func init() {
	prometheus.MustRegister(
		prometheusNfqueueShutdownCount,
		prometheusNfqueueVerdictFailCount,
		prometheusNfqueueQueuedLatency,
		prometheusPacketHoldTime,
		prometheusPacketsHeld,
		prometheusPacketsSeen,
		prometheusReleasedProgrammed,
		prometheusReleasedTimeout,
		prometheusDroppedConnClosed,
	)
}

const (
	holdTimeCheckInterval = 50 * time.Millisecond
)

type PacketProcessor interface {
	Start()
	Stop()
}

func New(
	queueID uint16, queueLength uint32, maxHoldDuration time.Duration, domainInfoStore *common.DomainInfoStore,
) PacketProcessor {
	options := []nfqueue.Option{
		nfqueue.OptMaxQueueLength(queueLength),
		// The max size of DNS packet over UDP is 512 bytes, but set a max of 1024 to handle any additional
		// encapsulation.
		nfqueue.OptMaxPacketLength(1024),
		nfqueue.OptMaxHoldTime(maxHoldDuration),
		nfqueue.OptHoldTimeCheckInterval(holdTimeCheckInterval),
		// Fail open ensures packets are accepted if the queue is full.
		nfqueue.OptFailOpen(),
		nfqueue.OptPacketsSeenCounter(prometheusPacketsSeen),
		nfqueue.OptPacketsHeldGauge(prometheusPacketsHeld),
		nfqueue.OptShutdownCounter(prometheusNfqueueShutdownCount),
		nfqueue.OptSetVerdictFailureCounter(prometheusNfqueueVerdictFailCount),
		nfqueue.OptPacketReleasedAfterHoldTimeCounter(prometheusReleasedTimeout),
		nfqueue.OptPacketReleasedCounter(prometheusReleasedProgrammed),
		nfqueue.OptPacketInNfQueueSummary(prometheusNfqueueQueuedLatency),
		nfqueue.OptConnectionClosedDroppedCounter(prometheusDroppedConnClosed),
		nfqueue.OptPacketHoldTimeSummary(prometheusPacketHoldTime),
	}

	return &packetProcessor{
		nfc: nfqueue.NewNfQueueConnector(queueID, &handler{domainInfoStore}, options...),
	}
}

type packetProcessor struct {
	domainInfoStore *common.DomainInfoStore
	nfc             nfqueue.NfQueueConnector
	cancel          context.CancelFunc
}

func (p *packetProcessor) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.nfc.Connect(ctx)
}

func (p *packetProcessor) Stop() {
	p.cancel()
}

type handler struct {
	domainInfoStore *common.DomainInfoStore
}

func (h *handler) OnPacket(packet nfqueue.Packet) {
	var timestamp uint64
	if packet.Timestamp != nil {
		timestamp = uint64(packet.Timestamp.Unix())
	}
	h.domainInfoStore.MsgChannel() <- common.DataWithTimestamp{
		Data:      packet.Payload,
		Timestamp: timestamp,
		Callback:  packet.Release,
	}
}

func (_ *handler) OnRelease(_ uint32, _ nfqueue.ReleaseReason) {
	// no-op
}
