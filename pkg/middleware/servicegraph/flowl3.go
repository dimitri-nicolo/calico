// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/set"

	lmav1 "github.com/tigera/lma/pkg/apis/v1"
	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/elastic"
)

// This file provides the main interface into elasticsearch for service graph. It is used to load flows for a given
// time range, to correlate the source and destination flows and to aggregate out ports and protocols that are not
// accessed via a service. Where elasticsearch raw flow logs may contain separate source and destination flows, this
// will return a single flow with statistics for allowed, denied-at-source and denied-at-dest.

const (
	maxAggregatedProtocol              = 10
	maxAggregatedPortRangesPerProtocol = 5
)

const (
	flowsBucketName = "flows"
)

const (
	// The ordering of these composite sources is important. We want to enumerate all services across all sources for
	// a given destination, and we need to ensure:
	// - service (DNATed) flows are returned before non-service flows
	// - all sources are enumerated for a given destination port
	// This allows us to decide whether a destination port and protocol is being accessed by a service or not (and
	// remember it could be accessed via a service for one source and directly for another). If the endpoint+port+proto
	// is not accessed via a service then we'll aggregate the port and proto - this prevents things like port scans
	// from making the graph unreadable.
	FlowDestTypeIdx = iota
	FlowDestNamespaceIdx
	FlowDestNameAggrIdx
	FlowDestServiceNamespaceIdx
	FlowDestServiceNameIdx
	FlowDestServicePortIdx
	FlowProtoIdx
	FlowDestPortIdx
	FlowSourceTypeIdx
	FlowSourceNamespaceIdx
	FlowSourceNameAggrIdx
	FlowProcessIdx
	FlowReporterIdx
	FlowActionIdx

	FlowIdxMax
)

var (
	flowCompositeSources = []lmaelastic.AggCompositeSourceInfo{
		{Name: "dest_type", Field: "dest_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name_aggr", Field: "dest_name_aggr"},
		{Name: "dest_service_namespace", Field: "dest_service_namespace", Order: "desc"},
		{Name: "dest_service_name", Field: "dest_service_name"},
		{Name: "dest_service_port", Field: "dest_service_port"},
		{Name: "proto", Field: "proto"},
		{Name: "dest_port", Field: "dest_port"},
		{Name: "source_type", Field: "source_type"},
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "source_name_aggr", Field: "source_name_aggr"},
		{Name: "process_name", Field: "process_name"},
		{Name: "reporter", Field: "reporter"},
		{Name: "action", Field: "action"},
	}
	zeroGraphTCPStats = v1.GraphTCPStats{}
)

const (
	//TODO(rlb): We might want to abbreviate these to reduce the amount of data on the wire, json parsing and
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.  These
	//           could be anything really as long as each is unique.
	FlowAggSumNumFlows                 = "sum_num_flows"
	FlowAggSumNumFlowsStarted          = "sum_num_flows_started"
	FlowAggSumNumFlowsCompleted        = "sum_num_flows_completed"
	FlowAggSumPacketsIn                = "sum_packets_in"
	FlowAggSumBytesIn                  = "sum_bytes_in"
	FlowAggSumPacketsOut               = "sum_packets_out"
	FlowAggSumBytesOut                 = "sum_bytes_out"
	FlowAggSumTCPRetranmissions        = "sum_tcp_total_retransmissions"
	FlowAggSumTCPLostPackets           = "sum_tcp_lost_packets"
	FlowAggSumTCPUnrecoveredTO         = "sum_tcp_unrecovered_to"
	FlowAggMinProcessNames             = "process_names_min_num"
	FlowAggMinProcessIds               = "process_ids_min_num"
	FlowAggMinTCPSendCongestionWindow  = "tcp_min_send_congestion_window"
	FlowAggMinTCPMSS                   = "tcp_min_mss"
	FlowAggMaxProcessNames             = "process_names_max_num"
	FlowAggMaxProcessIds               = "process_ids_max_num"
	FlowAggMaxTCPSmoothRTT             = "tcp_max_smooth_rtt"
	FlowAggMaxTCPMinRTT                = "tcp_max_min_rtt"
	FlowAggMeanTCPSendCongestionWindow = "tcp_mean_send_congestion_window"
	FlowAggMeanTCPSmoothRTT            = "tcp_mean_smooth_rtt"
	FlowAggMeanTCPMinRTT               = "tcp_mean_min_rtt"
	FlowAggMeanTCPMSS                  = "tcp_mean_mss"
)

