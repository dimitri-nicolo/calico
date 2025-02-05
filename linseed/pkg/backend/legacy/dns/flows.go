// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package dns

import (
	"context"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/backend"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/logtools"
	lmaindex "github.com/projectcalico/calico/linseed/pkg/internal/lma/elastic/index"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

const (
	// TODO(rlb): We might want to abbreviate these to reduce the amount of data on the wire, json parsing and
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.
	dnsAggSumCount         = "sum_count"
	dnsAggSumLatencyCount  = "sum_latency_count"
	dnsAggMeanLatencyCount = "mean_latency"
	dnsAggMaxLatencyCount  = "max_latency"
	dnsAggMinLatencyCount  = "min_latency"
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

	queryHelper lmaindex.Helper
	index       bapi.Index
}

func NewDNSFlowBackend(c lmaelastic.Client) bapi.DNSFlowBackend {
	return newFlowBackend(c, false)
}

func NewSingleIndexDNSFlowBackend(c lmaelastic.Client, options ...index.Option) bapi.DNSFlowBackend {
	return newFlowBackend(c, true, options...)
}

func newFlowBackend(c lmaelastic.Client, singleIndex bool, options ...index.Option) bapi.DNSFlowBackend {
	// These are the keys which define a DNS flow in ES, and will be used to create buckets in the ES result.
	compositeSources := []lmaelastic.AggCompositeSourceInfo{
		{Name: "cluster", Field: "cluster"},
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

	indexTemplate := index.DNSLogIndex(options...)
	helper := lmaindex.SingleIndexDNSLogs()
	if !singleIndex {
		indexTemplate = index.DNSLogMultiIndex
		helper = lmaindex.MultiIndexDNSLogs()
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

		queryHelper: helper,
		index:       indexTemplate,
	}
}

// List returns all flows which match the given options.
func (b *dnsFlowBackend) List(ctx context.Context, i bapi.ClusterInfo, opts *v1.DNSFlowParams) (*v1.List[v1.DNSFlow], error) {
	log := bapi.ContextLogger(i)

	if err := i.Valid(); err != nil {
		return nil, err
	}

	// Build the aggregation request.
	q, err := b.buildQuery(i, opts)
	if err != nil {
		return nil, err
	}

	query := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           b.index.Index(i),
		Query:                   q,
		Name:                    "buckets",
		AggCompositeSourceInfos: b.compositeSources,
		AggSumInfos:             b.aggSums,
		AggMaxInfos:             b.aggMaxs,
		AggMinInfos:             b.aggMins,
		AggMeanInfos:            b.aggMeans,
		AggNestedTermInfos:      b.aggNested,
		MaxBucketsPerQuery:      opts.GetMaxPageSize(),
	}
	log.Infof("Listing DNS flows from index %s", query.DocumentIndex)

	// Perform the request.
	page, key, err := lmaelastic.PagedSearch(ctx, b.lmaclient, query, log, b.convertBucket, opts.AfterKey)
	return &v1.List[v1.DNSFlow]{
		Items:    page,
		AfterKey: key,
	}, err
}

// convertBucket turns a composite aggregation bucket into an DNSFlow.
func (b *dnsFlowBackend) convertBucket(log *logrus.Entry, bucket *lmaelastic.CompositeAggregationBucket) *v1.DNSFlow {
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
		Cluster:      b.ft.ValueString(key, "cluster"),
	}
	flow.LatencyStats = &v1.DNSLatencyStats{
		MeanRequestLatency: bucket.AggregatedMean[dnsAggMeanLatencyCount],
		MinRequestLatency:  bucket.AggregatedMin[dnsAggMinLatencyCount],
		MaxRequestLatency:  bucket.AggregatedMax[dnsAggMaxLatencyCount],
		LatencyCount:       int(bucket.AggregatedSums[dnsAggSumLatencyCount]),
	}
	flow.Count = int64(bucket.AggregatedSums[dnsAggSumCount])
	return &flow
}

// buildQuery builds an elastic query using the given parameters.
func (b *dnsFlowBackend) buildQuery(i bapi.ClusterInfo, opts *v1.DNSFlowParams) (elastic.Query, error) {
	query, err := b.queryHelper.BaseQuery(i, opts)
	if err != nil {
		return nil, err
	}

	// Parse times from the request.
	query.Filter(b.queryHelper.NewTimeRangeQuery(
		logtools.WithDefaultLast5Minutes(opts.TimeRange),
	))

	return query, nil
}
