// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	lmav1 "github.com/tigera/lma/pkg/apis/v1"
	lmaelastic "github.com/tigera/lma/pkg/elastic"
	lmaindex "github.com/tigera/lma/pkg/elastic/index"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

type L7Flow struct {
	Edge  FlowEdge
	Stats v1.GraphL7Stats
}

const (
	l7DestTypeIdx = iota
	l7DestNamespaceIdx
	l7DestNameAggrIdx
	l7DestServiceNamespaceIdx
	l7DestServiceNameIdx
	l7DestServicePortNameIdx
	l7DestServicePortNumIdx
	l7DestPortNumIdx
	l7SourceTypeIdx
	l7SourceNamespaceIdx
	l7SourceNameAggrIdx
	//l7ProcessIdx
	//l7ReporterIdx
	l7ResponseCodeIdx
)

const l7Proto = "tcp"

var (
	l7CompositeSources = []lmaelastic.AggCompositeSourceInfo{
		{Name: "dest_type", Field: "dest_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name_aggr", Field: "dest_name_aggr"},
		{Name: "dest_service_namespace", Field: "dest_service_namespace", Order: "desc"},
		{Name: "dest_service_name", Field: "dest_service_name"},
		{Name: "dest_service_port_name", Field: "dest_service_port_name"},
		{Name: "dest_service_port_num", Field: "dest_service_port_num"},
		{Name: "dest_port_num", Field: "dest_port_num"},
		{Name: "source_type", Field: "source_type"},
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "source_name_aggr", Field: "source_name_aggr"},
		{Name: "response_code", Field: "response_code"},
	}

	zeroGraphL7PacketStats = v1.GraphL7PacketStats{}
)

const (
	//TODO(rlb): We might want to abbreviate these to reduce the amount of data on the wire, json parsing and
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.
	flowL7AggSumBytesIn   = "sum_bytes_in"
	flowL7AggSumBytesOut  = "sum_bytes_out"
	flowL7AggSumCount     = "count"
	flowL7AggMinDuration  = "duration_min_mean"
	flowL7AggMaxDuration  = "duration_max"
	flowL7AggMeanDuration = "duration_mean"
)

var (
	l7AggregationSums = []lmaelastic.AggSumInfo{
		{Name: flowL7AggSumBytesIn, Field: "bytes_in"},
		{Name: flowL7AggSumBytesOut, Field: "bytes_out"},
		{Name: flowL7AggSumCount, Field: "count"},
	}
	l7AggregationMin = []lmaelastic.AggMaxMinInfo{
		{Name: flowL7AggMinDuration, Field: "duration_mean"},
	}
	l7AggregationMax = []lmaelastic.AggMaxMinInfo{
		{Name: flowL7AggMaxDuration, Field: "duration_max"},
	}
	l7AggregationMean = []lmaelastic.AggMeanInfo{
		{Name: flowL7AggMeanDuration, Field: "duration_mean"},
	}
)

