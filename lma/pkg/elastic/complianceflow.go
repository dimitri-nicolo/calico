// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.
package elastic

import (
	"context"
	"time"

	elastic "github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/resources"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/lma/pkg/api"
)

// This file contains the flow accessor methods used by compliance. Much of this has been superceded by the
// pip/flowviz/recommendation flow processing, as a result there is a lot of duplication of ideas.

const (
	FlowLogIndex              = "tigera_secure_ee_flows"
	noResultsSize             = 0
	endTimeField              = "end_time"
	nameHasBeenAggregatedOver = "-"
)

// Flow log fields that are interesting to us for reporting on traffic to/from endpoints.
var (
	reportFlowFields = []string{
		"source_type", "source_namespace", "source_name", "source_name_aggr",
		"dest_type", "dest_namespace", "dest_name", "dest_name_aggr",
	}
)

func getFlowEndpointName(name, nameAggr string) (flowName string, isAggregated bool) {
	flowName = name
	if name == nameHasBeenAggregatedOver {
		flowName = nameAggr
		isAggregated = true
	}
	return
}

func getFlowEndpointType(flowLogEndpointType, endpointName string) (flowEndpointType string) {
	switch flowLogEndpointType {
	case api.FlowLogEndpointTypeHEP:
		flowEndpointType = resources.TypeCalicoHostEndpoints.Kind
	case api.FlowLogEndpointTypeWEP:
		flowEndpointType = resources.TypeK8sPods.Kind
	case api.FlowLogEndpointTypeNetworkSet:
		flowEndpointType = resources.TypeCalicoGlobalNetworkSets.Kind
	case api.FlowLogEndpointTypeNetwork:
		switch endpointName {
		case api.FlowLogNetworkPublic:
			flowEndpointType = apiv3.KindFlowPublic
		case api.FlowLogNetworkPrivate:
			flowEndpointType = apiv3.KindFlowPrivate
		default:
			log.WithFields(log.Fields{
				"type": flowLogEndpointType,
				"name": endpointName,
			}).Error("Unknown endpoint type")
		}
	default:
		log.WithField("endpoint-type", flowLogEndpointType).Error("Unknown endpoint type")
	}
	return
}

func getFlowEndpointNamespace(ns string) string {
	if ns == api.FlowLogGlobalNamespace {
		return ""
	}
	return ns
}

func NewFlowEndpointFromElasticResult(key map[string]interface{}) *apiv3.EndpointsReportFlow {
	sname, sIsAggregated := getFlowEndpointName(key["source_name"].(string), key["source_name_aggr"].(string))
	dname, dIsAggregated := getFlowEndpointName(key["dest_name"].(string), key["dest_name_aggr"].(string))
	stype := getFlowEndpointType(key["source_type"].(string), sname)
	dtype := getFlowEndpointType(key["dest_type"].(string), dname)
	sns := getFlowEndpointNamespace(key["source_namespace"].(string))
	dns := getFlowEndpointNamespace(key["dest_namespace"].(string))
	return &apiv3.EndpointsReportFlow{
		Source: apiv3.FlowEndpoint{
			Kind:                    stype,
			Namespace:               sns,
			Name:                    sname,
			NameIsAggregationPrefix: sIsAggregated,
		},
		Destination: apiv3.FlowEndpoint{
			Kind:                    dtype,
			Namespace:               dns,
			Name:                    dname,
			NameIsAggregationPrefix: dIsAggregated,
		},
	}
}

