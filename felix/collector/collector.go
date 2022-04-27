// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gavv/monotime"
	"github.com/google/gopacket/layers"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/ip"
	"github.com/projectcalico/calico/felix/jitter"
	logutil "github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/rules"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"

	kapiv1 "k8s.io/api/core/v1"
)

const (
	expectedLocalEither byte = iota
	expectedLocalDestination
	expectedLocalSource
)

var (
	// conntrack processing prometheus metrics
	histogramConntrackLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "felix_collector_conntrack_processing_latency_seconds",
		Help: "Histogram for measuring the latency of Conntrack processing.",
	})

	// TODO: find a way to track errors for conntrack processing as there are no
	//indicative method to track errors currently

	// process info processing prometheus metrics
	histogramPacketInfoLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "felix_collector_packet_info_processing_latency_seconds",
		Help: "Histogram for measuring latency Process Info processing.",
	})

	// TODO: find a way to track errors for process info processing as there are no
	//indicative method to track errors currently

	// dumpStats processing prometheus metrics
	histogramDumpStatsLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "felix_collector_dumpstats_latency_seconds",
		Help: "Histogram for measuring latency for processing cached stats to stats file in config.StatsDumpFilePath.",
	})

	// TODO: find a way to track errors for epStats dump processing as there are no
	//indicative method to track errors currently

	// dataplaneStatsUpdate processing prometheus metrics
	histogramDataplaneStatsUpdate = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "felix_collector_dataplanestats_update_processing_latency_seconds",
		Help: "Histogram for measuring latency for processing merging the proto.DataplaneStatistics to the current data cache.",
	})

	gaugeDataplaneStatsUpdateErrorsPerMinute = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_collector_dataplanestats_update_processing_errors_per_minute",
		Help: "Number of errors encountered when processing merging the proto.DataplaneStatistics to the current data cache.",
	})

	dataplaneStatsUpdateLastErrorReportTime time.Time
	dataplaneStatsUpdateErrorsInLastMinute  uint32

	// epStats cache prometheus metrics
	gaugeEpStatsCacheSizeLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "felix_collector_epstats",
		Help: "Total number of entries currently residing in the epStats cache.",
	})
)

func init() {
	prometheus.MustRegister(histogramConntrackLatency)
	prometheus.MustRegister(histogramPacketInfoLatency)
	prometheus.MustRegister(histogramDumpStatsLatency)
	prometheus.MustRegister(gaugeEpStatsCacheSizeLength)
	prometheus.MustRegister(histogramDataplaneStatsUpdate)
	prometheus.MustRegister(gaugeDataplaneStatsUpdateErrorsPerMinute)

}

type Config struct {
	StatsDumpFilePath string

	AgeTimeout            time.Duration
	InitialReportingDelay time.Duration
	ExportingInterval     time.Duration
	EnableNetworkSets     bool
	EnableServices        bool

	MaxOriginalSourceIPsIncluded int
	IsBPFDataplane               bool
}

// A collector (a StatsManager really) collects StatUpdates from data sources
// and stores them as a Data object in a map keyed by Tuple.
// All data source channels must be specified when creating the
//
// Note that the dataplane statistics channel (ds) is currently just used for the
// policy syncer but will eventually also include NFLOG stats as well.
type collector struct {
	packetInfoReader      PacketInfoReader
	conntrackInfoReader   ConntrackInfoReader
	processInfoCache      ProcessInfoCache
	domainLookup          EgressDomainCache
	luc                   *calc.LookupsCache
	epStats               map[Tuple]*Data
	ticker                jitter.JitterTicker
	sigChan               chan os.Signal
	config                *Config
	dumpLog               *log.Logger
	reporterMgr           *ReporterManager
	ds                    chan *proto.DataplaneStats
	dnsLogReporter        DNSLogReporterInterface
	l7LogReporter         L7LogReporterInterface
	displayDebugTraceLogs bool
}

// newCollector instantiates a new collector. The StartDataplaneStatsCollector function is the only public
// function for collector instantiation.
func newCollector(lc *calc.LookupsCache, rm *ReporterManager, cfg *Config) Collector {
	displayTraceLogs := false
	if rm != nil {
		displayTraceLogs = rm.displayDebugTraceLogs
	}

	return &collector{
		luc:                   lc,
		epStats:               make(map[Tuple]*Data),
		ticker:                jitter.NewTicker(cfg.ExportingInterval, cfg.ExportingInterval/10),
		sigChan:               make(chan os.Signal, 1),
		config:                cfg,
		dumpLog:               log.New(),
		reporterMgr:           rm,
		ds:                    make(chan *proto.DataplaneStats, 1000),
		displayDebugTraceLogs: displayTraceLogs,
	}
}

// ReportingChannel returns the channel used to report dataplane statistics.
func (c *collector) ReportingChannel() chan<- *proto.DataplaneStats {
	return c.ds
}

func (c *collector) Start() error {
	// The packet and conntrack info readers are essential for flow logs, but it still makes
	// sense for the collector to start without them, in order to handle DNS logs.
	if c.packetInfoReader == nil {
		log.Warning("missing PacketInfoReader")
	} else if err := c.packetInfoReader.Start(); err != nil {
		return fmt.Errorf("PacketInfoReader failed to start: %w", err)
	}
	if c.conntrackInfoReader == nil {
		log.Warning("missing ConntrackInfoReader")
	} else if err := c.conntrackInfoReader.Start(); err != nil {
		return fmt.Errorf("ConntrackInfoReader failed to start: %w", err)
	}

	if c.processInfoCache == nil {
		c.processInfoCache = NewNilProcessInfoCache()
	}
	if err := c.processInfoCache.Start(); err != nil {
		return fmt.Errorf("ProcessInfoCache failed to start: %w", err)
	}

	go c.startStatsCollectionAndReporting()
	c.setupStatsDumping()

	if c.dnsLogReporter != nil {
		c.dnsLogReporter.Start()
	}

	if c.l7LogReporter != nil {
		c.l7LogReporter.Start()
	}

	// init prometheus metrics timings
	dataplaneStatsUpdateLastErrorReportTime = time.Now()

	return nil
}

