// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package dnsdeniedpacket

import (
	"net"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/nfqueue"
	"github.com/projectcalico/calico/felix/timeshim"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	cprometheus "github.com/projectcalico/calico/libcalico-go/lib/prometheus"
	"github.com/projectcalico/calico/libcalico-go/lib/set"

	"github.com/prometheus/client_golang/prometheus"

	gonfqueue "github.com/florianl/go-nfqueue"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var (
	prometheusNfqueueQueuedLatency = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_policy_nfqueue_monitor_queued_latency",
		Help: "Summary of the length of time packets where in the nfqueue queue",
	})

	prometheusPacketReleaseLatency = cprometheus.NewSummary(prometheus.SummaryOpts{
		Name: "felix_dns_policy_nfqueue_monitor_release_latency",
		Help: "Summary of the latency for releasing packets",
	})

	prometheusReleasePacketBatchSizeGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_dns_policy_nfqueue_monitor_release_packets_batch_size",
		Help: "Gauge of the number of packets to release currently in memory",
	})

	prometheusPacketsInCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_packets_in",
		Help: "Count of the number of packets seen",
	})

	prometheusPacketsReleaseCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_packets_released",
		Help: "Count of the number of packets that have been released",
	})

	prometheusDNRDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "felix_dns_policy_nfqueue_monitor_packets_dnr_dropped",
		Help: "Count of the number of packets that have been dropped because the DNR mark was present",
	})
)

func init() {
	prometheus.MustRegister(prometheusPacketReleaseLatency)
	prometheus.MustRegister(prometheusReleasePacketBatchSizeGauge)
	prometheus.MustRegister(prometheusPacketsInCount)
	prometheus.MustRegister(prometheusNfqueueQueuedLatency)
	prometheus.MustRegister(prometheusPacketsReleaseCount)
	prometheus.MustRegister(prometheusDNRDroppedCount)
}

const (
	// defaultPacketReleaseTimeout is the maximum length of time to hold a packet while waiting for an IP set update
	// containing the destination IP of the packet.
	defaultPacketReleaseTimeout  = 1000 * time.Millisecond
	defaultReleaseTickerDuration = 50 * time.Millisecond
	// defaultIPCacheDuration is the default maximum length of time to store ips in the cache before deleting them. This
	// defaults to 1 second because that is the minimum TTL for an IP in a DNS response.
	defaultIPCacheDuration = 1000 * time.Millisecond

	failedToSetVerdictMessage = "failed to set the nfqueue verdict for the packet"

	// The maximum number of items to subsequently read from a channel.
	channelPeakLimit = 100
)

type PacketProcessor interface {
	Start()
	Stop()
	OnIPSetMemberUpdates(ips set.Set)
}

// packetProcessor listens for incoming nfqueue packets on a given channel and holds it until it receives a signal to
// release the packet. For more details on the inner workings, look at the comments for the loopProcessPackets
// function.
type packetProcessor struct {
	nf nfqueue.Nfqueue

	stopOnce sync.Once

	dnrMark uint32

	// Wait group that will wait for the loopProcessPackets function to return.
	loopProcessPacketsWG sync.WaitGroup

	// Various timeout durations
	packetReleaseTimeout  time.Duration
	releaseTickerDuration time.Duration
	ipCacheDuration       time.Duration

	// Channels used to communicate with the process loop
	ipsetMemberUpdates       chan set.Set
	packetReleaseChan        chan []nfqueuePacket
	stopLoopPacketProcessing chan struct{}

	// Arrays and maps used to store packets and IPs
	dstIPToPacketMap map[string][]nfqueuePacket
	packetsToRelease []nfqueuePacket
	ipCache          map[string]time.Time

	time timeshim.Interface

	rateLimitedErrLogger *logutils.RateLimitedLogger
}

// nfqueuePacket represents a packet pulled off the nfqueue that's being monitored. It contains a subset of the
// information given to the monitor about the nfqueued packets to leave a smaller memory imprint.
type nfqueuePacket struct {
	// packetID is the ID used to set a verdict for the packet.
	packetID uint32

	queuedTime   time.Time
	protocol     layers.IPProtocol
	srcIP, dstIP net.IP
	srcPort      uint16
	dstPort      uint16
}

// logLinePrefix returns a string with information about the packet that should be used as a prefix to log lines that
// pertain to the packet.
func (packet *nfqueuePacket) logLinePrefix() string {
	// Don't use Sprintf as this might be called many times.
	return "[protocol: " + strconv.Itoa(int(packet.protocol)) + ", srcIP: " + string(packet.srcIP) + ", dstIP: " +
		string(packet.dstIP) + ", srcPort: " + strconv.Itoa(int(packet.srcPort)) + ", dstPort: " +
		strconv.Itoa(int(packet.dstPort)) + "] "
}

