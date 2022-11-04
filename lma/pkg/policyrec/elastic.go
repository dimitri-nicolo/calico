// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.
package policyrec

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	elastic "github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/lma/pkg/api"
	pelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

const defaultTier = "default"

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

/*{
  "bool": {
    "must": [
      {"range": {"start_time": { "gte": "{{.StartTime}}"}}},
      {"range": {"end_time": { "lte": "{{.EndTime}}"}}},
      {"terms":{"source_type":["net","ns","wep","hep"]}},
      {"terms":{"dest_type":["net","ns","wep","hep"]}},
      {"nested": {
        "path": "policies",
        "query": {
          "wildcard": {
            "policies.all_policies": {
              "value": "*|__PROFILE__|__PROFILE__.kns.{{.Namespace}}|allow"
            }
          }
        }
      }},
      {"bool": {
        "should": [
          {"bool": {
						"must": [
              {"term": {"source_namespace": "{{.Namespace}}"}}
            ]
						OR
            "must": [
              {"term": {"source_name_aggr": "{{.EndpointName}}"}},
              {"term": {"source_namespace": "{{.Namespace}}"}}
            ]
          }},
          {"bool": {
						"must": [
              {"term": {"dest_namespace": "{{.Namespace}}"}}
            ]
						OR
            "must": [
              {"term": {"dest_name_aggr": "{{.EndpointName}}"}},
              {"term": {"dest_namespace": "{{.Namespace}}"}}
            ]
          }}
        ]
      }}
    ]
  }
}*/
func BuildElasticQuery(params *PolicyRecommendationParams) elastic.Query {
	query := elastic.NewBoolQuery()

	startQuery := elastic.NewRangeQuery("start_time").Gte(params.StartTime)
	endQuery := elastic.NewRangeQuery("end_time").Lte(params.EndTime)
	sourceTermsQuery := elastic.NewTermsQuery("source_type", "net", "ns", "wep", "hep")
	destTermsQuery := elastic.NewTermsQuery("dest_type", "net", "ns", "wep", "hep")

	nameAndNamespaceQuery := buildNameOrNamespaceQuery(params.Namespace, params.EndpointName)

	unprotectedWildcardQuery := buildUnprotectedQuery(params.Namespace)

	// If the request is only for unprotected flows then return a query that will
	// specifically only pick flows that are allowed by a profile.
	if params.Unprotected {
		unprotectedQuery := elastic.NewNestedQuery("policies", unprotectedWildcardQuery)
		return query.Must(
			startQuery,
			endQuery,
			sourceTermsQuery,
			destTermsQuery,
			unprotectedQuery,
			nameAndNamespaceQuery,
		)
	}

	// Otherwise fetch all flows seen (allow, deny, and pass) by the default tier
	// and allowed by profiles.
	defaultTierWildcardQuery := buildTierQuery(defaultTier)

	matchingTiers := elastic.NewBoolQuery()
	matchingTiers.Should(defaultTierWildcardQuery, unprotectedWildcardQuery)
	nestedTiersQuery := elastic.NewNestedQuery("policies", matchingTiers)

	return query.Must(
		startQuery,
		endQuery,
		nestedTiersQuery,
		nameAndNamespaceQuery,
	)
}

// buildTierQuery builds a wildcarded nested query that will match a policy hit in the
// default tier.
func buildTierQuery(tierName string) elastic.Query {
	tier := fmt.Sprintf("*|%s|*|*", tierName)
	return elastic.NewWildcardQuery("policies.all_policies", tier)
}

func buildUnprotectedQuery(namespace string) elastic.Query {
	namespaceProfile := fmt.Sprintf("*|__PROFILE__|__PROFILE__.kns.%s|allow*", namespace)
	return elastic.NewWildcardQuery("policies.all_policies", namespaceProfile)
}

func buildNameOrNamespaceQuery(namespace, name string) elastic.Query {
	nameOrNamespaceQuery := elastic.NewBoolQuery()
	sourceQuery := elastic.NewBoolQuery()
	destQuery := elastic.NewBoolQuery()

	// An empty name results in a namespace only query.
	if name == "" {
		sourceQuery = sourceQuery.Must(
			elastic.NewTermQuery("source_namespace", namespace),
		)
		destQuery = destQuery.Must(
			elastic.NewTermQuery("dest_namespace", namespace),
		)
	} else {
		sourceQuery = sourceQuery.Must(
			elastic.NewTermQuery("source_namespace", namespace),
			elastic.NewTermQuery("source_name_aggr", name),
		)
		destQuery = destQuery.Must(
			elastic.NewTermQuery("dest_namespace", namespace),
			elastic.NewTermQuery("dest_name_aggr", name),
		)
	}

	return nameOrNamespaceQuery.Should(sourceQuery, destQuery)

}

// CompositeAggregator is an interface to provide composite aggregation via Elasticsearch.
type CompositeAggregator interface {
	SearchCompositeAggregations(
		context.Context, *pelastic.CompositeAggregationQuery, pelastic.CompositeAggregationKey,
	) (<-chan *pelastic.CompositeAggregationBucket, <-chan error)
}

// TODO: Add special error handling for elastic queries that are rejected because elastic permissions are bad.
func SearchFlows(ctx context.Context, c CompositeAggregator, query elastic.Query, params *PolicyRecommendationParams) ([]*api.Flow, error) {
	aggQuery := &pelastic.CompositeAggregationQuery{
		DocumentIndex:           params.DocumentIndex,
		Query:                   query,
		Name:                    FlowlogBuckets,
		AggCompositeSourceInfos: CompositeSources,
		AggNestedTermInfos:      AggregatedTerms,
	}

	// Search for the raw data in ES.
	rcvdBuckets, rcvdErrs := c.SearchCompositeAggregations(ctx, aggQuery, nil)

	flows := []*api.Flow{}
	// Iterate through all the raw buckets from ES until the channel is closed.
	for rawBucket := range rcvdBuckets {
		// Convert the bucket to an api.Flow
		flow := pelastic.ConvertFlow(rawBucket, PRCompositeSourcesIdxs, PRAggregatedTermNames)
		flows = append(flows, flow)
	}

	if err, ok := <-rcvdErrs; ok {
		log.WithError(err).Warning("Hit error processing flow logs")
		return flows, err
	}

	return flows, nil
}
