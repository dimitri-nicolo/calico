// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package nfqueue

import (
	"crypto/sha1"
	"fmt"
	"net"
	"time"

	"github.com/projectcalico/felix/nfqueue/timemanager"

	cprometheus "github.com/projectcalico/libcalico-go/lib/prometheus"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"

	gonfqueue "github.com/florianl/go-nfqueue"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var (
	prometheusNfqueueQueuedLatency = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_policy_nfqueue_monitor_queued_latency",
		Help: "Summary for measuring how long a packet was nfqueued for",
	})

	prometheusPacketReleaseLatency = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_policy_nfqueue_monitor_release_latency",
		Help: "Summary for measuring the latency of releasing packets.",
	})

	prometheusPacketDropLatency = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_policy_nfqueue_monitor_drop_latency",
		Help: "Summary for measuring the latency of dropping packets.",
	})

	prometheusReleasePacketBatchSizeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_dns_policy_nfqueue_monitor_release_packets_batch_size",
		Help: "Gauge of the number of packets to release currently in memory.",
	})

	prometheusDropPacketBatchSizeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_dns_policy_nfqueue_monitor_drop_packets_batch_size",
		Help: "Gauge of the number of packets to drop currently in memory.",
	})

	prometheusPacketsInCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_packets_in",
		Help: "Count of the number of packets that have come into the monitor",
	})

	prometheusPacketsNfDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_nf_dropped",
		Help: "Count of the number of packets that have been nf_dropped",
	})

	prometheusPacketsNfRepeatedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_nf_repeated",
		Help: "Count of the number of packets that have been nf_repeated",
	})
)

func init() {
	prometheus.MustRegister(prometheusPacketReleaseLatency)
	prometheus.MustRegister(prometheusPacketDropLatency)
	prometheus.MustRegister(prometheusReleasePacketBatchSizeGauge)
	prometheus.MustRegister(prometheusDropPacketBatchSizeGauge)
	prometheus.MustRegister(prometheusPacketsInCount)
	prometheus.MustRegister(prometheusPacketsNfDroppedCount)
	prometheus.MustRegister(prometheusPacketsNfRepeatedCount)
	prometheus.MustRegister(prometheusNfqueueQueuedLatency)
}

const (
	defaultPacketDropTimeout     = 1000 * time.Millisecond
	defaultPacketReleaseTimeout  = 300 * time.Millisecond
	defaultReleaseTickerDuration = 50 * time.Millisecond

	failedToSetVerdictMessage = "failed to set the nfqueue verdict for the packet"
)

// TODO rethink this objects name, i.e. is it a processor, a monitor (I keep referring to it as a monitor).

// DNSPolicyPacketProcessor listens for incoming nfqueue packets on a given channel and holds it until it receives a
// signal.
type DNSPolicyPacketProcessor struct {
	nf Nfqueue

	done chan struct{}

	packetDropTimeout     time.Duration
	packetReleaseTimeout  time.Duration
	releaseTickerDuration time.Duration

	packetReleaseChan chan []nfqueuePacket
	packetDropChan    chan []nfqueuePacket

	previouslyQueuedMark uint32

	timeManager timemanager.TimeManager
}

// nfqueuePacket represents a packet pulled off the nfqueue that's being monitored. It contains a subset of the
// information given to the monitor about the nfqueued packets to leave a smaller memory imprint.
type nfqueuePacket struct {
	// packetID is the ID used to set a verdict for the packet.
	packetID uint32

	// id is the ID set by the client, and used to uniquely identify packets repeatedly queued.
	id           uint16
	queuedTime   time.Time
	protocol     layers.IPProtocol
	srcIP, dstIP net.IP
	srcPort      uint16
	dstPort      uint16
}