var (
	flowAggregationSums = []lmaelastic.AggSumInfo{
		{Name: FlowAggSumNumFlows, Field: "num_flows"},
		{Name: FlowAggSumNumFlowsStarted, Field: "num_flows_started"},
		{Name: FlowAggSumNumFlowsCompleted, Field: "num_flows_completed"},
		{Name: FlowAggSumPacketsIn, Field: "packets_in"},
		{Name: FlowAggSumBytesIn, Field: "bytes_in"},
		{Name: FlowAggSumPacketsOut, Field: "packets_out"},
		{Name: FlowAggSumBytesOut, Field: "bytes_out"},
		{Name: FlowAggSumTCPRetranmissions, Field: "tcp_total_retransmissions"},
		{Name: FlowAggSumTCPLostPackets, Field: "tcp_lost_packets"},
		{Name: FlowAggSumTCPUnrecoveredTO, Field: "tcp_unrecovered_to"},
	}
	flowAggregationMin = []lmaelastic.AggMaxMinInfo{
		{Name: FlowAggMinProcessNames, Field: "num_process_names"},
		{Name: FlowAggMinProcessIds, Field: "num_process_ids"},
		{Name: FlowAggMinTCPSendCongestionWindow, Field: "tcp_min_send_congestion_window"},
		{Name: FlowAggMinTCPMSS, Field: "tcp_min_mss"},
	}
	flowAggregationMax = []lmaelastic.AggMaxMinInfo{
		{Name: FlowAggMaxProcessNames, Field: "num_process_names"},
		{Name: FlowAggMaxProcessIds, Field: "num_process_ids"},
		{Name: FlowAggMaxTCPSmoothRTT, Field: "tcp_max_smooth_rtt"},
		{Name: FlowAggMaxTCPMinRTT, Field: "tcp_max_min_rtt"},
	}
	flowAggregationMean = []lmaelastic.AggMeanInfo{
		{Name: FlowAggMeanTCPSendCongestionWindow, Field: "tcp_mean_send_congestion_window"},
		{Name: FlowAggMeanTCPSmoothRTT, Field: "tcp_mean_smooth_rtt"},
		{Name: FlowAggMeanTCPMinRTT, Field: "tcp_mean_min_rtt"},
		{Name: FlowAggMeanTCPMSS, Field: "tcp_mean_mss"},
	}
)

type FlowEndpoint struct {
	Type      v1.GraphNodeType
	Namespace string
	Name      string
	NameAggr  string
	Port      int
	Proto     string
}

func (e FlowEndpoint) String() string {
	return fmt.Sprintf("FlowEndpoint(%s/%s/%s/%s:%s:%d)", e.Type, e.Namespace, e.Name, e.NameAggr, e.Proto, e.Port)
}

type L3Flow struct {
	Edge                 FlowEdge
	AggregatedProtoPorts *v1.AggregatedProtoPorts
	Stats                v1.GraphL3Stats
	Processes            *v1.GraphProcesses
}

func (f L3Flow) String() string {
	return fmt.Sprintf("%s [%#v; %#v]", f.Edge, f.AggregatedProtoPorts, f.Stats)
}

type FlowEdge struct {
	Source      FlowEndpoint
	Dest        FlowEndpoint
	ServicePort *ServicePort
}

func (e FlowEdge) String() string {
	if e.ServicePort == nil {
		return fmt.Sprintf("%s -> %s", e.Source, e.Dest)
	}
	return fmt.Sprintf("%s -> %s -> %s", e.Source, e.ServicePort, e.Dest)
}

type ServicePort struct {
	v1.NamespacedName
	Port  string
	Proto string
}

func (s ServicePort) String() string {
	return fmt.Sprintf("ServicePort(%s/%s:%s %s)", s.Namespace, s.Name, s.Port, s.Proto)
}

