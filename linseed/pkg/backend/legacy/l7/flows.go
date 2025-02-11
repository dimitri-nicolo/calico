// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package l7

import (
	"context"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	kapiv1 "k8s.io/apimachinery/pkg/types"

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
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.  These
	//           could be anything really as long as each is unique.
	flowL7AggSumBytesIn   = "sum_bytes_in"
	flowL7AggSumBytesOut  = "sum_bytes_out"
	flowL7AggSumCount     = "count"
	flowL7AggMinDuration  = "duration_min_mean"
	flowL7AggMaxDuration  = "duration_max"
	flowL7AggMeanDuration = "duration_mean"
)

// l7FlowBackend implements the Backend interface for flows stored
// in elasticsearch in the legacy storage model.
type l7FlowBackend struct {
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

	queryHelper lmaindex.Helper
	index       bapi.Index
}

func NewL7FlowBackend(c lmaelastic.Client) bapi.L7FlowBackend {
	return newBackend(c, false)
}

func NewSingleIndexL7FlowBackend(c lmaelastic.Client, options ...index.Option) bapi.L7FlowBackend {
	return newBackend(c, true, options...)
}

func newBackend(c lmaelastic.Client, singleIndex bool, options ...index.Option) bapi.L7FlowBackend {
	// These are the keys which define an L7 in ES, and will be used to create buckets in the ES result.
	compositeSources := []lmaelastic.AggCompositeSourceInfo{
		{Name: "cluster", Field: "cluster"},
		{Name: "dest_type", Field: "dest_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name_aggr", Field: "dest_name_aggr"},
		{Name: "dest_service_namespace", Field: "dest_service_namespace", Order: "desc"},
		{Name: "dest_service_name", Field: "dest_service_name"},
		{Name: "dest_service_port_name", Field: "dest_service_port_name", AllowMissingBucket: true},
		{Name: "dest_service_port_num", Field: "dest_service_port"},
		{Name: "dest_port_num", Field: "dest_port_num", AllowMissingBucket: true},
		{Name: "source_type", Field: "src_type"},
		{Name: "source_namespace", Field: "src_namespace"},
		{Name: "source_name_aggr", Field: "src_name_aggr"},
		{Name: "response_code", Field: "response_code"},
	}

	sums := []lmaelastic.AggSumInfo{
		{Name: flowL7AggSumBytesIn, Field: "bytes_in"},
		{Name: flowL7AggSumBytesOut, Field: "bytes_out"},
		{Name: flowL7AggSumCount, Field: "count"},
	}
	mins := []lmaelastic.AggMaxMinInfo{
		{Name: flowL7AggMinDuration, Field: "duration_mean"},
	}
	maxs := []lmaelastic.AggMaxMinInfo{
		{Name: flowL7AggMaxDuration, Field: "duration_max"},
	}
	means := []lmaelastic.AggMeanInfo{
		{Name: flowL7AggMeanDuration, Field: "duration_mean"},
	}

	helper := lmaindex.MultiIndexL7Logs()
	idx := index.L7LogMultiIndex
	if singleIndex {
		helper = lmaindex.SingleIndexL7Logs()
		idx = index.L7LogIndex(options...)
	}

	return &l7FlowBackend{
		lmaclient: c,
		ft:        backend.NewFieldTracker(compositeSources),

		// Configuration for the aggregation queries we make against ES.
		compositeSources: compositeSources,
		aggSums:          sums,
		aggMins:          mins,
		aggMaxs:          maxs,
		aggMeans:         means,

		queryHelper: helper,
		index:       idx,
	}
}

// List returns all flows which match the given options.
func (b *l7FlowBackend) List(ctx context.Context, i bapi.ClusterInfo, opts *v1.L7FlowParams) (*v1.List[v1.L7Flow], error) {
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
		Name:                    "flows",
		AggCompositeSourceInfos: b.compositeSources,
		AggSumInfos:             b.aggSums,
		AggMaxInfos:             b.aggMaxs,
		AggMinInfos:             b.aggMins,
		AggMeanInfos:            b.aggMeans,
		MaxBucketsPerQuery:      opts.GetMaxPageSize(),
	}

	log.Debugf("Listing flows from index %s", query.DocumentIndex)

	// Perform the request.
	page, key, err := lmaelastic.PagedSearch(ctx, b.lmaclient, query, log, b.convertBucket, opts.AfterKey)
	return &v1.List[v1.L7Flow]{
		Items:    page,
		AfterKey: key,
	}, err
}

func (b *l7FlowBackend) convertBucket(log *logrus.Entry, bucket *lmaelastic.CompositeAggregationBucket) *v1.L7Flow {
	key := bucket.CompositeAggregationKey

	f := v1.L7Flow{Key: v1.L7FlowKey{}}
	f.Key.Protocol = "tcp" // We only support TCP for now.
	f.Key.Cluster = b.ft.ValueString(key, "cluster")
	f.Key.Source = v1.Endpoint{
		Type:           v1.EndpointType(b.ft.ValueString(key, "src_type")),
		AggregatedName: b.ft.ValueString(key, "src_name_aggr"),
		Namespace:      b.ft.ValueString(key, "src_namespace"),
	}
	f.Key.Destination = v1.Endpoint{
		Type:           v1.EndpointType(b.ft.ValueString(key, "dest_type")),
		AggregatedName: b.ft.ValueString(key, "dest_name_aggr"),
		Namespace:      b.ft.ValueString(key, "dest_namespace"),
		Port:           b.ft.ValueInt64(key, "dest_port_num"),
	}
	f.Key.DestinationService = v1.ServicePort{
		Service: kapiv1.NamespacedName{
			Name:      b.ft.ValueString(key, "dest_service_name"),
			Namespace: b.ft.ValueString(key, "dest_service_namespace"),
		},
		PortName: b.ft.ValueString(key, "dest_service_port_name"),
		Port:     b.ft.ValueInt64(key, "dest_service_port"),
	}
	f.Stats = &v1.L7Stats{
		BytesIn:      int64(bucket.AggregatedSums[flowL7AggSumBytesIn]),
		BytesOut:     int64(bucket.AggregatedSums[flowL7AggSumBytesOut]),
		MeanDuration: int64(bucket.AggregatedMean[flowL7AggMeanDuration]),
		MinDuration:  int64(bucket.AggregatedMin[flowL7AggMinDuration]),
		MaxDuration:  int64(bucket.AggregatedMax[flowL7AggMaxDuration]),
	}
	f.Code = b.ft.ValueInt32(key, "response_code")
	f.LogCount = int64(bucket.AggregatedSums[flowL7AggSumCount])

	// Check if we have any stats to report. If not, we skip this flow.
	if (*f.Stats == v1.L7Stats{}) {
		f := logrus.Fields{
			"src":  f.Key.Source,
			"dest": f.Key.Source,
			"svc":  f.Key.DestinationService,
			"code": f.Code,
		}
		log.WithFields(f).Debugf("Skipping empty L7 flow")
		return nil
	}
	return &f
}

// buildQuery builds an elastic query using the given parameters.
func (b *l7FlowBackend) buildQuery(i bapi.ClusterInfo, opts *v1.L7FlowParams) (elastic.Query, error) {
	query, err := b.queryHelper.BaseQuery(i, opts)
	if err != nil {
		return nil, err
	}

	query.Must(b.queryHelper.NewTimeRangeQuery(
		logtools.WithDefaultUntilNow(opts.TimeRange),
	))
	return query, nil
}