// GetL7FlowData queries and returns the set of L7 flow data.
func GetL7FlowData(ctx context.Context, es lmaelastic.Client, cluster string, tr lmav1.TimeRange) ([]L7Flow, error) {
	// Track the total buckets queried and the response flows.
	var totalBuckets int
	var fs []L7Flow

	// Trace stats at debug level.
	if log.IsLevelEnabled(log.DebugLevel) {
		start := time.Now()
		log.Debug("GetL7FlowData called")
		defer func() {
			log.Debugf("GetL7FlowData took %s; buckets=%d; flows=%d", time.Since(start), totalBuckets, len(fs))
		}()
	}

	index := lmaindex.L7Logs().GetIndex(cluster)
	aggQueryL7 := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   lmaindex.L7Logs().NewTimeRangeQuery(tr.From, tr.To),
		Name:                    flowsBucketName,
		AggCompositeSourceInfos: l7CompositeSources,
		AggSumInfos:             l7AggregationSums,
		AggMinInfos:             l7AggregationMin,
		AggMaxInfos:             l7AggregationMax,
		AggMeanInfos:            l7AggregationMean,
	}

	addFlow := func(source, dest FlowEndpoint, svc v1.ServicePort, stats v1.GraphL7Stats) {
		if svc.Name != "" {
			fs = append(fs, L7Flow{
				Edge: FlowEdge{
					Source:      source,
					Dest:        dest,
					ServicePort: &svc,
				},
				Stats: stats,
			})
		} else {
			fs = append(fs, L7Flow{
				Edge: FlowEdge{
					Source: source,
					Dest:   dest,
				},
				Stats: stats,
			})
		}
		if log.IsLevelEnabled(log.DebugLevel) {
			if svc.Name != "" {
				log.Debugf("- Adding L7 flow: %s -> %s -> %s (stats %#v)", source, svc, dest, stats)
			} else {
				log.Debugf("- Adding L7 flow: %s -> %s (stats %#v)", source, dest, stats)
			}
		}
	}

	rcvdL7Buckets, rcvdL7Errors := es.SearchCompositeAggregations(ctx, aggQueryL7, nil)

	var foundFlow bool
	var l7Stats v1.GraphL7Stats
	var lastSource, lastDest FlowEndpoint
	var lastSvc v1.ServicePort
	for bucket := range rcvdL7Buckets {
		totalBuckets++
		key := bucket.CompositeAggregationKey
		code := key[l7ResponseCodeIdx].String()
		source := FlowEndpoint{
			Type:      mapRawTypeToGraphNodeType(key[l7SourceTypeIdx].String(), true),
			NameAggr:  singleDashToBlank(key[l7SourceNameAggrIdx].String()),
			Namespace: singleDashToBlank(key[l7SourceNamespaceIdx].String()),
		}
		svc := v1.ServicePort{
			NamespacedName: v1.NamespacedName{
				Name:      singleDashToBlank(key[l7DestServiceNameIdx].String()),
				Namespace: singleDashToBlank(key[l7DestServiceNamespaceIdx].String()),
			},
			Protocol: l7Proto,
			PortName: singleDashToBlank(key[l7DestServicePortNameIdx].String()),
			Port:     int(key[l7DestServicePortNumIdx].Float64()),
		}
		dest := FlowEndpoint{
			Type:      mapRawTypeToGraphNodeType(key[l7DestTypeIdx].String(), true),
			NameAggr:  singleDashToBlank(key[l7DestNameAggrIdx].String()),
			Namespace: singleDashToBlank(key[l7DestNamespaceIdx].String()),
			PortNum:   int(key[l7DestPortNumIdx].Float64()),
			Protocol:  l7Proto,
		}

		l7PacketStats := v1.GraphL7PacketStats{
			GraphByteStats: v1.GraphByteStats{
				BytesIn:  int64(bucket.AggregatedSums[flowL7AggSumBytesIn]),
				BytesOut: int64(bucket.AggregatedSums[flowL7AggSumBytesOut]),
			},
			MeanDuration: bucket.AggregatedMean[flowL7AggMeanDuration],
			MinDuration:  bucket.AggregatedMin[flowL7AggMinDuration],
			MaxDuration:  bucket.AggregatedMax[flowL7AggMaxDuration],
		}
		if l7PacketStats == zeroGraphL7PacketStats {
			log.Debugf("Skipping empty L7 flow: %s -> %s -> %s (code %s)", source, svc, dest, code)
			continue
		}
		// Now set the count (we couldn't do that before because we wanted to check the zero value).
		l7PacketStats.Count = int64(bucket.AggregatedSums[flowL7AggSumCount])

		if !foundFlow {
			// For the first entry we need to store off the first flow details.
			lastSource, lastDest, lastSvc = source, dest, svc
			foundFlow = true
		} else if lastSource != source || lastSvc != svc || lastDest != dest {
			addFlow(lastSource, lastDest, lastSvc, l7Stats)
			lastSource, lastDest, lastSvc, l7Stats = source, dest, svc, v1.GraphL7Stats{}
		}

		if log.IsLevelEnabled(log.DebugLevel) {
			if svc.Name != "" {
				log.Debugf("Processing L7 flow: %s -> %s -> %s (code %s)", source, svc, dest, code)
			} else {
				log.Debugf("Processing L7 flow: %s -> %s (code %s)", source, dest, code)
			}
		}

		if code_val, err := strconv.Atoi(code); err == nil && code_val >= 100 && code_val < 600 {
			if code_val < 200 {
				l7Stats.ResponseCode1xx = l7Stats.ResponseCode1xx.Combine(l7PacketStats)
			} else if code_val < 300 {
				l7Stats.ResponseCode2xx = l7Stats.ResponseCode2xx.Combine(l7PacketStats)
			} else if code_val < 400 {
				l7Stats.ResponseCode3xx = l7Stats.ResponseCode3xx.Combine(l7PacketStats)
			} else if code_val < 500 {
				l7Stats.ResponseCode4xx = l7Stats.ResponseCode4xx.Combine(l7PacketStats)
			} else {
				l7Stats.ResponseCode5xx = l7Stats.ResponseCode5xx.Combine(l7PacketStats)
			}
		} else {
			// Either not a number or not a valid response code.  Bucket in the no-response category.
			l7Stats.NoResponse = l7Stats.NoResponse.Combine(l7PacketStats)
		}
	}
	if foundFlow {
		addFlow(lastSource, lastDest, lastSvc, l7Stats)
	}

	return fs, <-rcvdL7Errors
}