// Internal value used for tracking.
type reporter byte

const (
	reportedAtSource reporter = iota
	reportedAtDest
)

type L3FlowData struct {
	Flows []L3Flow
}

// GetL3FlowData queries, correlates and aggregates L3 flow data.
// - Source and dest flows are correlated so that we have a single flow with stats for denied-at-source,
//   allowed-at-dest and denied-at-dest.
// - Port information is aggregated when an endpoint port is not part of a service - this prevents bloating a graph
//   when an endpoint is subjected to a port scan.
// - Stats for TCP and Processes are aggregated for each flow.
func GetL3FlowData(ctx context.Context, es lmaelastic.Client, cluster string, tr lmav1.TimeRange, fc *FlowConfig) ([]L3Flow, error) {
	// Track the total buckets queried and the response flows.
	var totalBuckets int
	var fs []L3Flow

	// Trace stats at debug level.
	if log.IsLevelEnabled(log.DebugLevel) {
		start := time.Now()
		log.Debug("GetL3FlowData called")
		defer func() {
			log.Debugf("GetL3FlowData took %s; buckets=%d; flows=%d", time.Since(start), totalBuckets, len(fs))
		}()
	}

	index := elastic.GetFlowLogsIndex(cluster)
	aggQueryL3 := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   elastic.GetEndTimeRangeEpochSecondQuery(tr.From, tr.To),
		Name:                    flowsBucketName,
		AggCompositeSourceInfos: flowCompositeSources,
		AggSumInfos:             flowAggregationSums,
		AggMaxInfos:             flowAggregationMax,
		AggMinInfos:             flowAggregationMin,
		AggMeanInfos:            flowAggregationMean,
	}

	// Perform the L3 composite aggregation query.
	rcvdL3Buckets, rcvdL3Errors := es.SearchCompositeAggregations(ctx, aggQueryL3, nil)

	var lastDestGp *FlowEndpoint
	var dgd *destinationGroupData
	for bucket := range rcvdL3Buckets {
		totalBuckets++
		key := bucket.CompositeAggregationKey
		reporter := key[FlowReporterIdx].String()
		action := key[FlowActionIdx].String()
		proto := singleDashToBlank(key[FlowProtoIdx].String())
		processName := singleDashToBlank(key[FlowProcessIdx].String())
		source := FlowEndpoint{
			Type:      mapRawTypeToGraphNodeType(key[FlowSourceTypeIdx].String(), true),
			NameAggr:  singleDashToBlank(key[FlowSourceNameAggrIdx].String()),
			Namespace: singleDashToBlank(key[FlowSourceNamespaceIdx].String()),
		}
		svc := ServicePort{
			NamespacedName: v1.NamespacedName{
				Name:      singleDashToBlank(key[FlowDestServiceNameIdx].String()),
				Namespace: singleDashToBlank(key[FlowDestServiceNamespaceIdx].String()),
			},
			Port:  singleDashToBlank(key[FlowDestServicePortIdx].String()),
			Proto: proto,
		}
		dest := FlowEndpoint{
			Type:      mapRawTypeToGraphNodeType(key[FlowDestTypeIdx].String(), true),
			NameAggr:  singleDashToBlank(key[FlowDestNameAggrIdx].String()),
			Namespace: singleDashToBlank(key[FlowDestNamespaceIdx].String()),
			Port:      int(key[FlowDestPortIdx].Float64()),
			Proto:     proto,
		}
		gcs := v1.GraphConnectionStats{
			TotalPerSampleInterval: int64(bucket.AggregatedSums[FlowAggSumNumFlows]),
			Started:                int64(bucket.AggregatedSums[FlowAggSumNumFlowsStarted]),
			Completed:              int64(bucket.AggregatedSums[FlowAggSumNumFlowsCompleted]),
		}
		gps := &v1.GraphPacketStats{
			PacketsIn:  int64(bucket.AggregatedSums[FlowAggSumPacketsIn]),
			PacketsOut: int64(bucket.AggregatedSums[FlowAggSumPacketsOut]),
			BytesIn:    int64(bucket.AggregatedSums[FlowAggSumBytesIn]),
			BytesOut:   int64(bucket.AggregatedSums[FlowAggSumBytesOut]),
		}

		// Determine the endpoint key used to group together service groups.
		destGp := GetServiceGroupFlowEndpointKey(dest)

		var tcp *v1.GraphTCPStats
		if proto == "tcp" {
			tcp = &v1.GraphTCPStats{
				SumTotalRetransmissions:  int64(bucket.AggregatedSums[FlowAggSumTCPRetranmissions]),
				SumLostPackets:           int64(bucket.AggregatedSums[FlowAggSumTCPLostPackets]),
				SumUnrecoveredTo:         int64(bucket.AggregatedSums[FlowAggSumTCPUnrecoveredTO]),
				MinSendCongestionWindow:  bucket.AggregatedMin[FlowAggMinTCPSendCongestionWindow],
				MinSendMSS:               bucket.AggregatedMin[FlowAggMinTCPMSS],
				MaxSmoothRTT:             bucket.AggregatedMax[FlowAggMaxTCPSmoothRTT],
				MaxMinRTT:                bucket.AggregatedMax[FlowAggMaxTCPMinRTT],
				MeanSendCongestionWindow: bucket.AggregatedMean[FlowAggMeanTCPSendCongestionWindow],
				MeanSmoothRTT:            bucket.AggregatedMean[FlowAggMeanTCPSmoothRTT],
				MeanMinRTT:               bucket.AggregatedMean[FlowAggMeanTCPMinRTT],
				MeanMSS:                  bucket.AggregatedMean[FlowAggMeanTCPMSS],
			}

			// TCP stats have min and means which could be adversely impacted by zero data which indicates
			// no data rather than actually 0. Only set the document number if the data is non-zero. This prevents us
			// diluting when merging with non-zero data.
			if *tcp != zeroGraphTCPStats {
				tcp.Count = bucket.DocCount
			} else {
				tcp = nil
			}
		}

		// If the source and/or dest group have changed, and we were in the middle of reconciling multiple flows then
		// calculate the final flows.
		if dgd != nil && (destGp == nil || lastDestGp == nil || *destGp != *lastDestGp) {
			fs = append(fs, dgd.getFlows(lastDestGp)...)
			dgd = nil
		}

		// Determine the process info if available in the logs.
		var processes v1.GraphEndpointProcesses
		if processName != "" {
			processes = v1.GraphEndpointProcesses{
				processName: v1.GraphEndpointProcess{
					Name:               processName,
					MinNumNamesPerFlow: int(bucket.AggregatedMin[FlowAggMinProcessNames]),
					MaxNumNamesPerFlow: int(bucket.AggregatedMax[FlowAggMaxProcessNames]),
					MinNumIDsPerFlow:   int(bucket.AggregatedMin[FlowAggMinProcessIds]),
					MaxNumIDsPerFlow:   int(bucket.AggregatedMax[FlowAggMaxProcessIds]),
				},
			}
		}

		// The enumeration order ensures that for any endpoint pair we'll enumerate services before no-services for all
		// sources.
		if dgd == nil {
			log.Debugf("Collating flows: %s -> %s", source, destGp)
			dgd = newDestinationGroupData()
		}
		if log.IsLevelEnabled(log.DebugLevel) {
			if svc.Name != "" {
				log.Debugf("- Processing %s reported flow: %s -> %s -> %s", reporter, source, svc, dest)
			} else {
				log.Debugf("- Processing %s reported flow: %s -> %s", reporter, source, dest)
			}
		}
		dgd.add(reporter, action, source, svc, dest,
			flowStats{packetStats: gps, connStats: gcs, tcpStats: tcp, processes: processes},
		)

		// Store the last dest group.
		lastDestGp = destGp
	}

	// If we were reconciling multiple flows then calculate the final flows.
	if dgd != nil {
		fs = append(fs, dgd.getFlows(lastDestGp)...)
		dgd = nil
	}

	// Adjust some of the statistics based on the aggregation interval.
	timeInterval := tr.Duration()
	l3Flushes := float64(timeInterval) / float64(fc.L3FlowFlushInterval)
	for i := range fs {
		fs[i].Stats.Connections.TotalPerSampleInterval = int64(float64(fs[i].Stats.Connections.TotalPerSampleInterval) / l3Flushes)
	}

	return fs, <-rcvdL3Errors
}