func (c *collector) SetPacketInfoReader(pir PacketInfoReader) {
	c.packetInfoReader = pir
}

func (c *collector) SetConntrackInfoReader(cir ConntrackInfoReader) {
	c.conntrackInfoReader = cir
}

func (c *collector) SetProcessInfoCache(pic ProcessInfoCache) {
	c.processInfoCache = pic
}

func (c *collector) SetDomainLookup(dlu EgressDomainCache) {
	c.domainLookup = dlu
}

func (c *collector) startStatsCollectionAndReporting() {
	var pktInfoC <-chan PacketInfo
	var ctInfoC <-chan []ConntrackInfo

	if c.packetInfoReader != nil {
		pktInfoC = c.packetInfoReader.PacketInfoChan()
	}
	if c.conntrackInfoReader != nil {
		ctInfoC = c.conntrackInfoReader.ConntrackInfoChan()
	}

	// When a collector is started, we respond to the following events:
	// 1. StatUpdates for incoming datasources (chan c.mux).
	// 2. A signal handler that will dump logs on receiving SIGUSR2.
	// 3. A done channel for stopping and cleaning up collector (TODO).
	for {
		select {
		case ctInfos := <-ctInfoC:
			conntrackProcessStart := time.Now()
			for _, ctInfo := range ctInfos {
				logutil.Tracef(c.displayDebugTraceLogs, "Collector event: %v", ctInfo)
				c.handleCtInfo(ctInfo)
			}
			histogramConntrackLatency.Observe(float64(time.Since(conntrackProcessStart).Seconds()))
		case pktInfo := <-pktInfoC:
			log.WithField("PacketInfo", pktInfo).Debug("collector event")
			processInfoProcessSttart := time.Now()
			c.applyPacketInfo(pktInfo)
			histogramPacketInfoLatency.Observe(float64(time.Since(processInfoProcessSttart).Seconds()))
		case <-c.ticker.Channel():
			c.checkEpStats()
		case <-c.sigChan:
			dumpStatsProcessStart := time.Now()
			c.dumpStats()
			histogramDumpStatsLatency.Observe(float64(time.Since(dumpStatsProcessStart).Seconds()))
		case ds := <-c.ds:
			dataplaneStatsUpdateStart := time.Now()
			c.convertDataplaneStatsAndApplyUpdate(ds)
			histogramDataplaneStatsUpdate.Observe(float64(time.Since(dataplaneStatsUpdateStart).Seconds()))
		}
	}
}

// getDataAndUpdateEndpoints returns a pointer to the data structure keyed off the supplied tuple.  If there
// is no entry and the tuple is for an active flow then an entry is created.
//
// This may return nil if the endpoint data does not match up with the requested data type.
//
// This method also updates the endpoint data from the cache, so beware - it is not as lightweight as a
// simple map lookup.
func (c *collector) getDataAndUpdateEndpoints(tuple Tuple, expired bool, packetinfo bool) *Data {
	data, okData := c.epStats[tuple]
	if expired {
		// If the connection has expired then return the data as is. If there is no entry, that's fine too.
		return data
	}

	// Get the source endpoint.
	srcEp := c.lookupEndpoint(tuple.src, false)
	srcEpIsNotLocal := srcEp == nil || !srcEp.IsLocal

	// Get the destination endpoint. If the source is local then we can use egress domain lookups if required.
	dstEp := c.lookupEndpoint(tuple.dst, !srcEpIsNotLocal)
	dstEpIsNotLocal := dstEp == nil || !dstEp.IsLocal

	if !okData {
		// For new entries, check that at least one of the endpoints is local.
		if srcEpIsNotLocal && dstEpIsNotLocal {
			return nil
		}

		// The entry does not exist. Go ahead and create a new one and add it to the map.
		data = NewData(tuple, srcEp, dstEp, c.config.MaxOriginalSourceIPsIncluded)
		c.updateEpStatsCache(tuple, data)

		// Return the new entry.
		return data
	}

	if data.reported {
		// Data has been reported.  If the request has not come from a packet info update (e.g. nflog) then the
		// endpoint data should be considered frozen since this is a pre-existing connection.
		if !packetinfo {
			return data
		}

		// If the endpoints have changed then we'll need to expire the current data and possibly delete the entry if
		// if not longer represents local endpoints.
		if endpointChanged(data.srcEp, srcEp) || endpointChanged(data.dstEp, dstEp) {
			// The endpoint information has now changed. Handle the endpoint changes.
			c.handleDataEndpointOrRulesChanged(data)

			// For updated entries, check that at least one of the endpoints is still local. If not delete the entry.
			if srcEpIsNotLocal && dstEpIsNotLocal {
				c.deleteDataFromEpStats(data)
				return nil
			}
		}

		// Update the source and dest data. We do this even if the endpoints haven't changed because the labels on the
		// endpoints may have changed and so our matches might be different.
		data.srcEp, data.dstEp = srcEp, dstEp

		return data
	}

	// Data has not been reported. Don't downgrade found endpoints (in case the endpoint is deleted prior to being
	// reported).
	if srcEp != nil {
		data.srcEp = srcEp
	}
	if dstEp != nil {
		data.dstEp = dstEp
	}

	return data
}

