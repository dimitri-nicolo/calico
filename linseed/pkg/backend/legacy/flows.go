// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package legacy

import (
	"context"
	"fmt"
	"strings"
	"time"

	elastic "github.com/olivere/elastic/v7"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
	"github.com/sirupsen/logrus"
)

var (
	// These are the keys which define a flow in ES, and will be used to create buckets in the ES result.
	flowCompositeSources = []lmaelastic.AggCompositeSourceInfo{
		{Name: "dest_type", Field: "dest_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name_aggr", Field: "dest_name_aggr"},
		{Name: "dest_service_namespace", Field: "dest_service_namespace", Order: "desc"},
		{Name: "dest_service_name", Field: "dest_service_name"},
		{Name: "dest_service_port_name", Field: "dest_service_port"},
		{Name: "dest_service_port_num", Field: "dest_service_port_num", AllowMissingBucket: true},
		{Name: "proto", Field: "proto"},
		{Name: "dest_port_num", Field: "dest_port"},
		{Name: "source_type", Field: "source_type"},
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "source_name_aggr", Field: "source_name_aggr"},
		{Name: "process_name", Field: "process_name"},
		{Name: "reporter", Field: "reporter"},
		{Name: "action", Field: "action"},
	}

	// Track mapping of field name to its index in the ES response.
	// TODO: Right now, this is unused.
	fieldToIndex map[string]int
)

func init() {
	// Build mappings of field name to its corresponding index. Since ES reponses are
	// ordered according to the order of the composite sources, we need the indices to match.
	fieldToIndex = map[string]int{}

	for idx, source := range flowCompositeSources {
		fieldToIndex[source.Field] = idx
	}
}

// TODO: Replace this with the fieldToIndex mapping.
const (
	// The ordering of these composite sources is important. We want to enumerate all services across all sources for
	// a given destination, and we need to ensure:
	FlowDestTypeIdx = iota
	FlowDestNamespaceIdx
	FlowDestNameAggrIdx
	FlowDestServiceNamespaceIdx
	FlowDestServiceNameIdx
	FlowDestServicePortNameIdx
	FlowDestServicePortNumIdx
	FlowProtoIdx
	FlowDestPortNumIdx
	FlowSourceTypeIdx
	FlowSourceNamespaceIdx
	FlowSourceNameAggrIdx
	FlowProcessIdx
	FlowReporterIdx
	FlowActionIdx
)

// FlowBackend implements the Backend interface for flows stored
// in elasticsearch in the legacy storage model.
type FlowBackend struct {
	// Elasticsearch client.
	client *elastic.Client

	lmaclient lmaelastic.Client
}

func NewFlowBackend(c lmaelastic.Client) *FlowBackend {
	return &FlowBackend{
		client:    c.Backend(),
		lmaclient: c,
	}
}

// TODO: Move this to a common flows package.
type GetOptions struct {
	// Common options for scoping the request.
	Cluster string
	Tenant  string

	// Identifiers for a flow
	StartTime string
	EndTime   string

	// Sets a limit on number of returned flows.
	MaxFlows int
}

// TODO: Fill this out. Belongs in common API package.
type LabelMatch struct{}

func contextLogger(opts v1.L3FlowParams) *logrus.Entry {
	f := logrus.Fields{}
	return logrus.WithFields(f)
}