func singleDashToBlank(val string) string {
	if val == "-" {
		return ""
	}
	return val
}

func blankToSingleDash(val string) string {
	if val == "" {
		return "-"
	}
	return val
}

func mapRawTypeToGraphNodeType(val string, agg bool) v1.GraphNodeType {
	switch val {
	case "wep":
		if agg {
			return v1.GraphNodeTypeReplicaSet
		}
		return v1.GraphNodeTypeWorkload
	case "hep":
		return v1.GraphNodeTypeHost
	case "net":
		return v1.GraphNodeTypeNetwork
	case "ns":
		return v1.GraphNodeTypeNetworkSet
	}
	return v1.GraphNodeTypeUnknown
}

func mapGraphNodeTypeToRawType(val v1.GraphNodeType) (string, bool) {
	switch val {
	case v1.GraphNodeTypeWorkload:
		return "wep", false
	case v1.GraphNodeTypeReplicaSet:
		return "wep", true
	case v1.GraphNodeTypeHosts:
		return "hep", true
	case v1.GraphNodeTypeHost:
		return "hep", false
	case v1.GraphNodeTypeNetwork:
		return "net", true
	case v1.GraphNodeTypeNetworkSet:
		return "ns", true
	}
	return "", false
}

type ports struct {
	ranges []v1.PortRange
}