// endpointChanged determines if the endpoint has changed.
func endpointChanged(ep1, ep2 *calc.EndpointData) bool {
	if ep1 == ep2 {
		return false
	} else if ep1 == nil {
		return ep2 != nil
	} else if ep2 == nil {
		return true
	}
	return ep1.Key != ep2.Key
}

func (c *collector) lookupEndpoint(ip [16]byte, canCheckEgressDomains bool) *calc.EndpointData {
	// Get the endpoint data for this entry.
	if ep, ok := c.luc.GetEndpoint(ip); ok {
		return ep
	}

	// No matching endpoint. If NetworkSets are enabled for flows then check if the IP matches a NetworkSet and
	// return that.
	if c.config.EnableNetworkSets {
		if ep, ok := c.luc.GetNetworkSet(ip); ok {
			return ep
		}

		// NetworkSets endpoint types are enabled, but no NetworkSet matches the IP.  If canCheckEgressDomains is true
		// then we can also check for egress domain and lookup NetworkSet from that.
		if canCheckEgressDomains && c.domainLookup != nil {
			if domain := c.domainLookup.GetWatchedDomainForIP(ip); domain != "" {
				if ep, ok := c.luc.GetNetworkSetFromEgressDomain(domain); ok {
					return ep
				}
			}
		}
	}
	return nil
}

// updateEpStatsCache updates/add entry to the epStats cache (map[Tuple]*Data) and update the
// prometheus reporting
func (c *collector) updateEpStatsCache(tuple Tuple, data *Data) {
	c.epStats[tuple] = data

	c.reportEpStatsCacheMetrics()
}

// reportEpStatsCacheMetrics reports of current epStats cache status to Prometheus
func (c *collector) reportEpStatsCacheMetrics() {
	gaugeEpStatsCacheSizeLength.Set(float64(len(c.epStats)))
}

// applyConntrackStatUpdate applies a stats update from a conn track poll.
// If entryExpired is set then, this means that the update is for a recently
// expired entry. One of the following will be done:
// - If we already track the tuple, then the stats will be updated and will
//   then be expired from epStats.
// - If we don't track the tuple, this call will be a no-op as this update
//   is just waiting for the conntrack entry to timeout.
func (c *collector) applyConntrackStatUpdate(
	data *Data, packets int, bytes int, reversePackets int, reverseBytes int, entryExpired bool,
) {
	if data != nil {
		data.SetConntrackCounters(packets, bytes)
		data.SetConntrackCountersReverse(reversePackets, reverseBytes)

		if entryExpired {
			// The connection has expired. if the metrics can be reported then report and expire them now.
			// Otherwise, flag as expired and allow the export timer to process the connection - this allows additional
			// time for asynchronous meta data to be gathered (such as service info and process info).
			if c.reportMetrics(data, false) {
				c.expireMetrics(data)
				c.deleteDataFromEpStats(data)
			} else {
				data.SetExpired()
			}
		}
	}
}

// applyNflogStatUpdate applies a stats update from an NFLOG.
func (c *collector) applyNflogStatUpdate(data *Data, ruleID *calc.RuleID, matchIdx, numPkts, numBytes int) {
	if ru := data.AddRuleID(ruleID, matchIdx, numPkts, numBytes); ru == RuleMatchIsDifferent {
		c.handleDataEndpointOrRulesChanged(data)
		data.ReplaceRuleID(ruleID, matchIdx, numPkts, numBytes)
	}
}

func (c *collector) handleDataEndpointOrRulesChanged(data *Data) {
	// The endpoints or rule matched have changed. If reported then expire the metrics and update the
	// endpoint data.
	if c.reportMetrics(data, false) {
		// We only need to expire metric entries that've probably been reported
		// in the first place.
		c.expireMetrics(data)

		// Reset counters and replace the rule.
		data.ResetConntrackCounters()
		data.ResetApplicationCounters()
		data.ResetTcpStats()
		// Set reported to false so the data can be updated without further reports.
		data.reported = false
	}
}

func (c *collector) checkEpStats() {
	// We report stats at initial reporting delay after the last rule update. This aims to ensure we have the full set
	// of data before we report the stats. As a minor finesse, pre-calculate the latest update time to consider reporting.
	minLastRuleUpdatedAt := monotime.Now() - c.config.InitialReportingDelay
	minExpirationAt := monotime.Now() - c.config.AgeTimeout

	// For each entry
	// - report metrics.  Metrics reported through the ticker processing will wait for the initial reporting delay
	//   before reporting.  Note that this may be short-circuited by conntrack events or nflog events that inidicate
	//   the flow is terminated or has changed.
	// - check age and expire the entry if needed.
	for _, data := range c.epStats {
		if data.IsDirty() && (data.reported || data.RuleUpdatedAt() < minLastRuleUpdatedAt) {
			c.checkPreDNATTuple(data)
			c.reportMetrics(data, true)
		}
		if data.UpdatedAt() < minExpirationAt {
			c.expireMetrics(data)
			c.deleteDataFromEpStats(data)
		}
	}
}