const (
	// TODO(rlb): We might want to abbreviate these to reduce the amount of data on the wire, json parsing and
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

func (b *FlowBackend) Initialize(ctx context.Context) error {
	return nil
}

// List returns all flows which match the given options.
func (b *FlowBackend) List(ctx context.Context, i bapi.ClusterInfo, opts v1.L3FlowParams) ([]v1.L3Flow, error) {
	log := contextLogger(opts)

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID set on flow request")
	}

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

	// Parse times from the request.
	var start, end time.Time
	if opts.QueryParams != nil && opts.QueryParams.TimeRange != nil {
		start = opts.QueryParams.TimeRange.From
		end = opts.QueryParams.TimeRange.To
	} else {
		// Default to the latest 5 minute window.
		start = time.Now().Add(-5 * time.Minute)
		end = time.Now()
	}

	// Build the aggregation request.
	aggQueryL3 := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           buildFlowsIndex(i.Cluster),
		Query:                   lmaindex.FlowLogs().NewTimeRangeQuery(start, end),
		Name:                    "flog_buckets",
		AggCompositeSourceInfos: flowCompositeSources,
		AggSumInfos:             flowAggregationSums,
		AggMaxInfos:             flowAggregationMax,
		AggMinInfos:             flowAggregationMin,
		AggMeanInfos:            flowAggregationMean,
		MaxBucketsPerQuery:      opts.MaxResults,
	}

	// Context for the ES request.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	singleDashToBlank := func(s string) string {
		if s == "-" {
			return ""
		}
		return s
	}

	log.Infof("Listing flows from index %s", aggQueryL3.DocumentIndex)

	allFlows := []v1.L3Flow{}

	// Perform the request.
	// TODO: We're iterating over a channel. We need to support paging.
	rcvdL3Buckets, rcvdL3Errors := b.lmaclient.SearchCompositeAggregations(ctx, aggQueryL3, nil)
	for bucket := range rcvdL3Buckets {
		log.Infof("Processing bucket built from %d flows", bucket.DocCount)
		key := bucket.CompositeAggregationKey

		flow := v1.L3Flow{Key: v1.L3FlowKey{}}

		// Build the key for this flow.
		flow.Key.Reporter = key[FlowReporterIdx].String()
		flow.Key.Action = key[FlowActionIdx].String()
		flow.Key.Protocol = key[FlowProtoIdx].String()
		flow.Key.Source = v1.Endpoint{
			Type:           v1.EndpointType(key[FlowSourceTypeIdx].String()),
			AggregatedName: singleDashToBlank(key[FlowSourceNameAggrIdx].String()),
			Namespace:      singleDashToBlank(key[FlowSourceNamespaceIdx].String()),
		}
		flow.Key.Destination = v1.Endpoint{
			Type:           v1.EndpointType(key[FlowDestTypeIdx].String()),
			AggregatedName: singleDashToBlank(key[FlowDestNameAggrIdx].String()),
			Namespace:      singleDashToBlank(key[FlowDestNamespaceIdx].String()),
			Port:           int64(key[FlowDestPortNumIdx].Float64()),
		}

		// Build the flow.
		flow.LogStats = &v1.LogStats{
			LogCount:  int64(bucket.AggregatedSums[FlowAggSumNumFlows]),
			Started:   int64(bucket.AggregatedSums[FlowAggSumNumFlowsStarted]),
			Completed: int64(bucket.AggregatedSums[FlowAggSumNumFlowsCompleted]),
		}

		flow.Service = &v1.Service{
			Name:      singleDashToBlank(key[FlowDestServiceNameIdx].String()),
			Namespace: singleDashToBlank(key[FlowDestServiceNamespaceIdx].String()),
			Port:      int32(key[FlowDestServicePortNumIdx].Float64()),
			PortName:  singleDashToBlank(key[FlowDestServicePortNameIdx].String()),
		}

		flow.TrafficStats = &v1.TrafficStats{
			PacketsIn:  int64(bucket.AggregatedSums[FlowAggSumPacketsIn]),
			PacketsOut: int64(bucket.AggregatedSums[FlowAggSumPacketsOut]),
			BytesIn:    int64(bucket.AggregatedSums[FlowAggSumBytesIn]),
			BytesOut:   int64(bucket.AggregatedSums[FlowAggSumBytesOut]),
		}

		if flow.Key.Protocol == "tcp" {
			flow.TCPStats = &v1.TCPStats{
				TotalRetransmissions:     int64(bucket.AggregatedSums[FlowAggSumTCPRetranmissions]),
				LostPackets:              int64(bucket.AggregatedSums[FlowAggSumTCPLostPackets]),
				UnrecoveredTo:            int64(bucket.AggregatedSums[FlowAggSumTCPUnrecoveredTO]),
				MinSendCongestionWindow:  bucket.AggregatedMin[FlowAggMinTCPSendCongestionWindow],
				MinMSS:                   bucket.AggregatedMin[FlowAggMinTCPMSS],
				MaxSmoothRTT:             bucket.AggregatedMax[FlowAggMaxTCPSmoothRTT],
				MaxMinRTT:                bucket.AggregatedMax[FlowAggMaxTCPMinRTT],
				MeanSendCongestionWindow: bucket.AggregatedMean[FlowAggMeanTCPSendCongestionWindow],
				MeanSmoothRTT:            bucket.AggregatedMean[FlowAggMeanTCPSmoothRTT],
				MeanMinRTT:               bucket.AggregatedMean[FlowAggMeanTCPMinRTT],
				MeanMSS:                  bucket.AggregatedMean[FlowAggMeanTCPMSS],
			}
		}

		// Determine the process info if available in the logs.
		// var processes v1.GraphEndpointProcesses
		processName := singleDashToBlank(key[FlowProcessIdx].String())
		if processName != "" {
			flow.Process = &v1.Process{Name: processName}
			flow.ProcessStats = &v1.ProcessStats{
				MinNumNamesPerFlow: int(bucket.AggregatedMin[FlowAggMinProcessNames]),
				MaxNumNamesPerFlow: int(bucket.AggregatedMax[FlowAggMaxProcessNames]),
				MinNumIDsPerFlow:   int(bucket.AggregatedMin[FlowAggMinProcessIds]),
				MaxNumIDsPerFlow:   int(bucket.AggregatedMax[FlowAggMaxProcessIds]),
			}
		}

		// Add the flow to the batch.
		allFlows = append(allFlows, flow)

		// Track the number of flows. Bail if we hit the absolute maximum number of flows.
		if opts.MaxResults != 0 && len(allFlows) >= opts.MaxResults {
			log.Warnf("Maximum number of flows (%d) reached. Stopping flow processing", opts.MaxResults)
			break
		}
	}

	// TODO: check for errors. This is a temporary hack so that we
	// get some error reporting at least.
	select {
	case err := <-rcvdL3Errors:
		if err != nil {
			log.Errorf("Error processing list request: %s", err)
			return allFlows, err
		}
	default:
		// No error
	}

	// Adjust some of the statistics based on the aggregation interval.
	// timeInterval := tr.Duration()
	// l3Flushes := float64(timeInterval) / float64(fc.L3FlowFlushInterval)
	// for i := range fs {
	// 	fs[i].Stats.Connections.TotalPerSampleInterval = int64(float64(fs[i].Stats.Connections.TotalPerSampleInterval) / l3Flushes)
	// }

	return allFlows, nil
}

