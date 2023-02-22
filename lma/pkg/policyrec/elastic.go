// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.
package policyrec

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	lapi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/lma/pkg/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	pelastic "github.com/projectcalico/calico/lma/pkg/elastic"
	"github.com/projectcalico/calico/lma/pkg/timeutils"
)

const (
	FlowlogBuckets = "flog_buckets"
)

var (
	CompositeSources = []pelastic.AggCompositeSourceInfo{
		{Name: "source_type", Field: "source_type"},
		{Name: "source_namespace", Field: "source_namespace"},
		{Name: "source_name_aggr", Field: "source_name_aggr"},
		{Name: "dest_type", Field: "dest_type"},
		{Name: "dest_namespace", Field: "dest_namespace"},
		{Name: "dest_name_aggr", Field: "dest_name_aggr"},
		{Name: "proto", Field: "proto"},
		{Name: "dest_ip", Field: "dest_ip"},
		{Name: "source_ip", Field: "source_ip"},
		{Name: "source_port", Field: "source_port"},
		{Name: "dest_port", Field: "dest_port"},
		{Name: "reporter", Field: "reporter"},
		{Name: "action", Field: "action"},
	}
	AggregatedTerms = []pelastic.AggNestedTermInfo{
		{Name: "policies", Path: "policies", Term: "by_tiered_policy", Field: "policies.all_policies"},
		{Name: "source_labels", Path: "source_labels", Term: "by_kvpair", Field: "source_labels.labels"},
		{Name: "dest_labels", Path: "dest_labels", Term: "by_kvpair", Field: "dest_labels.labels"},
	}

	// Indexes for policy recommendation into the raw flow data
	PRCompositeSourcesRawIdxSourceType      = 0
	PRCompositeSourcesRawIdxSourceNamespace = 1
	PRCompositeSourcesRawIdxSourceNameAggr  = 2
	PRCompositeSourcesRawIdxDestType        = 3
	PRCompositeSourcesRawIdxDestNamespace   = 4
	PRCompositeSourcesRawIdxDestNameAggr    = 5
	PRCompositeSourcesRawIdxProto           = 6
	PRCompositeSourcesRawIdxDestIP          = 7
	PRCompositeSourcesRawIdxSourceIP        = 8
	PRCompositeSourcesRawIdxSourcePort      = 9
	PRCompositeSourcesRawIdxDestPort        = 10
	PRCompositeSourcesRawIdxReporter        = 11
	PRCompositeSourcesRawIdxAction          = 12

	PRCompositeSourcesIdxs map[string]int = map[string]int{
		"source_type":      PRCompositeSourcesRawIdxSourceType,
		"source_namespace": PRCompositeSourcesRawIdxSourceNamespace,
		"source_name_aggr": PRCompositeSourcesRawIdxSourceNameAggr,
		"dest_type":        PRCompositeSourcesRawIdxDestType,
		"dest_namespace":   PRCompositeSourcesRawIdxDestNamespace,
		"dest_name_aggr":   PRCompositeSourcesRawIdxDestNameAggr,
		"proto":            PRCompositeSourcesRawIdxProto,
		"dest_ip":          PRCompositeSourcesRawIdxDestIP,
		"source_ip":        PRCompositeSourcesRawIdxSourceIP,
		"source_port":      PRCompositeSourcesRawIdxSourcePort,
		"dest_port":        PRCompositeSourcesRawIdxDestPort,
		"reporter":         PRCompositeSourcesRawIdxReporter,
		"action":           PRCompositeSourcesRawIdxAction,
		"source_name":      PRCompositeSourcesRawIdxSourceNameAggr,
		"dest_name":        PRCompositeSourcesRawIdxDestNameAggr,
	}

	PRAggregatedTermsNamePolicies     = "policies"
	PRAggregatedTermsNameSourceLabels = "source_labels"
	PRAggregatedTermsNameDestLabels   = "dest_labels"

	PRAggregatedTermNames map[string]string = map[string]string{
		"policies":      PRAggregatedTermsNamePolicies,
		"source_labels": PRAggregatedTermsNameSourceLabels,
		"dest_labels":   PRAggregatedTermsNameDestLabels,
	}
)

func BuildQuery(params *PolicyRecommendationParams) *lapi.L3FlowParams {
	// Parse the start and end times.
	now := time.Now()
	start, _, err := timeutils.ParseTime(now, &params.StartTime)
	if err != nil {
		logrus.WithError(err).Warning("Failed to parse start time")
	}

	end, _, err := timeutils.ParseTime(now, &params.EndTime)
	if err != nil {
		logrus.WithError(err).Warning("Failed to parse end time")
	}

	fp := lapi.L3FlowParams{}
	fp.TimeRange = &lmav1.TimeRange{}
	if start != nil {
		fp.TimeRange.From = *start
	}
	if end != nil {
		fp.TimeRange.To = *end
	}

	fp.SourceTypes = []lapi.EndpointType{lapi.Network, lapi.NetworkSet, lapi.WEP, lapi.HEP}
	fp.DestinationTypes = []lapi.EndpointType{lapi.Network, lapi.NetworkSet, lapi.WEP, lapi.HEP}
	if params.Namespace != "" {
		fp.NamespaceMatches = []lapi.NamespaceMatch{
			{Type: lapi.MatchTypeAny, Namespaces: []string{params.Namespace}},
		}
	}
	if params.EndpointName != "" {
		fp.NameAggrMatches = []lapi.NameMatch{
			{Type: lapi.MatchTypeAny, Names: []string{params.EndpointName}},
		}
	}

	// If the request is only for unprotected flows then return a query that will
	// specifically only pick flows that are allowed by a profile.
	allow := lapi.FlowActionAllow
	if params.Unprotected {
		fp.PolicyMatches = []lapi.PolicyMatch{
			{
				Tier:   "__PROFILE__",
				Action: &allow,
			},
		}
	} else {
		// Otherwise, return any flows that are seen by the default tier
		// or allowed by a profile.
		fp.PolicyMatches = []lapi.PolicyMatch{
			{
				Tier: "default",
			},
			{
				Tier:   "__PROFILE__",
				Action: &allow,
			},
		}
	}
	return &fp
}

// CompositeAggregator is an interface to provide composite aggregation via Elasticsearch.
type CompositeAggregator interface {
	SearchCompositeAggregations(
		context.Context, *pelastic.CompositeAggregationQuery, pelastic.CompositeAggregationKey,
	) (<-chan *pelastic.CompositeAggregationBucket, <-chan error)
}

func SearchFlows(ctx context.Context, listFn client.ListFunc[lapi.L3Flow], pager client.ListPager[lapi.L3Flow]) ([]*api.Flow, error) {
	// Search for the raw data in ES.
	pages, errors := pager.Stream(ctx, listFn)

	flows := []*api.Flow{}
	for page := range pages {
		for _, f := range page.Items {
			flow := api.FromLinseedFlow(f)
			if flow != nil {
				flows = append(flows, flow)
			}
		}
	}

	if err, ok := <-errors; ok {
		log.WithError(err).Warning("Hit error processing flow logs")
		return flows, err
	}

	return flows, nil
}
