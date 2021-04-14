package servicegraph

import (
	"context"
	"strconv"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/middleware/flows"
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
	//l7DestServicePortIdx
	//l7ProtoIdx
	//l7DestPortIdx
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
		{Name: "source_type", Field: "src_type"},
		{Name: "source_namespace", Field: "src_namespace"},
		{Name: "source_name_aggr", Field: "src_name_aggr"},
		{Name: "response_code", Field: "response_code"},
	}
)

var (
	l7AggregationSums = []lmaelastic.AggSumInfo{
		{Name: "sum_bytes_in", Field: "bytes_in"},
		{Name: "sum_bytes_out", Field: "bytes_out"},
		{Name: "count", Field: "count"},
	}
	l7AggregationMax = []lmaelastic.AggMaxMinInfo{
		{Name: "duration_max", Field: "duration_max"},
	}
	l7AggregationMean = []lmaelastic.AggMeanInfo{
		{Name: "duration_mean", Field: "duration_mean"},
	}
)

func GetRawL7FlowData(client lmaelastic.Client, index string, t v1.TimeRange) ([]L7Flow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), flowTimeout)
	defer cancel()

	aggQueryL7 := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   flows.GetTimeRangeQuery(t),
		Name:                    flowsBucketName,
		AggCompositeSourceInfos: l7CompositeSources,
		AggSumInfos:             l7AggregationSums,
		AggMaxInfos:             l7AggregationMax,
		AggMeanInfos:            l7AggregationMean,
	}

	// Perform the L3 and L7 composite aggregation queries together.
	var fs []L7Flow
	addFlow := func(source, dest FlowEndpoint, svc ServicePort, stats v1.GraphL7Stats) {
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
				log.Debugf("- Adding L7 flow: %s -> %s (code %#v)", source, dest, stats)
			}
		}
	}

	rcvdL7Buckets, rcvdL7Errors := client.SearchCompositeAggregations(ctx, aggQueryL7, nil)

	var foundFlow bool
	var l7Stats v1.GraphL7Stats
	var lastSource, lastDest FlowEndpoint
	var lastSvc ServicePort
	for bucket := range rcvdL7Buckets {
		key := bucket.CompositeAggregationKey
		code := key[l7ResponseCodeIdx].String()
		source := FlowEndpoint{
			Type:      mapType(key[l7SourceTypeIdx].String(), true),
			NameAggr:  removeSingleDash(key[l7SourceNameAggrIdx].String()),
			Namespace: removeSingleDash(key[l7SourceNamespaceIdx].String()),
		}
		svc := ServicePort{
			NamespacedName: types.NamespacedName{
				Name:      removeSingleDash(key[l7DestServiceNameIdx].String()),
				Namespace: removeSingleDash(key[l7DestServiceNamespaceIdx].String()),
			},
			//Port:  removeSingleDash(key[flowDestServicePortIdx].String()),
			Proto: l7Proto,
		}
		dest := FlowEndpoint{
			Type:      mapType(key[l7DestTypeIdx].String(), true),
			NameAggr:  removeSingleDash(key[l7DestNameAggrIdx].String()),
			Namespace: removeSingleDash(key[l7DestNamespaceIdx].String()),
			//Port:      int(key[l7DestPortIdx].Float64()),
			Proto: l7Proto,
		}

		if !foundFlow {
			// For the first entry we need to store off the first flow details.
			lastSource, lastDest, lastSvc = source, dest, svc
			foundFlow = true
		} else if lastSource != source || lastSvc != svc || lastDest != dest {
			addFlow(lastSource, lastDest, lastSvc, l7Stats)
			lastSource, lastDest, lastSvc, l7Stats = source, dest, svc, v1.GraphL7Stats{}
		}

		l7PacketStats := v1.GraphL7PacketStats{
			GraphPacketStats: v1.GraphPacketStats{
				BytesIn:  int64(bucket.AggregatedSums["sum_bytes_in"]),
				BytesOut: int64(bucket.AggregatedSums["sum_bytes_out"]),
			},
			MeanDuration: bucket.AggregatedSums["duration_mean"],
			MaxDuration:  bucket.AggregatedSums["duration_max"],
			Count:        int64(bucket.AggregatedSums["count"]),
		}

		if log.IsLevelEnabled(log.DebugLevel) {
			if svc.Name != "" {
				log.Debugf("Processing L7 flow: %s -> %s -> %s (code %s)", source, svc, dest, code)
			} else {
				log.Debugf("Processing L7 flow: %s -> %s (code %s)", source, dest, code)
			}
		}

		if code_val, err := strconv.Atoi(code); err == nil {
			if code_val < 200 {
				l7Stats.ResponseCode1xx = l7Stats.ResponseCode1xx.Add(l7PacketStats)
			} else if code_val < 300 {
				l7Stats.ResponseCode2xx = l7Stats.ResponseCode2xx.Add(l7PacketStats)
			} else if code_val < 400 {
				l7Stats.ResponseCode3xx = l7Stats.ResponseCode3xx.Add(l7PacketStats)
			} else if code_val < 500 {
				l7Stats.ResponseCode4xx = l7Stats.ResponseCode4xx.Add(l7PacketStats)
			} else {
				l7Stats.ResponseCode5xx = l7Stats.ResponseCode5xx.Add(l7PacketStats)
			}
		}

		lastSource, lastDest, lastSvc = source, dest, svc
	}
	if foundFlow {
		addFlow(lastSource, lastDest, lastSvc, l7Stats)
	}

	return fs, <-rcvdL7Errors
}