func (b *FlowBackend) Get(ctx context.Context, opts GetOptions) (*v1.L3Flow, error) {
	return nil, fmt.Errorf("Not implemented")
}

func buildFlowsIndex(cluster string) string {
	// TODO: Handle presence of a tenant ID
	return fmt.Sprintf("tigera_secure_ee_flows.%s", cluster)
}

// getLabelsFromLabelAggregation parses the labels out from the given aggregation and puts them into a map map[string][]FlowResponseLabels
// that can be sent back in the response.
func getLabelsFromLabelAggregation(labelAggregation *elastic.AggregationSingleBucket) map[string][]string {
	tracker := labelTracker{m: map[string][]string{}}

	if terms, found := labelAggregation.Terms("by_kvpair"); found {
		for _, bucket := range terms.Buckets {
			key, ok := bucket.Key.(string)
			if !ok {
				// TODO: Use context logger
				logrus.WithField("key", key).Warning("skipping bucket with non string key type")
				continue
			}

			labelParts := strings.Split(key, "=")
			if len(labelParts) != 2 {
				// TODO: Use context logger
				logrus.WithField("key", key).Warning("skipping bucket with key with invalid format (format should be 'key=value')")
				continue
			}

			labelName, labelValue := labelParts[0], labelParts[1]
			// TODO: Do we need to include bucket.DocCount per-label?
			tracker.Add(labelName, labelValue)
		}
	}

	return tracker.Map()
}

type labelTracker struct {
	m map[string][]string
}

func (t *labelTracker) Add(k, v string) {
	if _, ok := t.m[k]; !ok {
		// New label key
		t.m[k] = []string{}
	}
	t.m[k] = append(t.m[k], v)
}

func (t *labelTracker) Map() map[string][]string {
	return t.m
}