func (p *ports) add(port int) {
	for i := range p.ranges {
		if p.ranges[i].MinPort >= port && p.ranges[i].MaxPort <= port {
			// Already have this Port range. Nothing to do.
			return
		}
		if p.ranges[i].MinPort == port+1 {
			// Expand the lower value of this range.
			p.ranges[i].MinPort = port
			if i > 0 && p.ranges[i-1].MaxPort == port {
				// Consolidate previous with this entry.
				p.ranges[i-1].MaxPort = p.ranges[i].MaxPort
				p.ranges = append(p.ranges[:i-1], p.ranges[i:]...)
			}
			return
		}
		if p.ranges[i].MaxPort == port-1 {
			// Expand the upper value of this range.
			p.ranges[i].MaxPort = port
			if i < len(p.ranges)-1 && p.ranges[i+1].MinPort == port {
				// Consolidate this with next entry.
				p.ranges[i].MaxPort = p.ranges[i+1].MaxPort
				p.ranges = append(p.ranges[:i], p.ranges[i+1:]...)
			}
			return
		}
		if p.ranges[i].MinPort > port {
			// This entry is between the previous and this one. Shift along and insert. Note that the append copies
			// this entry twice which is then copied over - but this makes for simple code.
			p.ranges = append(p.ranges[:i+1], p.ranges[i:]...)
			p.ranges[i] = v1.PortRange{MinPort: port, MaxPort: port}
			return
		}
	}
	// Extend the slice with this Port.
	p.ranges = append(p.ranges, v1.PortRange{MinPort: port, MaxPort: port})
}

func newDestinationGroupData() *destinationGroupData {
	return &destinationGroupData{
		sources:                make(map[FlowEndpoint]*sourceData),
		allServiceDestinations: make(map[FlowEndpoint]bool),
	}
}

// destinationGroupData is used to temporarily collate flow data associated with a common source -> destination group.
type destinationGroupData struct {
	sources                map[FlowEndpoint]*sourceData
	allServiceDestinations map[FlowEndpoint]bool
}

func (d destinationGroupData) add(
	reporter, action string, source FlowEndpoint, svc ServicePort, destination FlowEndpoint, stats flowStats,
) {
	if svc.Name != "" {
		d.allServiceDestinations[destination] = true
	}

	sourceGroup := d.sources[source]
	if sourceGroup == nil {
		sourceGroup = newSourceData()
		d.sources[source] = sourceGroup
	}
	sourceGroup.add(reporter, action, svc, destination, stats, d.allServiceDestinations[destination])
}