func (c *collector) LookupProcessInfoCacheAndUpdate(data *Data) {
	t, _ := data.PreDNATTuple()
	processInfo, ok := c.processInfoCache.Lookup(t, TrafficDirOutbound)

	// In BPF dataplane, the existing connection tuples will be pre-DNAT and the new connections will
	// be post-DNAT, because of connecttime load balancer. Hence if the lookup with preDNAT tuple fails,
	// do a lookup with post DNAT tuple.
	if !ok && c.config.IsBPFDataplane {
		logutil.Tracef(c.displayDebugTraceLogs,
			"Lookup process cache for post DNAT tuple %+v for Outbound traffic", data.Tuple)
		processInfo, ok = c.processInfoCache.Lookup(data.Tuple, TrafficDirOutbound)
	}

	if ok {
		logutil.Tracef(c.displayDebugTraceLogs,
			"Setting source process name to %s and pid to %d for tuple %+v", processInfo.Name, processInfo.Pid, data.Tuple)
		if !data.reported && data.SourceProcessData().Name == "" && data.SourceProcessData().Pid == 0 {
			data.SetSourceProcessData(processInfo.Name, processInfo.Arguments, processInfo.Pid)
		}
		if processInfo.TcpStatsData.IsDirty {
			data.SetTcpSocketStats(processInfo.TcpStatsData)
			// Since we have read the data TCP stats data from the cache, set it to false
			c.processInfoCache.Update(t, false)
			logutil.Tracef(c.displayDebugTraceLogs,
				"Setting tcp stats to %+v for tuple %+v", processInfo.TcpStatsData, processInfo.Tuple)
		}
	}

	processInfo, ok = c.processInfoCache.Lookup(t, TrafficDirInbound)
	if !ok && c.config.IsBPFDataplane {
		logutil.Tracef(c.displayDebugTraceLogs,
			"Lookup process cache for post DNAT tuple %+v for Inbound traffic", data.Tuple)
		processInfo, ok = c.processInfoCache.Lookup(data.Tuple, TrafficDirInbound)
	}

	if ok {
		logutil.Tracef(c.displayDebugTraceLogs,
			"Setting dest process name to %s and pid to %d from reverse tuple %+v", processInfo.Name, processInfo.Pid, t.GetReverseTuple())
		if !data.reported && data.DestProcessData().Name == "" && data.DestProcessData().Pid == 0 {
			data.SetDestProcessData(processInfo.Name, processInfo.Arguments, processInfo.Pid)
		}
		if processInfo.TcpStatsData.IsDirty {
			data.SetTcpSocketStats(processInfo.TcpStatsData)
			// Since we have read the data TCP stats data from the cache, set it to false
			c.processInfoCache.Update(t, false)
			logutil.Tracef(c.displayDebugTraceLogs,
				"Setting tcp stats to %+v for tuple %+v", processInfo.TcpStatsData, processInfo.Tuple)
		}
	}
}

func (c *collector) checkPreDNATTuple(data *Data) {
	preDNATTuple, err := data.PreDNATTuple()
	if err != nil {
		return
	}
	preDNATData, ok := c.epStats[preDNATTuple]
	if !ok {
		return
	}
	log.Debugf("Found data that resembles PreDNAT connection data->%+v, preDNATData->%+v", data, preDNATData)

	// If we are tracking a denied connection attempt that has the same tuple as the
	// pre-DNAT tuple of a similar allowed connection then make sure that we expire the
	// tuple that looks like the pre-DNAT tuple. This guards us against a scenario where
	// the first tracked tuple that looks like the preDNAT tuple of a long running connection
	// never expires when TCP stats are enabled.
	// We don't worry about a allowed connection becoming denied because we don't currently
	// delete an already established connection.
	// We only try to report a metric when
	// - the tuple that looks like the pre DNAT tuple is not a connection i.e, we only received NFLOGs.
	// - Both ingress and egress rule trace is not dirty. Otherwise we want to let the usual report metrics
	//   go ahead first as this cleanup here is a last resort.
	if !preDNATData.isConnection && !preDNATData.EgressRuleTrace.IsDirty() && !preDNATData.IngressRuleTrace.IsDirty() {
		c.reportMetrics(preDNATData, true)
		c.expireMetrics(preDNATData)
		c.deleteDataFromEpStats(preDNATData)
	}
}

// reportMetrics reports the metrics if all required data is present, or returns false if not reported.
// Set the force flag to true if the data should be reported before all asynchronous data is collected.
func (c *collector) reportMetrics(data *Data, force bool) bool {
	foundService := true
	c.LookupProcessInfoCacheAndUpdate(data)

	if !data.reported {
		// Check if the destination was accessed via a service. Once reported, this will not be updated again.
		if data.dstSvc.Name == "" {
			if data.isDNAT {
				// Destination is NATed, look up service from the pre-DNAT record.
				data.dstSvc, foundService = c.luc.GetServiceFromPreDNATDest(data.preDNATAddr, data.preDNATPort, data.Tuple.proto)
			} else if _, ok := c.luc.GetNode(data.Tuple.dst); ok {
				// Destination is a node, so could be a node port service.
				data.dstSvc, foundService = c.luc.GetNodePortService(data.Tuple.l4Dst, data.Tuple.proto)
			}
		}
		if !force {
			// If not forcing then return if:
			// - There may be a service to report
			// - The verdict rules have not been found for the local endpoints
			// - The remote endpoint is not known (which could potentially resolve to a DNS name or NetworkSet).
			// In this case data will be reported later during ticker processing.
			if !foundService || !data.VerdictFound() || data.dstEp == nil {
				log.Debug("Service not found - delay statistics reporting until normal flush processing")
				return false
			}
		}
	}

	// Send the metrics.
	c.sendMetrics(data, false)
	data.reported = true
	return true
}