// Issue an aggregated Elasticsearch query that matches flow logs that are
// generated or received by the specified namespaces and occurred within the
// start and end time range. We do not filter flow logs using endpoint based
// queries due to potentially large number of in-scope endpoints that may
// have to be specified in the Elasticsearch Terms query.
func (c *client) SearchFlowLogs(ctx context.Context, namespaces []string, start, end *time.Time) <-chan *api.FlowLogResult {
	resultChan := make(chan *api.FlowLogResult, resultBucketSize)
	flogSearchIndex := c.ClusterIndex(FlowLogIndex, "*")

	log.Debugf("Searching across namespaces %+v", namespaces)

	compositeAggrSources := []elastic.CompositeAggregationValuesSource{}
	for _, flowField := range reportFlowFields {
		compositeAggrSources = append(compositeAggrSources, elastic.NewCompositeAggregationTermsValuesSource(flowField).Field(flowField))
	}

	// We want flow logs that are within the reporting interval and must also
	// start and/or end in the namespaces that make up inscope endpoints.
	queries := buildFlowLogQuery(start, end, namespaces, namespaces)

	go func() {
		defer close(resultChan)

		// Query the flow log index. We aren't interested in the actual search
		// results but rather only the aggregated results.
		searchQuery := c.Search().
			Index(flogSearchIndex).
			Size(noResultsSize).
			Query(queries)

		// Aggregate flow logs and fetch results based on Elasticsearch recommended
		// batches of "resultBucketSize".
		compositeFlogs := elastic.NewCompositeAggregation().
			Sources(compositeAggrSources...).
			Size(resultBucketSize)

		var resultsAfter map[string]interface{}

		// Issue the query to Elasticsearch and send results out through the resultsChan.
		// We only terminate the search if when there are no more "buckets" returned by
		// Elasticsearch or the equivalent no-or-empty "after_key" in the aggregated
		// search results.
		for {
			if resultsAfter != nil {
				compositeFlogs = compositeFlogs.AggregateAfter(resultsAfter)
			}

			log.Debug("Issuing search query")
			searchResult, err := searchQuery.
				Aggregation("flog_buckets", compositeFlogs).
				Do(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to search flowlogs")
				resultChan <- &api.FlowLogResult{Err: err}
				return
			}

			flogBuckets, ok := searchResult.Aggregations.Composite("flog_buckets")
			if !ok {
				log.WithError(err).Error("Error fetching composite results for flow logs.")
				resultChan <- &api.FlowLogResult{Err: err}
				return
			}

			if len(flogBuckets.Buckets) == 0 {
				log.Debug("Completed processing flow logs")
				return
			}

			for _, item := range flogBuckets.Buckets {
				resultChan <- &api.FlowLogResult{
					EndpointsReportFlow: NewFlowEndpointFromElasticResult(item.Key),
				}
			}

			if flogBuckets.AfterKey == nil || len(flogBuckets.AfterKey) == 0 {
				return
			}

			resultsAfter = flogBuckets.AfterKey
			log.Debugf("Next set of results will be after key %+v", flogBuckets.AfterKey)
		}
	}()

	return resultChan
}

func buildFlowLogQuery(start, end *time.Time, sourceNamespaces, destNamespaces []string) elastic.Query {
	queries := []elastic.Query{}

	if start != nil || end != nil {
		withinTimeRange := elastic.NewRangeQuery(endTimeField)
		if start != nil {
			withinTimeRange = withinTimeRange.From((*start).Unix())
		}
		if end != nil {
			withinTimeRange = withinTimeRange.To((*end).Unix())
		}
		queries = append(queries, withinTimeRange)
	}

	// Add the namespace queries. We care about flows where either source or destination match, so if searching for
	// both use the Should operator.
	namespaceQueries := []elastic.Query{}
	if len(sourceNamespaces) != 0 {
		namespaceQueries = append(namespaceQueries, buildTermsQuery("source_namespace", sourceNamespaces))
	}
	if len(destNamespaces) != 0 {
		namespaceQueries = append(namespaceQueries, buildTermsQuery("dest_namespace", destNamespaces))
	}
	if len(namespaceQueries) == 1 {
		queries = append(queries, namespaceQueries[0])
	} else if len(namespaceQueries) == 2 {
		queries = append(queries, elastic.NewBoolQuery().Should(namespaceQueries...))
	}

	return elastic.NewBoolQuery().Must(queries...)
}

func buildTermsQuery(fieldName string, items []string) elastic.Query {
	itemIf := make([]interface{}, len(items))
	for i, item := range items {
		itemIf[i] = item
	}
	return elastic.NewTermsQuery(fieldName, itemIf...)
}