func (d *destinationGroupData) getFlows(destGp *FlowEndpoint) []L3Flow {
	var fs []L3Flow
	log.Debug("Handling source/dest reconciliation")
	for source, data := range d.sources {
		fs = append(fs, data.getFlows(source, destGp)...)
	}
	return fs
}

type sourceData struct {
	// Service Endpoints.
	serviceDestinations map[FlowEndpoint]*flowReconciliationData

	// AggregatedProtoPorts data for non-service Endpoints.
	other      *flowReconciliationData
	protoPorts map[string]*ports
}

func newSourceData() *sourceData {
	return &sourceData{
		serviceDestinations: make(map[FlowEndpoint]*flowReconciliationData),
		protoPorts:          make(map[string]*ports),
	}
}

func (s *sourceData) add(
	reporter, action string, svc ServicePort, destination FlowEndpoint, stats flowStats, isServiceEndpoint bool,
) {
	rc := s.serviceDestinations[destination]
	if rc == nil && isServiceEndpoint {
		// If there is a service then we can create a service destination (since services are enumerated before
		// no service).
		rc = newFlowReconciliationData()
		s.serviceDestinations[destination] = rc
	}
	if rc != nil {
		// We have a flowReconciliationData for the service. Combine the stats to that.
		log.Debug("  endpoint is part of a service")
		rc.add(reporter, action, svc, stats)
		return
	}

	// Aggregate the Port and Proto information.
	log.Debug("  endpoint is not part of a service - aggregate port and proto info")

	// We do not have a flowReconciliationData which means we must be aggregating out the Port and Proto for this
	// (non-service related) flow.
	if rc = s.other; rc == nil {
		// There is no existing service destination and this flow does not contain a service. Since services are
		// enumerated first then this Proto Port combination is not part of a service and we should consolidate
		// the Proto and ports.
		log.Debug("  create new aggregated reconciliation data")
		rc = newFlowReconciliationData()
		s.other = rc
	}

	p, ok := s.protoPorts[svc.Proto]
	if !ok {
		if destination.Port != 0 {
			p = &ports{}
		}
		s.protoPorts[svc.Proto] = p
	}
	if p != nil {
		p.add(destination.Port)
	}

	// Combine the data to the aggregated data set.
	rc.add(reporter, action, ServicePort{}, stats)
}

func (s *sourceData) getFlows(source FlowEndpoint, destGp *FlowEndpoint) []L3Flow {
	var fs []L3Flow

	// Combine the reconciled flows for each endpoint/Proto that is part of one or more services.
	for dest, frd := range s.serviceDestinations {
		fs = append(fs, frd.getFlows(source, dest)...)
	}

	// Combine the aggregated info. There should at most a single flow here.
	if s.other != nil {
		log.Debug(" add flow with aggregated ports and protocols")
		dest := FlowEndpoint{
			Type:      destGp.Type,
			Namespace: destGp.Namespace,
			Name:      destGp.Name,
			NameAggr:  destGp.NameAggr,
		}
		if other := s.other.getFlows(source, dest); len(other) == 1 {
			log.Debug(" calculate aggregated ports and protocols")
			f := other[0]
			f.AggregatedProtoPorts = &v1.AggregatedProtoPorts{}
			for proto, ports := range s.protoPorts {
				aggPorts := v1.AggregatedPorts{
					Protocol: proto,
				}
				if ports != nil {
					for i := range ports.ranges {
						if len(aggPorts.PortRanges) >= maxAggregatedPortRangesPerProtocol {
							aggPorts.NumOtherPorts += ports.ranges[i].Num()
						} else {
							aggPorts.PortRanges = append(aggPorts.PortRanges, ports.ranges[i])
						}
					}
				}
				f.AggregatedProtoPorts.ProtoPorts = append(f.AggregatedProtoPorts.ProtoPorts, aggPorts)

				if len(f.AggregatedProtoPorts.ProtoPorts) >= maxAggregatedProtocol {
					f.AggregatedProtoPorts.NumOtherProtocols = len(s.protoPorts) - len(f.AggregatedProtoPorts.ProtoPorts)
					break
				}
			}

			fs = append(fs, f)
		} else {
			log.Errorf("Multiple flows with aggregated ports and protocols: %#v", other)
		}
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		if len(fs) == 0 {
			log.Debug("Collated flows discarded")
		} else {
			log.Debug("Collated flows converted ")
			for _, f := range fs {
				log.Debugf("- %s", f)
			}
		}
	}

	return fs
}