func (c *collector) expireMetrics(data *Data) {
	if data.reported {
		c.sendMetrics(data, true)
	}
}

func (c *collector) deleteDataFromEpStats(data *Data) {
	delete(c.epStats, data.Tuple)

	c.reportEpStatsCacheMetrics()
}

func (c *collector) sendMetrics(data *Data, expired bool) {
	ut := UpdateTypeReport
	if expired {
		ut = UpdateTypeExpire
	}
	// For connections and non-connections, we only send ingress and egress updates if:
	// -  There is something to report, i.e.
	//    -  flow is expired, or
	//    -  associated stats are dirty
	// -  The policy verdict rule has been determined. Note that for connections the policy verdict may be "Deny" due
	//    to DropActionOverride setting (e.g. if set to ALLOW, then we'll get connection stats, but the metrics will
	//    indicate Denied).
	// Only clear the associated stats and dirty flag once the metrics are reported.
	if data.isConnection {
		// Report connection stats.
		if expired || data.IsDirty() {
			// Track if we need to send a separate expire metric update. This is required when we are only
			// reporting Original IP metric updates and want to send a corresponding expiration metric update.
			// When they are correlated with regular metric updates and connection metrics, we don't need to
			// send this.
			sendOrigSourceIPsExpire := true
			if data.EgressRuleTrace.FoundVerdict() {
				c.reporterMgr.ReportChan <- data.metricUpdateEgressConn(ut)
			}
			if data.IngressRuleTrace.FoundVerdict() {
				sendOrigSourceIPsExpire = false
				c.reporterMgr.ReportChan <- data.metricUpdateIngressConn(ut)
			}

			// We may receive HTTP Request data after we've flushed the connection counters.
			if (expired && data.origSourceIPsActive && sendOrigSourceIPsExpire) || data.NumUniqueOriginalSourceIPs() != 0 {
				data.origSourceIPsActive = !expired
				c.reporterMgr.ReportChan <- data.metricUpdateOrigSourceIPs(ut)
			}

			// Clear the connection dirty flag once the stats have been reported. Note that we also clear the
			// rule trace stats here too since any data stored in them has been superceded by the connection
			// stats.
			data.clearConnDirtyFlag()
			data.EgressRuleTrace.ClearDirtyFlag()
			data.IngressRuleTrace.ClearDirtyFlag()
		}
	} else {
		// Report rule trace stats.
		if (expired || data.EgressRuleTrace.IsDirty()) && data.EgressRuleTrace.FoundVerdict() {
			c.reporterMgr.ReportChan <- data.metricUpdateEgressNoConn(ut)
			data.EgressRuleTrace.ClearDirtyFlag()
		}
		if (expired || data.IngressRuleTrace.IsDirty()) && data.IngressRuleTrace.FoundVerdict() {
			c.reporterMgr.ReportChan <- data.metricUpdateIngressNoConn(ut)
			data.IngressRuleTrace.ClearDirtyFlag()
		}

		// We do not need to clear the connection stats here. Connection stats are fully reset if the Data moves
		// from a connection to non-connection state.
	}
	data.TcpStats.ClearDirtyFlag()
}

// handleCtInfo handles an update from conntrack
// We expect and process connections (conntrack entries) of 3 different flavors.
//
// - Connections that *neither* begin *nor* terminate locally.
// - Connections that either begin or terminate locally.
// - Connections that begin *and* terminate locally.
//
// When processing these, we also check if the connection is flagged as a
// destination NAT (DNAT) connection. If it is a DNAT-ed connection, we
// process the conntrack entry after we figure out the DNAT-ed destination and port.
// This is important for services where the connection will have the cluster IP as the
// pre-DNAT-ed destination, but we want the post-DNAT workload IP and port.
// The pre-DNAT entry will also be used to lookup service related information.
func (c *collector) handleCtInfo(ctInfo ConntrackInfo) {
	// Get or create a data entry and update the counters. If no entry is returned then neither source nor dest are
	// calico managed endpoints. A relevant conntrack entry requires at least one of the endpoints to be a local
	// Calico managed endpoint.

	if data := c.getDataAndUpdateEndpoints(ctInfo.Tuple, ctInfo.Expired, false); data != nil {

		if !data.isDNAT && ctInfo.IsDNAT {
			originalTuple := ctInfo.PreDNATTuple
			data.isDNAT = true
			data.preDNATAddr = originalTuple.dst
			data.preDNATPort = originalTuple.l4Dst
		}
		data.isProxied = ctInfo.IsProxy
		data.natOutgoingPort = ctInfo.NatOutgoingPort

		c.applyConntrackStatUpdate(data,
			ctInfo.Counters.Packets, ctInfo.Counters.Bytes,
			ctInfo.ReplyCounters.Packets, ctInfo.ReplyCounters.Bytes,
			ctInfo.Expired)
	}
}