func NewPacketProcessor(nf nfqueue.Nfqueue, dnrMark uint32, options ...Option) PacketProcessor {
	processor := &packetProcessor{
		nf:                       nf,
		dnrMark:                  dnrMark,
		packetReleaseTimeout:     defaultPacketReleaseTimeout,
		releaseTickerDuration:    defaultReleaseTickerDuration,
		ipCacheDuration:          defaultIPCacheDuration,
		packetReleaseChan:        make(chan []nfqueuePacket, 100),
		ipsetMemberUpdates:       make(chan set.Set, 100),
		stopLoopPacketProcessing: make(chan struct{}),
		packetsToRelease:         make([]nfqueuePacket, 0, 100),
		ipCache:                  make(map[string]time.Time),
		dstIPToPacketMap:         make(map[string][]nfqueuePacket),
		time:                     timeshim.RealTime(),
		rateLimitedErrLogger:     logutils.NewRateLimitedLogger(logutils.OptInterval(15 * time.Second)),
	}

	for _, option := range options {
		option(processor)
	}

	return processor
}

func (processor *packetProcessor) Start() {
	processor.loopProcessPacketsWG.Add(1)
	go processor.loopProcessPackets()
	go loopReleasePackets(processor.nf, processor.packetReleaseChan, processor.dnrMark, processor.time, processor.rateLimitedErrLogger)
}

func (processor *packetProcessor) Stop() {
	processor.stopOnce.Do(func() {
		close(processor.stopLoopPacketProcessing)

		processor.loopProcessPacketsWG.Wait()

		close(processor.packetReleaseChan)
		close(processor.ipsetMemberUpdates)
	})
}

// loopProcessPackets receives and holds packets from the monitored NFQUEUE and "releases" them when:
// - An update from the ipsetMemberUpdates channel arrives containing the destination IP of the packet
// - The packet has been held for a maximum amount of time (specified by the packetReleaseTimeout field of the
//   PacketProcessor)
//
// Releasing a packet entails removing it from the internal storage and setting a netfilter verdict. Setting a verdict
// tells iptables to continue processing the packet. The verdicts the PacketProcessor will set are:
// - NF_REPEAT with a DNR (do not repeat) mark bit if the packet is being "released"
// - NF_DROP if a packet is received from NFQUEUE with the DNR mark bit set (in the case the iptables NFQUEUE rule does
//   does not have a negative match against the DNR bit)
//
// The first case is the mainline case, where the packet is repeated through iptables because:
// - The IP we've been waiting to be programmed to ipsets has been programmed (and we were notified of this through the
//   ipsetMemberUpdates channel)
// - The IP we've been waiting to be programmed has not been programmed within the time limit and we now want iptables
//   to continue with the next processing steps for the packet (likely dropping the packet)
//
// The DNR mark bit ensures that any packet that has been NFQUEUE'd is not NFQUEUE'd again (this should be enforced
// by a negative match against the DNR mark bit in the iptables rule that NFQUEUEs the packet). This means that in the
// case that a packet is released because it times out it will not be re NFQUEUE'd, and will be evaluated by the
// iptables rules following the NFQUEUE rule.
func (processor *packetProcessor) loopProcessPackets() {
	defer processor.loopProcessPacketsWG.Done()

	releaseTicker := processor.time.NewTicker(processor.releaseTickerDuration)
	defer releaseTicker.Stop()

done:
	for {
		select {
		case <-processor.stopLoopPacketProcessing:
			break done
		case memberUpdates := <-processor.ipsetMemberUpdates:
			if memberUpdates == nil {
				continue
			}

			ips := make([]string, 0, memberUpdates.Len())
			memberUpdates.Iter(func(i interface{}) error {
				ipAddr := i.(ip.Addr)
				ips = append(ips, ipAddr.String())

				processor.ipCache[ipAddr.String()] = processor.time.Now()

				// Check if we have any packets held with this IP as it's destination, then add them to the list of
				// packets to release and remove them from the current packet map.
				if packets, exists := processor.dstIPToPacketMap[ipAddr.String()]; exists {
					processor.packetsToRelease = append(processor.packetsToRelease, packets...)

					delete(processor.dstIPToPacketMap, ipAddr.String())
				}

				return nil
			})

			log.WithField("ips", ips).Debug("Received ipset member update.")
		case attr, ok := <-processor.nf.PacketAttributesChannel():
			// If ok is not true then the underlying nf has been shutdown and there is nothing more that can be done in
			// the packet processor. The owner of this packet processor should be monitoring the injected nf object for
			// shutdowns, and subsequently call Stop() on this packet processor when it detects an nf shutdown.
			if !ok {
				log.Info("Packet attribute channel closed, shutting down.")
				break done
			}

			// Process the first packet read off the channel.
			processor.processPacket(attr)

		processPacketsMsgLoop:
			// Now attempt to read more all the packets on the channel (up to a maximum limit) before breaking from this
			// case.
			for i := 0; i < channelPeakLimit; i++ {
				select {
				case attr, ok := <-processor.nf.PacketAttributesChannel():
					if !ok {
						log.Info("Packet attribute channel closed, shutting down.")
						break done
					}
					processor.processPacket(attr)
				default:
					break processPacketsMsgLoop
				}
			}
		case <-releaseTicker.Chan():
			// On every tick we check if any packets currently held in the dstIPToPacketMap have been held for the
			// maximum duration, and if they have then we add those packets to the packetsToRelease slice to release
			// the packet.
			newDstToPacketMap := map[string][]nfqueuePacket{}

			for dstIP, packets := range processor.dstIPToPacketMap {
				packetsToHold := make([]nfqueuePacket, 0, len(packets))
				for _, packet := range packets {
					if processor.time.Since(packet.queuedTime) >= processor.packetReleaseTimeout {
						log.Debug(packet.logLinePrefix(), "Packet expired, adding to release list.")

						processor.packetsToRelease = append(processor.packetsToRelease, packet)
					} else {
						packetsToHold = append(packetsToHold, packet)
					}
				}

				if len(packetsToHold) > 0 {
					newDstToPacketMap[dstIP] = packetsToHold
				}
			}

			for cachedIP, cachedTime := range processor.ipCache {
				if processor.time.Since(cachedTime) >= processor.ipCacheDuration {
					delete(processor.ipCache, cachedIP)
				}
			}

			// If there's any packets to release then send them to the release loop via the packetReleaseChan.
			if len(processor.packetsToRelease) > 0 {
				processor.packetReleaseChan <- processor.packetsToRelease
			}

			processor.dstIPToPacketMap = newDstToPacketMap
			processor.packetsToRelease = make([]nfqueuePacket, 0, 500)
		}
	}
}

