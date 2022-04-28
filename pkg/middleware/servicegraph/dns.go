// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"

	log "github.com/sirupsen/logrus"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	elasticvariant "github.com/tigera/es-proxy/pkg/elastic"
)

type DNSLog struct {
	Endpoint FlowEndpoint
	Stats    v1.GraphDNSStats
}

const (
	dnsBucketName = "dns"
)

const (
	dnsClientNamespaceIdx = iota
	dnsClientNameAggrIdx
	dnsRcodeIdx
)

var (
	dnsCompositeSources = []lmaelastic.AggCompositeSourceInfo{
		{Name: "client_namespace", Field: "client_namespace"},
		{Name: "client_name_aggr", Field: "client_name_aggr"},
		{Name: "rcode", Field: "rcode"},
	}
)

const (
	//TODO(rlb): We might want to abbreviate these to reduce the amount of data on the wire, json parsing and
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.
	dnsAggSumCount         = "sum_count"
	dnsAggSumLatencyCount  = "sum_latency_count"
	dnsAggMeanLatencyCount = "mean_latency"
	dnsAggMaxLatencyCount  = "max_latency"
	dnsAggMinLatencyCount  = "min_latency"
)

var (
	dnsAggregationSums = []lmaelastic.AggSumInfo{
		{Name: dnsAggSumCount, Field: "count"},
		{Name: dnsAggSumLatencyCount, Field: "latency_count"},
	}
	dnsAggregationMin = []lmaelastic.AggMaxMinInfo{
		{Name: dnsAggMinLatencyCount, Field: "latency_mean"},
	}
	dnsAggregationMax = []lmaelastic.AggMaxMinInfo{
		{Name: dnsAggMaxLatencyCount, Field: "latency_max"},
	}
	dnsAggregationMean = []lmaelastic.AggMeanInfo{
		{Name: dnsAggMeanLatencyCount, Field: "latency_mean", WeightField: "latency_count"},
	}
)

// GetDNSClientData queries and returns the set of DNS logs.
func GetDNSClientData(
	ctx context.Context, es lmaelastic.Client, cluster string, tr lmav1.TimeRange,
	cfg *Config,
) (logs []DNSLog, err error) {
	// Trace progress.
	progress := newElasticProgress("dns", tr)
	defer func() {
		progress.Complete(err)
	}()

	index := lmaindex.DnsLogs().GetIndex(elasticvariant.AddIndexInfix(cluster))
	aggQueryL7 := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           index,
		Query:                   lmaindex.DnsLogs().NewTimeRangeQuery(tr.From, tr.To),
		Name:                    dnsBucketName,
		AggCompositeSourceInfos: dnsCompositeSources,
		AggSumInfos:             dnsAggregationSums,
		AggMinInfos:             dnsAggregationMin,
		AggMaxInfos:             dnsAggregationMax,
		AggMeanInfos:            dnsAggregationMean,
		MaxBucketsPerQuery:      cfg.ServiceGraphCacheMaxBucketsPerQuery,
	}

	addLog := func(source FlowEndpoint, stats *v1.GraphDNSStats) {
		logs = append(logs, DNSLog{
			Endpoint: source,
			Stats:    *stats,
		})
		progress.IncAggregated()
	}

	// Perform the DNS composite aggregation query.
	// Always ensure we cancel the query if we bail early.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	rcvdDNSBuckets, rcvdDNSErrors := es.SearchCompositeAggregations(ctx, aggQueryL7, nil)

	var foundLog bool
	var dnsStats *v1.GraphDNSStats
	var lastSource FlowEndpoint
	for bucket := range rcvdDNSBuckets {
		progress.IncRaw()
		key := bucket.CompositeAggregationKey
		code := key[dnsRcodeIdx].String()
		source := FlowEndpoint{
			Type:      v1.GraphNodeTypeReplicaSet,
			NameAggr:  singleDashToBlank(key[dnsClientNameAggrIdx].String()),
			Namespace: singleDashToBlank(key[dnsClientNamespaceIdx].String()),
		}

		if !foundLog {
			// For the first entry we need to store off the first flow details.
			lastSource = source
			foundLog = true
		} else if lastSource != source {
			addLog(lastSource, dnsStats)
			lastSource, dnsStats = source, nil
		}

		gls := v1.GraphLatencyStats{
			MeanRequestLatency: bucket.AggregatedMean[dnsAggMeanLatencyCount],
			MinRequestLatency:  bucket.AggregatedMin[dnsAggMinLatencyCount],
			MaxRequestLatency:  bucket.AggregatedMax[dnsAggMaxLatencyCount],
			LatencyCount:       int64(bucket.AggregatedSums[dnsAggSumLatencyCount]),
		}
		dnsStats = dnsStats.Combine(&v1.GraphDNSStats{
			GraphLatencyStats: gls,
			ResponseCodes: map[string]v1.GraphDNSResponseCode{
				code: {
					Code:              code,
					Count:             int64(bucket.AggregatedSums[dnsAggSumCount]),
					GraphLatencyStats: gls,
				},
			},
		})

		if log.IsLevelEnabled(log.DebugLevel) {
			log.Debugf("Processing DNS Log: %s (code %s)", source, code)
		}

		// Track the number of aggregated logs. Bail if we hit the absolute maximum number of aggregated logs.
		if len(logs) > cfg.ServiceGraphCacheMaxAggregatedRecords {
			return logs, DataTruncatedError
		}
	}
	if foundLog {
		addLog(lastSource, dnsStats)
	}

	return logs, <-rcvdDNSErrors
}