func (c *collector) applyPacketInfo(pktInfo PacketInfo) {
	var (
		localEp        *calc.EndpointData
		localMatchData *calc.MatchData
		data           *Data
	)

	tuple := pktInfo.Tuple

	if data = c.getDataAndUpdateEndpoints(tuple, false, true); data == nil {
		// Data is nil, so the destination endpoint cannot be managed by local Calico.
		return
	}

	if !data.isDNAT && pktInfo.IsDNAT {
		originalTuple := pktInfo.PreDNATTuple
		data.isDNAT = true
		data.preDNATAddr = originalTuple.dst
		data.preDNATPort = originalTuple.l4Dst
	}

	// Determine the local endpoint for this update.
	switch pktInfo.Direction {
	case rules.RuleDirIngress:
		// The local destination should be local.
		if localEp = data.dstEp; localEp == nil || !localEp.IsLocal {
			return
		}
		localMatchData = localEp.Ingress
	case rules.RuleDirEgress:
		// The cache will return nil for egress if the source endpoint is not local.
		if localEp = data.srcEp; localEp == nil || !localEp.IsLocal {
			return
		}
		localMatchData = localEp.Egress
	default:
		return
	}

	for _, rule := range pktInfo.RuleHits {
		ruleID := rule.RuleID
		if ruleID == nil {
			continue
		}
		if ruleID.IsProfile() {
			// This is a profile verdict. Apply the rule unchanged, but at the profile match index (which is at the
			// very end of the match slice).
			c.applyNflogStatUpdate(data, ruleID, localMatchData.ProfileMatchIndex, rule.Hits, rule.Bytes)
			continue
		}

		if ruleID.IsEndOfTier() {
			// This is an end-of-tier action.
			// -  For deny convert the ruleID to the implicit drop rule
			// -  For pass leave the rule unchanged. We never return this to the user, but instead use it to determine
			//    whether we add staged policy end-of-tier denies.
			// For both deny and pass, add the rule at the end of tier match index.
			tier, ok := localMatchData.TierData[ruleID.Tier]
			if !ok {
				continue
			}

			switch ruleID.Action {
			case rules.RuleActionDeny:
				c.applyNflogStatUpdate(
					data, tier.ImplicitDropRuleID, tier.EndOfTierMatchIndex,
					rule.Hits, rule.Bytes,
				)
			case rules.RuleActionPass:
				c.applyNflogStatUpdate(
					data, ruleID, tier.EndOfTierMatchIndex,
					rule.Hits, rule.Bytes,
				)
			}
			continue
		}

		// This is one of:
		// -  An enforced rule match
		// -  A staged policy match
		// -  A staged policy miss
		// -  An end-of-tier pass (from tiers only containing staged policies)
		//
		// For all these cases simply add the unchanged ruleID using the match index reserved for that policy.
		// Extract the policy data from the ruleID.
		policyIdx, ok := localMatchData.PolicyMatches[ruleID.PolicyID]
		if !ok {
			continue
		}

		c.applyNflogStatUpdate(data, ruleID, policyIdx, rule.Hits, rule.Bytes)
	}

	if data.IsExpired() && c.reportMetrics(data, false) {
		// If the data is expired then attempt to report it now so that we can remove the connection entry. If reported
		// the data can be expired and deleted immediately, otherwise it will get exported during ticker processing.
		c.expireMetrics(data)
		c.deleteDataFromEpStats(data)
	}
}

// convertDataplaneStatsAndApplyUpdate merges the proto.DataplaneStatistics into the current
// data stored for the specific connection tuple.
func (c *collector) convertDataplaneStatsAndApplyUpdate(d *proto.DataplaneStats) {
	logutil.Tracef(c.displayDebugTraceLogs, "Received dataplane stats update %+v", d)
	// Create a Tuple representing the DataplaneStats.
	t, err := extractTupleFromDataplaneStats(d)
	if err != nil {
		log.Errorf("unable to extract 5-tuple from DataplaneStats: %v", err)
		reportDataplaneStatsUpdateErrorMetrics(1)
		return
	}

	// Locate the data for this connection, creating if not yet available (it's possible to get an update
	// from the dataplane before nflogs or conntrack).
	data := c.getDataAndUpdateEndpoints(t, false, false)

	var httpDataCount int
	var isL7Data bool
	for _, s := range d.Stats {
		if s.Relativity != proto.Statistic_DELTA {
			// Currently we only expect delta HTTP requests from the dataplane statistics API.
			log.WithField("relativity", s.Relativity.String()).Warning("Received a statistic from the dataplane that Felix cannot process")
			continue
		}
		switch s.Kind {
		case proto.Statistic_HTTP_REQUESTS:
			switch s.Action {
			case proto.Action_ALLOWED:
				data.IncreaseHTTPRequestAllowedCounter(int(s.Value))
			case proto.Action_DENIED:
				data.IncreaseHTTPRequestDeniedCounter(int(s.Value))
			}
		case proto.Statistic_HTTP_DATA:
			httpDataCount = int(s.Value)
			isL7Data = true
		case proto.Statistic_INGRESS_DATA:
			httpDataCount = int(s.Value)
		default:
			log.WithField("kind", s.Kind.String()).Warnf("Received a statistic from the dataplane that Felix cannot process")
			continue
		}
	}

	ips := make([]net.IP, 0, len(d.HttpData))

	for _, hd := range d.HttpData {
		if c.l7LogReporter != nil && isL7Data {
			// If the l7LogReporter has been set, then L7 logs are configured to be run.
			c.LogL7(hd, data, t, httpDataCount)
		} else if !isL7Data {
			var origSrcIP string
			if len(hd.XRealIp) != 0 {
				origSrcIP = hd.XRealIp
			} else if len(hd.XForwardedFor) != 0 {
				origSrcIP = hd.XForwardedFor
			} else {
				continue
			}
			sip := net.ParseIP(origSrcIP)
			if sip == nil {
				log.WithField("IP", origSrcIP).Warn("bad source IP")
				continue
			}
			ips = append(ips, sip)
		}
	}

	// ips will only be set for original source IP data
	if len(ips) != 0 {
		if httpDataCount == 0 {
			httpDataCount = len(ips)
		}

		bs := NewBoundedSetFromSliceWithTotalCount(c.config.MaxOriginalSourceIPsIncluded, ips, httpDataCount)
		data.AddOriginalSourceIPs(bs)
	} else if httpDataCount != 0 && !isL7Data {
		data.IncreaseNumUniqueOriginalSourceIPs(httpDataCount)
	} else if httpDataCount != 0 && c.l7LogReporter != nil && len(d.HttpData) == 0 && isL7Data {
		// Record overflow L7 log counts
		// Overflow logs will have counts but no HttpData
		// Create an empty HTTPData since this is an overflow log
		hd := &proto.HTTPData{}
		c.LogL7(hd, data, t, httpDataCount)
	}
}