func (processor *packetProcessor) processPacket(attr gonfqueue.Attribute) {
	packet := nfqueuePacket{
		packetID:   *attr.PacketID,
		queuedTime: processor.time.Now(),
	}

	prometheusReleasePacketBatchSizeGauge.Set(float64(len(processor.packetsToRelease)))

	// This case protects against a packet looping forever in case the nfqueue rule is missing the negative match
	// against the dnr mark.
	if attr.Mark != nil && *attr.Mark&processor.dnrMark != 0x0 {
		processor.rateLimitedErrLogger.Error(packet.logLinePrefix(), "dropping packet with do not repeat mark.")

		repeatSetVerdictOnFail(3, func() error {
			return processor.nf.SetVerdict(packet.packetID, gonfqueue.NfDrop)
		}, processor.rateLimitedErrLogger, packet.logLinePrefix(), failedToSetVerdictMessage)

		prometheusDNRDroppedCount.Inc()
		return
	}

	rawPacket := gopacket.NewPacket(*attr.Payload, layers.LayerTypeIPv4, gopacket.Default)
	if rawPacket.ErrorLayer() != nil {
		rawPacket = gopacket.NewPacket(*attr.Payload, layers.LayerTypeIPv6, gopacket.Default)
		if rawPacket.ErrorLayer() != nil {
			processor.rateLimitedErrLogger.Error(packet.logLinePrefix(), "releasing unknown packet type (neither ipv4 nor ipv6)")
			processor.packetsToRelease = append(processor.packetsToRelease, packet)
			return
		}
	}

	switch rawPacket.NetworkLayer().LayerType() {
	case layers.LayerTypeIPv4:
		ipv4 := rawPacket.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

		packet.srcIP = ipv4.SrcIP
		packet.dstIP = ipv4.DstIP
	case layers.LayerTypeIPv6:
		ipv6 := rawPacket.Layer(layers.LayerTypeIPv6).(*layers.IPv6)

		packet.srcIP = ipv6.SrcIP
		packet.dstIP = ipv6.DstIP
	default:
		processor.rateLimitedErrLogger.Error(packet.logLinePrefix(), "releasing unknown packet type (neither ipv4 nor ipv6)")
		processor.packetsToRelease = append(processor.packetsToRelease, packet)
		return
	}

	transportLayer := rawPacket.TransportLayer()
	// Attempt to the get transport layer but don't fail if an error occurs. We don't actually need to know
	// any transport layer information, it's used for logging and stats.
	if transportLayer != nil {
		switch rawPacket.TransportLayer().LayerType() {
		case layers.LayerTypeTCP:
			tcp := rawPacket.Layer(layers.LayerTypeTCP).(*layers.TCP)

			packet.srcPort = uint16(tcp.SrcPort)
			packet.dstPort = uint16(tcp.DstPort)
			packet.protocol = layers.IPProtocolTCP
		case layers.LayerTypeUDP:
			udp := rawPacket.Layer(layers.LayerTypeUDP).(*layers.UDP)
			packet.srcPort = uint16(udp.SrcPort)
			packet.dstPort = uint16(udp.DstPort)
			packet.protocol = layers.IPProtocolUDP
		default:
			log.Debug(packet.logLinePrefix(), "Unknown transport layer type")
		}
	} else {
		log.Debug(packet.logLinePrefix(), "No transport layer type")
	}

	prometheusPacketsInCount.Inc()

	log.Debug(packet.logLinePrefix(), "Processing new packet.")

	// If there is a timestamp on the packet attempt to gather some metrics about how long it took the first
	// packet to get to this point.
	if attr.Timestamp != nil {
		prometheusNfqueueQueuedLatency.Observe(processor.time.Since(*attr.Timestamp).Seconds())
	}

	// If the destination IP is in the IP cache then we received the IP set member update just before the packet so we
	// just release the packet immediately.
	if _, exists := processor.ipCache[packet.dstIP.String()]; exists {
		log.Debug(packet.logLinePrefix(), "Processing new packet.")
		processor.packetsToRelease = append(processor.packetsToRelease, packet)
		return
	}

	if _, exists := processor.dstIPToPacketMap[packet.dstIP.String()]; !exists {
		processor.dstIPToPacketMap[packet.dstIP.String()] = []nfqueuePacket{}
	}

	processor.dstIPToPacketMap[packet.dstIP.String()] = append(processor.dstIPToPacketMap[packet.dstIP.String()], packet)
}

