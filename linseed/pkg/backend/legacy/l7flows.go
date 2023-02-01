// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package legacy

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	kapiv1 "k8s.io/apimachinery/pkg/types"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	lmaindex "github.com/projectcalico/calico/lma/pkg/elastic/index"
)

// l7FlowBackend implements the Backend interface for flows stored
// in elasticsearch in the legacy storage model.
type l7FlowBackend struct {
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
}

func NewL7FlowBackend(c lmaelastic.Client) bapi.L7FlowBackend {
	// These are the keys which define an L7 in ES, and will be used to create buckets in the ES result.
	compositeSources := []lmaelastic.AggCompositeSourceInfo{
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

	return &l7FlowBackend{
		lmaclient: c,
		ft:        newFieldTracker(compositeSources),

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
	//           memory footprint.  Possibly a significant saving with large clusters or long time ranges.  These
	//           could be anything really as long as each is unique.
	flowL7AggSumBytesIn   = "sum_bytes_in"
	flowL7AggSumBytesOut  = "sum_bytes_out"
	flowL7AggSumCount     = "count"
	flowL7AggMinDuration  = "duration_min_mean"
	flowL7AggMaxDuration  = "duration_max"
	flowL7AggMeanDuration = "duration_mean"
)

// List returns all flows which match the given options.
func (b *l7FlowBackend) List(ctx context.Context, i bapi.ClusterInfo, opts v1.L7FlowParams) ([]v1.L7Flow, error) {
	log := contextLogger(i)

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID set on L7 flow request")
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

	// Build the aggregation request.
	query := &lmaelastic.CompositeAggregationQuery{
		DocumentIndex:           buildL7FlowsIndex(i.Cluster),
		Query:                   lmaindex.L7Logs().NewTimeRangeQuery(start, end),
		Name:                    "flows",
		AggCompositeSourceInfos: b.compositeSources,
		AggSumInfos:             b.aggSums,
		AggMaxInfos:             b.aggMaxs,
		AggMinInfos:             b.aggMins,
		AggMeanInfos:            b.aggMeans,
		MaxBucketsPerQuery:      opts.MaxResults,
	}

	// Context for the ES request.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Infof("Listing L7 flows from index %s", query.DocumentIndex)

	allFlows := []v1.L7Flow{}

	// Perform the request.
	// TODO: We're iterating over a channel. We need to support paging.
	buckets, errors := b.lmaclient.SearchCompositeAggregations(ctx, query, nil)
	for bucket := range buckets {
		log.Infof("Processing bucket built from %d logs", bucket.DocCount)
		key := bucket.CompositeAggregationKey

		f := v1.L7Flow{Key: v1.L7FlowKey{}}
		f.Key.Protocol = "tcp" // We only support TCP for now.
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
			continue
		}

		// Add the flow to the batch.
		allFlows = append(allFlows, f)

		// Track the number of flows. Bail if we hit the absolute maximum number of flows.
		if opts.MaxResults != 0 && len(allFlows) >= opts.MaxResults {
			log.Warnf("Maximum number of L7 flows (%d) reached. Stopping flow processing", opts.MaxResults)
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

func buildL7FlowsIndex(cluster string) string {
	return fmt.Sprintf("tigera_secure_ee_l7.%s", cluster)
}