func extractTupleFromDataplaneStats(d *proto.DataplaneStats) (Tuple, error) {
	var protocol int32
	switch n := d.Protocol.GetNumberOrName().(type) {
	case *proto.Protocol_Number:
		protocol = n.Number
	case *proto.Protocol_Name:
		switch strings.ToLower(n.Name) {
		case "tcp":
			protocol = 6
		case "udp":
			protocol = 17
		default:
			reportDataplaneStatsUpdateErrorMetrics(1)
			return Tuple{}, fmt.Errorf("unhandled protocol: %s", n)
		}
	}

	// Use the standard go net library to parse the IP since this always returns IPs as 16 bytes.
	srcIP, ok := ip.ParseIPAs16Byte(d.SrcIp)
	if !ok {
		reportDataplaneStatsUpdateErrorMetrics(1)
		return Tuple{}, fmt.Errorf("bad source IP: %s", d.SrcIp)
	}
	dstIP, ok := ip.ParseIPAs16Byte(d.DstIp)
	if !ok {
		reportDataplaneStatsUpdateErrorMetrics(1)
		return Tuple{}, fmt.Errorf("bad destination IP: %s", d.DstIp)
	}

	// Locate the data for this connection, creating if not yet available (it's possible to get an update
	// before nflogs or conntrack).
	return MakeTuple(srcIP, dstIP, int(protocol), int(d.SrcPort), int(d.DstPort)), nil
}

// reportDataplaneStatsUpdateErrorMetrics reports error statistics encoutered when updating Dataplane stats
func reportDataplaneStatsUpdateErrorMetrics(dataplaneErrorDelta uint32) {

	if dataplaneStatsUpdateLastErrorReportTime.Before(time.Now().Add(-1 * time.Minute)) {
		dataplaneStatsUpdateErrorsInLastMinute = dataplaneErrorDelta
	} else {
		dataplaneStatsUpdateErrorsInLastMinute += dataplaneErrorDelta
	}

	dataplaneStatsUpdateErrorsInLastMinute += dataplaneErrorDelta
	gaugeDataplaneStatsUpdateErrorsPerMinute.Set(float64(dataplaneStatsUpdateErrorsInLastMinute))
}

// Write stats to file pointed by Config.StatsDumpFilePath.
// When called, clear the contents of the file Config.StatsDumpFilePath before
// writing the stats to it.
func (c *collector) dumpStats() {
	logutil.Tracef(c.displayDebugTraceLogs, "Dumping Stats to %v", c.config.StatsDumpFilePath)

	_ = os.Truncate(c.config.StatsDumpFilePath, 0)
	c.dumpLog.Infof("Stats Dump Started: %v", time.Now().Format("2006-01-02 15:04:05.000"))
	c.dumpLog.Infof("Number of Entries: %v", len(c.epStats))
	for _, v := range c.epStats {
		c.dumpLog.Info(fmtEntry(v))
	}
	c.dumpLog.Infof("Stats Dump Completed: %v", time.Now().Format("2006-01-02 15:04:05.000"))
}

func fmtEntry(data *Data) string {
	return fmt.Sprintf("%v", data)
}

// Logrus Formatter that strips the log entry of formatting such as time, log
// level and simply outputs *only* the message.
type MessageOnlyFormatter struct{}

func (f *MessageOnlyFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	b.WriteString(entry.Message)
	b.WriteByte('\n')
	return b.Bytes(), nil
}

// DNS activity logging.
func (c *collector) SetDNSLogReporter(reporter DNSLogReporterInterface) {
	c.dnsLogReporter = reporter
}

func (c *collector) LogDNS(server, client net.IP, dns *layers.DNS, latencyIfKnown *time.Duration) {
	if c.dnsLogReporter == nil {
		return
	}
	// DNS responses come through here, so the source IP is the DNS server and the dest IP is
	// the client.
	serverEP, _ := c.luc.GetEndpoint(ipTo16Byte(server))
	clientEP, _ := c.luc.GetEndpoint(ipTo16Byte(client))
	if serverEP == nil {
		serverEP, _ = c.luc.GetNetworkSet(ipTo16Byte(server))
	}
	logutil.Tracef(c.displayDebugTraceLogs, "Src %v -> Server %v", server, serverEP)
	logutil.Tracef(c.displayDebugTraceLogs, "Dst %v -> Client %v", client, clientEP)
	if latencyIfKnown != nil {
		logutil.Tracef(c.displayDebugTraceLogs, "DNS-LATENCY: Log %v", *latencyIfKnown)
	}
	update := DNSUpdate{
		ClientIP:       client,
		ClientEP:       clientEP,
		ServerIP:       server,
		ServerEP:       serverEP,
		DNS:            dns,
		LatencyIfKnown: latencyIfKnown,
	}
	if err := c.dnsLogReporter.Log(update); err != nil {
		if !strings.Contains(err.Error(), "No questions in DNS packet") {
			log.WithError(err).WithFields(log.Fields{
				"server": server,
				"client": client,
				"dns":    dns,
			}).Error("Failed to log DNS packet")
		}
	}
}

