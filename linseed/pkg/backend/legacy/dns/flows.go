// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package dns

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

// dnsFlowBackend implements the Backend interface for flows stored
// in elasticsearch in the legacy storage model.
type dnsFlowBackend struct {
	// Elasticsearch client.
	lmaclient lmaelastic.Client

	// Track mapping of field name to its index in the ES response.
	ft *backend.FieldTracker

	// The sources and aggregations to use when building an aggregation query against ES.
	compositeSources []lmaelastic.AggCompositeSourceInfo
	aggSums          []lmaelastic.AggSumInfo
	aggMins          []lmaelastic.AggMaxMinInfo
	aggMaxs          []lmaelastic.AggMaxMinInfo
	aggMeans         []lmaelastic.AggMeanInfo
	aggNested        []lmaelastic.AggNestedTermInfo
}

func NewDNSFlowBackend(c lmaelastic.Client) bapi.DNSFlowBackend {
	// These are the keys which define a DNS flow in ES, and will be used to create buckets in the ES result.
	compositeSources := []lmaelastic.AggCompositeSourceInfo{
		{Name: "client_namespace", Field: "client_namespace"},
		{Name: "client_name_aggr", Field: "client_name_aggr"},
		{Name: "rcode", Field: "rcode"},
	}

	sums := []lmaelastic.AggSumInfo{
		{Name: dnsAggSumCount, Field: "count"},
		{Name: dnsAggSumLatencyCount, Field: "latency_count"},
	}
	mins := []lmaelastic.AggMaxMinInfo{
		{Name: dnsAggMinLatencyCount, Field: "latency_mean"},
	}
	maxs := []lmaelastic.AggMaxMinInfo{
		{Name: dnsAggMaxLatencyCount, Field: "latency_max"},
	}
	means := []lmaelastic.AggMeanInfo{
		{Name: dnsAggMeanLatencyCount, Field: "latency_mean", WeightField: "latency_count"},
	}

	return &dnsFlowBackend{
		lmaclient: c,
		ft:        backend.NewFieldTracker(compositeSources),

		// Configuration for the aggregation queries we make against ES.
		compositeSources: compositeSources,
		aggSums:          sums,
		aggMins:          mins,
		aggMaxs:          maxs,
		aggMeans:         means,
	}
}

const (
	// TODO(rlb): We might want to abbreviate these to reduce the amount of data on the wire, json parsing and
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.
	dnsAggSumCount         = "sum_count"
	dnsAggSumLatencyCount  = "sum_latency_count"
	dnsAggMeanLatencyCount = "mean_latency"
	dnsAggMaxLatencyCount  = "max_latency"
	dnsAggMinLatencyCount  = "min_latency"
)

// List returns all flows which match the given options.
func (b *dnsFlowBackend) List(ctx context.Context, i bapi.ClusterInfo, opts v1.DNSFlowParams) ([]v1.DNSFlow, error) {
	log := bapi.ContextLogger(i)

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
		DocumentIndex:           buildDNSIndex(i.Cluster),
		Query:                   lmaindex.DnsLogs().NewTimeRangeQuery(start, end),
		Name:                    "dns_buckets",
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

	log.Infof("Listing DNS flows from index %s", query.DocumentIndex)

	var allFlows []v1.DNSFlow

	// Perform the request.
	// TODO: We're iterating over a channel. We need to support paging.
	buckets, errors := b.lmaclient.SearchCompositeAggregations(ctx, query, nil)
	for bucket := range buckets {
		log.Infof("Processing bucket built from %d logs", bucket.DocCount)
		key := bucket.CompositeAggregationKey

		// Build the flow, starting with the key.
		flow := v1.DNSFlow{}
		flow.Key = v1.DNSFlowKey{
			Source: v1.Endpoint{
				// We only collect logs from workload endpoints.
				Type:           "wep",
				AggregatedName: b.ft.ValueString(key, "client_name_aggr"),
				Namespace:      b.ft.ValueString(key, "client_namespace"),
			},
			ResponseCode: b.ft.ValueString(key, "rcode"),
		}
		flow.LatencyStats = &v1.DNSLatencyStats{
			MeanRequestLatency: bucket.AggregatedMean[dnsAggMeanLatencyCount],
			MinRequestLatency:  bucket.AggregatedMin[dnsAggMinLatencyCount],
			MaxRequestLatency:  bucket.AggregatedMax[dnsAggMaxLatencyCount],
			LatencyCount:       int(bucket.AggregatedSums[dnsAggSumLatencyCount]),
		}
		flow.Count = int64(bucket.AggregatedSums[dnsAggSumCount])

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

	return allFlows, nil
}

func buildDNSIndex(cluster string) string {
	return fmt.Sprintf("tigera_secure_ee_dns.%s.*", cluster)
}