// OnIPSetMemberUpdates accepts a set of IPs which the packetProcessor uses to decide what, if any, packets should be
// released from the packetProcessor. The set of IPs are sent to the processing loop in loopProcessPackets. For more
// details on what happens with the IPs look at the comments for loopProcessPackets.
//
// Note that OnIPSetMemberUpdates must not be called after Stop() has been called on the packetProcessor. If this is
// a problem, consider looking at the packetProcessorWithNfqueueRestarter or wrapping this implementation with your
// own if that doesn't suite your needs.
//
// Note that the given set may be modified so it should not be used after calling this function
func (processor *packetProcessor) OnIPSetMemberUpdates(newMemberUpdates set.Set) {
	// This loop attempts to send newMemberUpdates over the ipsetMemberUpdates. If there is already a set on that
	// channel, then we pop it off and add those set members to newMemberUpdates and try to send newMemberUpdates
	// over the ipsetMemberUpdates on the next iteration. This technique stops us from blocking because the main
	// processing loop is taking awhile to process what it has and is not reading what's on the channel.
	for {
		select {
		case currentMemberUpdates := <-processor.ipsetMemberUpdates:
			currentMemberUpdates.Iter(func(ip interface{}) error {
				newMemberUpdates.Add(ip)
				return nil
			})
		case processor.ipsetMemberUpdates <- newMemberUpdates:
			return
		}
	}

}

// loopReleasePackets waits to receive batches of packets on the processor.packetReleaseChan and sets the nfqueue
// verdict for those packets, "releasing" them from the PacketProcessor.
//
// Note that this is intentionally not a function of the PacketProcessor. This is because there are attributes in the
// PacketProcessor that would not be safe for this function to access, as it should be run in it's own routine.
func loopReleasePackets(nf nfqueue.Nfqueue, packetReleaseChan chan []nfqueuePacket, dnrMark uint32,
	time timeshim.Interface, logger *logutils.RateLimitedLogger) {
	for packets := range packetReleaseChan {
		startTime := time.Now()
		for _, packet := range packets {
			logger.Debug(packet.logLinePrefix(), "Releasing packet.")

			prometheusPacketsReleaseCount.Inc()

			repeatSetVerdictOnFail(3, func() error {
				return nf.SetVerdictWithMark(packet.packetID, gonfqueue.NfRepeat, int(dnrMark))
			}, logger, packet.logLinePrefix(), "failed to set verdict for packet")
		}
		prometheusPacketReleaseLatency.Observe(time.Since(startTime).Seconds())
	}
}

// repeatSetVerdictOnFail repeats the given setVerdictFunc if it fails up to a maximum of numRepeats. If setting the
// verdict fails all attempts the error is logged with the failureMessages and the prometheusNfqueueVerdictFailCounter
// is incremented.
func repeatSetVerdictOnFail(numRepeats int, setVerdictFunc func() error, logger *logutils.RateLimitedLogger, failureMessages ...string) {
	var err error
	for i := 0; i < numRepeats; i++ {
		err = setVerdictFunc()
		if err == nil {
			return
		}
	}

	nfqueue.PrometheusNfqueueVerdictFailCount.Inc()
	logger.WithError(err).Error(failureMessages)
}