func newFlowReconciliationData() *flowReconciliationData {
	return &flowReconciliationData{
		sourceReportedDenied:  make(map[ServicePort]flowStats),
		sourceReportedAllowed: make(map[ServicePort]flowStats),
		destReportedDenied:    make(map[ServicePort]flowStats),
		destReportedAllowed:   make(map[ServicePort]flowStats),
	}
}

type flowStats struct {
	packetStats *v1.GraphPacketStats
	connStats   v1.GraphConnectionStats
	tcpStats    *v1.GraphTCPStats
	processes   v1.GraphEndpointProcesses
}

func (f flowStats) add(f2 flowStats) flowStats {
	return flowStats{
		packetStats: f.packetStats.Add(f2.packetStats),
		connStats:   f.connStats.Add(f2.connStats),
		tcpStats:    f.tcpStats.Combine(f2.tcpStats),
		processes:   f.processes.Combine(f2.processes),
	}
}

// flowReconciliationData is used to temporarily collate source and dest statistics when the flow will be recorded by
// both source and dest.
//
// Service information available in source reported flows may be missing from the destination flows. The destination
// flows have the final verdict (allow or deny) that is missing from the source flow. This helper divvies up the
// destination allowed and denied flows with the source reported allowed flows. We use the source data for the actual
// total packets stats and the destination data for the proportional values of which flows were allowed and denied at
// dest. This is obviously an approximation, but the best we can do without additional data to correlate.
type flowReconciliationData struct {
	sourceReportedDenied  map[ServicePort]flowStats
	sourceReportedAllowed map[ServicePort]flowStats
	destReportedAllowed   map[ServicePort]flowStats
	destReportedDenied    map[ServicePort]flowStats
}

func (d *flowReconciliationData) add(
	reporter, action string, svc ServicePort, f flowStats,
) {
	if reporter == "src" {
		if action == "allow" {
			log.Debug("  found source reported allowed flow")
			d.sourceReportedAllowed[svc] = d.sourceReportedAllowed[svc].add(f)
		} else {
			log.Debug("  found source reported denied flow")
			d.sourceReportedDenied[svc] = d.sourceReportedDenied[svc].add(f)
		}
	} else {
		if action == "allow" {
			log.Debug("  found dest reported allowed flow")
			d.destReportedAllowed[svc] = d.destReportedAllowed[svc].add(f)
		} else {
			log.Debug("  found dest reported denied flow")
			d.destReportedDenied[svc] = d.destReportedDenied[svc].add(f)
		}
	}
}