func (c *collector) SetL7LogReporter(reporter L7LogReporterInterface) {
	c.l7LogReporter = reporter
}

func (c *collector) LogL7(hd *proto.HTTPData, data *Data, tuple Tuple, httpDataCount int) {
	// Translate endpoint data into L7Update
	update := L7Update{
		Tuple:         tuple,
		Duration:      int(hd.Duration),
		DurationMax:   int(hd.DurationMax),
		BytesReceived: int(hd.BytesReceived),
		BytesSent:     int(hd.BytesSent),
		Method:        hd.RequestMethod,
		Path:          hd.RequestPath,
		UserAgent:     hd.UserAgent,
		Type:          hd.Type,
		Domain:        hd.Domain,
	}

	// If there is no endpoint data for the supplied tuple, skip logging this info.
	// This means that this log was collected for traffic that was forwarded through
	// this node (like traffic through a nodeport service).
	if data != nil {
		update.SrcEp = data.srcEp
		update.DstEp = data.dstEp
	}

	// Handle setting the response code. An empty response code is valid for overflow logs.
	if hd.ResponseCode != 0 {
		update.ResponseCode = strconv.Itoa(int(hd.ResponseCode))
	}

	// Handle setting the count for overflow logs
	if hd.Count != 0 {
		update.Count = int(hd.Count)
	} else {
		// overflow logs record the http request count per the tuple instead.
		update.Count = httpDataCount
	}

	// Grab the destination metadata to use the namespace to validate the service name
	dstMeta, err := getFlowLogEndpointMetadata(update.DstEp, tuple.dst)
	if err != nil {
		reportDataplaneStatsUpdateErrorMetrics(1)
		log.WithError(err).Errorf("Failed to extract metadata for destination %v", update.DstEp)
	}

	// Split out the service port if available
	addr, port := getAddressAndPort(hd.Domain)

	// Set default ports for specific log types.
	// Currently the only default ports we set are for http log types since
	// they are the only logs that capture service information.
	if strings.Contains(strings.ToLower(hd.Type), "http") && port == 0 {
		port = 80
	}

	var validService bool
	svcName := addr
	svcNamespace := dstMeta.Namespace

	// TODO: Find a separate way of retrieving this data when we do not
	// have endpoint data on this node.
	var svcPortName string
	if data != nil {
		svcPortName = data.dstSvc.Port
	}

	if ip := net.ParseIP(addr); ip != nil {
		// Address is an IP. Attempt to look up a service name by cluster IP
		k8sSvcPortName, found := c.luc.GetServiceFromPreDNATDest(ipStrTo16Byte(addr), port, tuple.proto)
		if found {
			svcName = k8sSvcPortName.NamespacedName.Name
			svcNamespace = k8sSvcPortName.NamespacedName.Namespace
			svcPortName = k8sSvcPortName.Port
			validService = true
		}
	} else {
		// Check if the address is a Kubernetes service name
		k8sSvcName, k8sSvcNamespace := extractK8sServiceNameAndNamespace(addr)
		if k8sSvcName != "" {
			svcName = k8sSvcName
			svcNamespace = k8sSvcNamespace
		}

		// Verify that the service name and namespace are valid
		serviceSpec, isValidService := c.luc.GetServiceSpecFromResourceKey(model.ResourceKey{
			Kind:      v3.KindK8sService,
			Name:      svcName,
			Namespace: svcNamespace,
		})
		validService = isValidService

		if isValidService {
			svcPortName = getPortNameFromServicSpec(serviceSpec, port)
		}
	}

	// Add the service name and port if they are available
	// The port may not have been specified. This will result in port being 0.
	if validService {
		update.ServiceName = svcName
		update.ServiceNamespace = svcNamespace
		update.ServicePortName = svcPortName
		update.ServicePortNum = port
	}

	// Send the update to the reporter
	if err := c.l7LogReporter.Log(update); err != nil {
		reportDataplaneStatsUpdateErrorMetrics(1)
		log.WithError(err).WithFields(log.Fields{
			"src": tuple.src,
			"dst": tuple.src,
		}).Error("Failed to log request")
	}
}

func getPortNameFromServicSpec(serviceSpec kapiv1.ServiceSpec, port int) string {
	for _, servicePort := range serviceSpec.Ports {
		if servicePort.Port == int32(port) {
			return servicePort.Name
		}
	}

	return ""
}

// NilProcessInfoCache implements the ProcessInfoCache interface and always returns false
// for lookups. It is used as a default implementation of a ProcessInfoCache when one is
// not explicitly set.
type NilProcessInfoCache struct{}

func NewNilProcessInfoCache() *NilProcessInfoCache {
	return &NilProcessInfoCache{}
}

func (r *NilProcessInfoCache) Lookup(tuple Tuple, dir TrafficDirection) (ProcessInfo, bool) {
	return ProcessInfo{}, false
}

func (r *NilProcessInfoCache) Start() error {
	return nil
}

func (r *NilProcessInfoCache) Stop() {}

func (r *NilProcessInfoCache) Update(tuple Tuple, dirty bool) {
}
