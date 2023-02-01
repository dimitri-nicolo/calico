// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package legacy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/set"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

// flowBackend implements the Backend interface for flows stored
// in elasticsearch in the legacy storage model.
type flowBackend struct {
	// Elasticsearch client.
	lmaclient lmaelastic.Client

	// Track mapping of field name to its index in the ES response.
	ft *fieldTracker

	// The sources and aggregations to use when building an aggregation query against ES.
	compositeSources []lmaelastic.AggCompositeSourceInfo
	aggSums          []lmaelastic.AggSumInfo
	aggMins          []lmaelastic.AggMaxMinInfo
	aggMaxs          []lmaelastic.AggMaxMinInfo
	aggMeans         []lmaelastic.AggMeanInfo
	aggNested        []lmaelastic.AggNestedTermInfo
}

func NewFlowBackend(c lmaelastic.Client) bapi.FlowBackend {
	// These are the keys which define a flow in ES, and will be used to create buckets in the ES result.
	compositeSources := []lmaelastic.AggCompositeSourceInfo{
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

	sums := []lmaelastic.AggSumInfo{
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
	mins := []lmaelastic.AggMaxMinInfo{
		{Name: FlowAggMinProcessNames, Field: "num_process_names"},
		{Name: FlowAggMinProcessIds, Field: "num_process_ids"},
		{Name: FlowAggMinTCPSendCongestionWindow, Field: "tcp_min_send_congestion_window"},
		{Name: FlowAggMinTCPMSS, Field: "tcp_min_mss"},
	}
	maxs := []lmaelastic.AggMaxMinInfo{
		{Name: FlowAggMaxProcessNames, Field: "num_process_names"},
		{Name: FlowAggMaxProcessIds, Field: "num_process_ids"},
		{Name: FlowAggMaxTCPSmoothRTT, Field: "tcp_max_smooth_rtt"},
		{Name: FlowAggMaxTCPMinRTT, Field: "tcp_max_min_rtt"},
	}
	means := []lmaelastic.AggMeanInfo{
		{Name: FlowAggMeanTCPSendCongestionWindow, Field: "tcp_mean_send_congestion_window"},
		{Name: FlowAggMeanTCPSmoothRTT, Field: "tcp_mean_smooth_rtt"},
		{Name: FlowAggMeanTCPMinRTT, Field: "tcp_mean_min_rtt"},
		{Name: FlowAggMeanTCPMSS, Field: "tcp_mean_mss"},
	}

	// We use a nested terms aggregation in order to query all the label key/value pairs
	// attached to the source and destination for this flow over it's life.
	//
	// NOTE: Nested terms aggregations have an inherent limit of 10 results. As a result,
	// for endpoints with many labels this can result in an incomplete response. We could use
	// a composite aggregation or increase the size instead, but that is more computationally expensive. A nested
	// terms aggregation is consistent with past behavior.
	// https://www.elastic.co/guide/en/elasticsearch/reference/current/search-aggregations-bucket-terms-aggregation.html#search-aggregations-bucket-terms-aggregation-size
	nested := []lmaelastic.AggNestedTermInfo{
		{
			Name:  "dest_labels",
			Path:  "dest_labels",
			Term:  "by_kvpair",
			Field: "dest_labels.labels",
		},
		{
			Name:  "source_labels",
			Path:  "source_labels",
			Term:  "by_kvpair",
			Field: "source_labels.labels",
		},
	}

	return &flowBackend{
		lmaclient: c,
		ft:        newFieldTracker(compositeSources),

		// Configuration for the aggregation queries we make against ES.
		compositeSources: compositeSources,
		aggSums:          sums,
		aggMins:          mins,
		aggMaxs:          maxs,
		aggMeans:         means,
		aggNested:        nested,
	}
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

// List returns all flows which match the given options.
func (b *flowBackend) List(ctx context.Context, i bapi.ClusterInfo, opts v1.L3FlowParams) ([]v1.L3Flow, error) {
	log := contextLogger(i)

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID set on flow request")
	}

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

	// Default the number of results to 1000 if there is no limit
	// set on the query.
	numResults := opts.MaxResults
	if numResults == 0 {
		numResults = 1000
	}

	// Build the aggregation request.
	query := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           buildFlowsIndex(i.Cluster),
		Query:                   lmaindex.FlowLogs().NewTimeRangeQuery(start, end),
		Name:                    "flog_buckets",
		AggCompositeSourceInfos: b.compositeSources,
		AggSumInfos:             b.aggSums,
		AggMaxInfos:             b.aggMaxs,
		AggMinInfos:             b.aggMins,
		AggMeanInfos:            b.aggMeans,
		AggNestedTermInfos:      b.aggNested,
		MaxBucketsPerQuery:      numResults,
	}

	// Context for the ES request.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Infof("Listing flows from index %s", query.DocumentIndex)

	var allFlows []v1.L3Flow

	// Perform the request.
	// TODO: We're iterating over a channel. We need to support paging.
	buckets, errors := b.lmaclient.SearchCompositeAggregations(ctx, query, nil)
	for bucket := range buckets {
		log.Infof("Processing bucket built from %d logs", bucket.DocCount)
		key := bucket.CompositeAggregationKey

		// Build the flow, starting with the key.
		flow := v1.L3Flow{Key: v1.L3FlowKey{}}
		flow.Key.Reporter = b.ft.ValueString(key, "reporter")
		flow.Key.Action = b.ft.ValueString(key, "action")
		flow.Key.Protocol = b.ft.ValueString(key, "proto")
		flow.Key.Source = v1.Endpoint{
			Type:           v1.EndpointType(b.ft.ValueString(key, "source_type")),
			AggregatedName: b.ft.ValueString(key, "source_name_aggr"),
			Namespace:      b.ft.ValueString(key, "source_namespace"),
		}
		flow.Key.Destination = v1.Endpoint{
			Type:           v1.EndpointType(b.ft.ValueString(key, "dest_type")),
			AggregatedName: b.ft.ValueString(key, "dest_name_aggr"),
			Namespace:      b.ft.ValueString(key, "dest_namespace"),
			Port:           b.ft.ValueInt64(key, "dest_port"),
		}

		// Build the flow.
		flow.LogStats = &v1.LogStats{
			LogCount:  int64(bucket.AggregatedSums[FlowAggSumNumFlows]),
			Started:   int64(bucket.AggregatedSums[FlowAggSumNumFlowsStarted]),
			Completed: int64(bucket.AggregatedSums[FlowAggSumNumFlowsCompleted]),
		}

		flow.Service = &v1.Service{
			Name:      b.ft.ValueString(key, "dest_service_name"),
			Namespace: b.ft.ValueString(key, "dest_service_namespace"),
			Port:      b.ft.ValueInt32(key, "dest_service_port_num"),
			PortName:  b.ft.ValueString(key, "dest_service_port"),
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
		processName := b.ft.ValueString(key, "process_name")
		if processName != "" {
			flow.Process = &v1.Process{Name: processName}
			flow.ProcessStats = &v1.ProcessStats{
				MinNumNamesPerFlow: int(bucket.AggregatedMin[FlowAggMinProcessNames]),
				MaxNumNamesPerFlow: int(bucket.AggregatedMax[FlowAggMaxProcessNames]),
				MinNumIDsPerFlow:   int(bucket.AggregatedMin[FlowAggMinProcessIds]),
				MaxNumIDsPerFlow:   int(bucket.AggregatedMax[FlowAggMaxProcessIds]),
			}
		}

		// Handle label aggregation.
		flow.DestinationLabels = getLabelsFromLabelAggregation(log, bucket.AggregatedTerms, "dest_labels")
		flow.SourceLabels = getLabelsFromLabelAggregation(log, bucket.AggregatedTerms, "source_labels")

		// TODO: Handle policy report aggregations

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
	case err := <-errors:
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

func buildFlowsIndex(cluster string) string {
	return fmt.Sprintf("tigera_secure_ee_flows.%s", cluster)
}

// getLabelsFromLabelAggregation parses the labels out from the given aggregation and puts them into a map map[string][]FlowResponseLabels
// that can be sent back in the response.
func getLabelsFromLabelAggregation(log *logrus.Entry, terms map[string]*lmaelastic.AggregatedTerm, k string) []v1.FlowLabels {
	tracker := newLabelTracker()
	logrus.Debugf("%s buckets: %+v", k, terms[k].Buckets)
	for i := range terms[k].Buckets {
		label, ok := i.(string)
		if !ok {
			log.WithField("value", i).Warning("skipping bucket with non-string label")
			continue
		}
		labelParts := strings.Split(label, "=")
		if len(labelParts) != 2 {
			log.WithField("value", label).Warning("skipping bucket with key with invalid format (format should be 'key=value')")
			continue
		}

		labelName, labelValue := labelParts[0], labelParts[1]

		// TODO: Do we need to include bucket.DocCount per-label?
		tracker.Add(labelName, labelValue)
	}

	return tracker.Labels()
}

func newLabelTracker() *labelTracker {
	return &labelTracker{
		s: make(map[string]set.Set[string]),
	}
}

type labelTracker struct {
	// Map of key to set of values seen for that key.
	s       map[string]set.Set[string]
	allKeys []string
}

func (t *labelTracker) Add(k, v string) {
	if _, ok := t.s[k]; !ok {
		// New label key
		t.s[k] = set.New[string]()
		t.allKeys = append(t.allKeys, k)
	}
	t.s[k].Add(v)
}

func (t *labelTracker) Labels() []v1.FlowLabels {
	labels := []v1.FlowLabels{}

	// Sort keys so we get a consistenly ordered output.
	sort.Strings(t.allKeys)

	for _, key := range t.allKeys {
		v := t.s[key]

		// Again, sort the values slice so that we get consistent output.
		values := v.Slice()
		sort.Strings(values)
		labels = append(labels, v1.FlowLabels{
			Key:    key,
			Values: values,
		})
	}
	return labels
}