// getFlows returns the final reconciled flows. This essentially divvies up the destination edges across the
// various source reported flows based on simple proportion.
func (d *flowReconciliationData) getFlows(source, dest FlowEndpoint) []L3Flow {
	var f []L3Flow

	addFlow := func(svc ServicePort, stats v1.GraphL3Stats, processes *v1.GraphProcesses) {
		log.Debugf("  Including flow for service: %s", svc)
		var spp *ServicePort
		if svc.Name != "" {
			spp = &svc
		}

		f = append(f, L3Flow{
			Edge: FlowEdge{
				Source:      source,
				Dest:        dest,
				ServicePort: spp,
			},
			Stats:     stats,
			Processes: processes,
		})
	}

	allServices := func(allowed, denied map[ServicePort]flowStats) set.Set {
		services := set.New()
		for s := range allowed {
			services.Add(s)
		}
		for s := range denied {
			services.Add(s)
		}
		return services
	}

	addSingleReportedFlows := func(allowed, denied map[ServicePort]flowStats, rep reporter) {
		allServices(allowed, denied).Iter(func(item interface{}) error {
			svc := item.(ServicePort)
			stats := v1.GraphL3Stats{
				Connections: allowed[svc].connStats.Add(denied[svc].connStats),
				Allowed:     allowed[svc].packetStats,
				TCP:         allowed[svc].tcpStats,
			}
			epProcesses := allowed[svc].processes.Combine(denied[svc].processes)
			var processes *v1.GraphProcesses

			if rep == reportedAtSource {
				stats.DeniedAtSource = denied[svc].packetStats
				if len(epProcesses) > 0 {
					processes = &v1.GraphProcesses{
						Source: epProcesses,
					}
				}
			} else {
				stats.DeniedAtDest = denied[svc].packetStats
				if len(epProcesses) > 0 {
					processes = &v1.GraphProcesses{
						Dest: epProcesses,
					}
				}
			}

			addFlow(svc, stats, processes)
			return nil
		})
	}

	sourceReported := len(d.sourceReportedAllowed) > 0 || len(d.sourceReportedDenied) > 0
	destReported := len(d.destReportedAllowed) > 0 || len(d.destReportedDenied) > 0

	if sourceReported {
		if !destReported {
			log.Debug("  L3Flow reported at source only")
			addSingleReportedFlows(d.sourceReportedAllowed, d.sourceReportedDenied, reportedAtSource)
			return f
		}
	} else if destReported {
		if !sourceReported {
			log.Debug("  L3Flow reported at dest only")
			addSingleReportedFlows(d.destReportedAllowed, d.destReportedDenied, reportedAtDest)
			return f
		}
	}

	// The flow will be reported at source and dest, which most importantly means the allowed flows at source need to be
	// divvied up to be allowed or denied at dest.
	log.Debug("  L3Flow reported at source and dest")
	allServices(d.sourceReportedAllowed, d.sourceReportedDenied).Iter(func(item interface{}) error {
		svc := item.(ServicePort)

		// Get the stats for allowed and denied at dest.  Combine the stats for direct A->B and A->SVC->B. We don't expect
		// the latter, but just in case...
		totalAllowedAtDest := d.destReportedAllowed[ServicePort{Proto: svc.Proto}].packetStats.
			Add(d.destReportedAllowed[svc].packetStats)
		totalDeniedAtDest := d.destReportedDenied[ServicePort{Proto: svc.Proto}].packetStats.
			Add(d.destReportedDenied[svc].packetStats)

		var allowed, deniedAtDest *v1.GraphPacketStats
		if totalAllowedAtDest == nil {
			deniedAtDest = d.sourceReportedAllowed[svc].packetStats
		} else if totalDeniedAtDest == nil {
			allowed = d.sourceReportedAllowed[svc].packetStats
		} else {
			// Get the proportion allowed at dest and we'll assume the remainder is denied.
			propAllowed := totalAllowedAtDest.Prop(totalDeniedAtDest)
			allowed = d.sourceReportedAllowed[svc].packetStats.Multiply(propAllowed)
			deniedAtDest = d.sourceReportedAllowed[svc].packetStats.Sub(allowed)
		}

		// Determine graph processes.
		var processes *v1.GraphProcesses
		sourceProcesses := d.sourceReportedAllowed[svc].processes.
			Combine(d.sourceReportedDenied[svc].processes)
		destProcesses := d.destReportedAllowed[ServicePort{Proto: svc.Proto}].processes.
			Combine(d.destReportedAllowed[svc].processes).
			Combine(d.destReportedDenied[ServicePort{Proto: svc.Proto}].processes).
			Combine(d.destReportedDenied[svc].processes)
		if len(destProcesses) > 0 && len(sourceProcesses) > 0 {
			processes = &v1.GraphProcesses{
				Source: sourceProcesses,
				Dest:   destProcesses,
			}
		}

		addFlow(svc, v1.GraphL3Stats{
			Allowed:        allowed,
			DeniedAtSource: d.sourceReportedDenied[svc].packetStats,
			DeniedAtDest:   deniedAtDest,
			Connections:    d.sourceReportedAllowed[svc].connStats.Add(d.sourceReportedDenied[svc].connStats),
			TCP: d.sourceReportedAllowed[svc].tcpStats.Combine(d.sourceReportedDenied[svc].tcpStats).
				Combine(d.destReportedAllowed[svc].tcpStats).Combine(d.destReportedDenied[svc].tcpStats),
		}, processes)
		return nil
	})
	return f
}