func (packet *nfqueuePacket) idHash() string {
	h := sha1.New()
	identifiers := []interface{}{packet.id, packet.protocol, packet.srcIP, packet.dstIP, packet.srcPort, packet.dstPort}
	h.Write([]byte(fmt.Sprintf("%q", identifiers)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// logLinePrefix returns a string with information about the packet that should be used as a prefix to log lines that
// pertain to the packet.
func (packet *nfqueuePacket) logLinePrefix() string {
	return fmt.Sprintf("[packetID: %d, protocol: %d, srcIP: %s, dstIP: %s, srcPort: %d, dstPort: %d]",
		packet.id, packet.protocol, packet.srcIP, packet.dstIP, packet.srcPort, packet.dstPort)
}

func NewDNSPolicyPacketProcessor(nf Nfqueue, options ...Option) *DNSPolicyPacketProcessor {
	processor := &DNSPolicyPacketProcessor{
		nf:                    nf,
		done:                  make(chan struct{}),
		packetDropTimeout:     defaultPacketDropTimeout,
		packetReleaseTimeout:  defaultPacketReleaseTimeout,
		releaseTickerDuration: defaultReleaseTickerDuration,
		packetReleaseChan:     make(chan []nfqueuePacket, 100),
		packetDropChan:        make(chan []nfqueuePacket, 100),
		timeManager:           timemanager.New(),
	}

	for _, option := range options {
		option(processor)
	}

	return processor
}

func (processor *DNSPolicyPacketProcessor) Start() {
	processor.timeManager.Start()

	go processor.listenForIncomingPackets()

	// We create separate release and drop loops so we don't block processing packets while we set the verdicts.
	go processor.loopReleasingPackets()
	go processor.loopDroppingPackets()
}

func (processor *DNSPolicyPacketProcessor) Stop() {
	close(processor.done)

	processor.timeManager.Stop()
}

func (processor *DNSPolicyPacketProcessor) listenForIncomingPackets() {
	// TODO rename this, since we have a packetsToDrop (why does this get called packets)
	packets := make([]nfqueuePacket, 0, 100)
	packetsToDrop := make([]nfqueuePacket, 0, 100)

	ticker := time.NewTicker(processor.releaseTickerDuration)

	defer ticker.Stop()
	defer close(processor.packetReleaseChan)
	defer close(processor.packetDropChan)

done:
	for {
		select {
		case <-processor.done:
			break done
		case attr, ok := <-processor.nf.PacketAttributesChannel():
			if !ok {
				log.Info("Received signal to exit packet monitoring loop.")
				break done
			}

			packet := nfqueuePacket{
				packetID:   *attr.PacketID,
				queuedTime: time.Now(),
			}

			prometheusReleasePacketBatchSizeGauge.Set(float64(len(packets)))
			prometheusDropPacketBatchSizeGauge.Set(float64(len(packetsToDrop)))

			rawPacket := gopacket.NewPacket(*attr.Payload, layers.LayerTypeIPv4, gopacket.Lazy)
			ipv4, _ := rawPacket.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
			if ipv4 == nil {
				log.Debug(packet.logLinePrefix(), "Dropping non ipv4 packet.")

				packetsToDrop = append(packetsToDrop, packet)
				continue
			}

			packet.id = ipv4.Id
			packet.srcIP = ipv4.SrcIP
			packet.protocol = ipv4.Protocol
			packet.dstIP = ipv4.DstIP

			if tcp, _ := rawPacket.Layer(layers.LayerTypeTCP).(*layers.TCP); tcp != nil {
				packet.srcPort = uint16(tcp.SrcPort)
				packet.dstPort = uint16(tcp.DstPort)
			} else if udp, _ := rawPacket.Layer(layers.LayerTypeUDP).(*layers.UDP); udp != nil {
				packet.srcPort = uint16(udp.SrcPort)
				packet.dstPort = uint16(udp.DstPort)
			}

			// If there is a timestamp on the packet attempt to gather some metrics about how long it took the first
			// packet to get to this point.
			if !processor.timeManager.Exists(packet.idHash()) && attr.Timestamp != nil {
				prometheusNfqueueQueuedLatency.Observe(time.Since(*attr.Timestamp).Seconds())
			}

			timestamp := processor.timeManager.AddTime(packet.idHash())

			// If the packets time in the system has exceed the maximum allowed time then drop the packet.
			if time.Since(timestamp) >= processor.packetDropTimeout {
				log.Debug(packet.logLinePrefix(), "Dropping packet that's exceeded the timeout.")

				packetsToDrop = append(packetsToDrop, packet)
				continue
			}

			prometheusPacketsInCount.Inc()

			log.Debug(packet.logLinePrefix(), "Processing new packet.")
			packets = append(packets, packet)
		case <-ticker.C:
			packetsToRelease := make([]nfqueuePacket, 0, len(packets))
			packetsToHold := make([]nfqueuePacket, 0, len(packets))

			for _, packet := range packets {
				if time.Since(packet.queuedTime) >= processor.packetReleaseTimeout {
					packetsToRelease = append(packetsToRelease, packet)
				} else {
					packetsToHold = append(packetsToHold, packet)
				}
			}

			if len(packetsToRelease) > 0 {
				processor.packetReleaseChan <- packetsToRelease
			}

			if len(packetsToDrop) > 0 {
				processor.packetDropChan <- packetsToDrop
			}

			packets = packetsToHold
			packetsToDrop = make([]nfqueuePacket, 0, 500)
		}
	}
}

func (processor *DNSPolicyPacketProcessor) loopReleasingPackets() {
	for packets := range processor.packetReleaseChan {
		startTime := time.Now()
		for _, packet := range packets {
			log.Debug(packet.logLinePrefix(), "Releasing packet.")

			prometheusPacketsNfRepeatedCount.Inc()

			repeatSetVerdictOnFail(3, func() error {
				return processor.nf.SetVerdict(packet.packetID, gonfqueue.NfRepeat)
			}, packet.logLinePrefix(), failedToSetVerdictMessage)
		}
		prometheusPacketReleaseLatency.Observe(time.Since(startTime).Seconds())
	}
}

func (processor *DNSPolicyPacketProcessor) loopDroppingPackets() {
	for packets := range processor.packetDropChan {
		startTime := time.Now()
		for _, packet := range packets {
			log.Debug(packet.logLinePrefix(), "Dropping packet.")

			prometheusPacketsNfDroppedCount.Inc()

			repeatSetVerdictOnFail(3, func() error {
				return processor.nf.SetVerdict(packet.packetID, gonfqueue.NfDrop)
			}, packet.logLinePrefix(), "failed to set verdict for packet")
		}
		prometheusPacketDropLatency.Observe(time.Since(startTime).Seconds())
	}
}

// repeatSetVerdictOnFail repeats the given setVerdictFunc if it fails up to a maximum of numRepeats. If setting the
// verdict fails all attempts the error is logged with the failureMessages and the prometheusNfqueueVerdictFailCounter
// is incremented.
func repeatSetVerdictOnFail(numRepeats int, setVerdictFunc func() error, failureMessages ...string) {
	var err error
	for i := 0; i < numRepeats; i++ {
		err = setVerdictFunc()
		if err == nil {
			return
		}
	}

	PrometheusNfqueueVerdictFailCount.Inc()
	log.WithError(err).Error(failureMessages)
}
