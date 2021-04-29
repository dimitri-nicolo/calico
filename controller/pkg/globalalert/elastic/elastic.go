// Copyright 2021 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"

	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"

	"github.com/olivere/elastic/v7"

	v3 "github.com/projectcalico/apiserver/pkg/apis/projectcalico/v3"
)

const (
	AggregationBucketSize    = 10000
	DefaultLookback          = time.Minute * 10
	QuerySize                = 10000
	AutoBulkFlush            = 500
	PaginationSize           = 500
	AlertEventType           = "alert"
	EventIndexPattern        = "tigera_secure_ee_events.%s"
	AuditIndexPattern        = "tigera_secure_ee_audit_*.%s.*"
	FlowLogIndexPattern      = "tigera_secure_ee_flows.%s.*"
	DNSLogIndexPattern       = "tigera_secure_ee_dns.%s.*"
	CompositeAggregationName = "composite_aggs"
)

type Service interface {
	DeleteElasticWatchers(context.Context)
	ExecuteAlert(*v3.GlobalAlert) libcalicov3.GlobalAlertStatus
}

type service struct {
	// esCLI is an Elasticsearch client.
	esCLI *elastic.Client
	// esBulkProcessor used for sending bulk Elasticsearch request.
	esBulkProcessor *elastic.BulkProcessor
	// clusterName is name of the cluster
	clusterName string
	// sourceIndexName is name of the Elasticsearch index to query data from.
	sourceIndexName string
	// eventIndexName is name of the Elasticsearch events index to write documents do.
	eventIndexName string
	// query has the entire Elasticsearch query based on GlobalAlert fields
	query map[string]interface{}
	// aggs has the composite aggregation query, this will be altered and reused during pagination with `after` key.
	aggs map[string]interface{}
}

// NewService builds Elasticsearch query that will be used periodically to query Elasticsearch data.
func NewService(esCLI *elastic.Client, clusterName string, alert *v3.GlobalAlert) (Service, error) {
	e := &service{
		esCLI:       esCLI,
		clusterName: clusterName,
	}
	e.buildIndexName(alert)

	err := e.buildEsQuery(alert)
	if err != nil {
		return nil, err
	}

	e.esBulkProcessor, err = e.esCLI.BulkProcessor().
		BulkActions(AutoBulkFlush).
		Do(context.Background())
	if err != nil {
		log.WithError(err).Errorf("failed to initialize Elasticsearch bulk processor for GlobalAlert %s", alert.Name)
		return nil, err
	}

	return e, nil
}

// buildIndexName updates the events index name and name of the source index to query.
func (e *service) buildIndexName(alert *v3.GlobalAlert) {
	e.eventIndexName = fmt.Sprintf(EventIndexPattern, e.clusterName)

	switch alert.Spec.DataSet {
	case libcalicov3.GlobalAlertDataSetAudit:
		e.sourceIndexName = fmt.Sprintf(AuditIndexPattern, e.clusterName)
	case libcalicov3.GlobalAlertDataSetDNS:
		e.sourceIndexName = fmt.Sprintf(DNSLogIndexPattern, e.clusterName)
	case libcalicov3.GlobalAlertDataSetFlows:
		e.sourceIndexName = fmt.Sprintf(FlowLogIndexPattern, e.clusterName)
	default:
		log.Errorf("unknown dataset %s in GlobalAlert %s.", alert.Spec.DataSet, alert.Name)
	}
}

// buildEsQuery build Elasticsearch query from the fields in GlobalAlert spec.
// Builds a metric aggregation query if spec.metric is set to either avg, min, max or sum.
// Builds a composite aggregation query if spec.aggregateBy is set.
// The size parameter for Elasticsearch query is 0 if either composite or metric aggregation is set, or
// if GlobalAlert spec.metric is 0, as individual index documents are not needed to generate events.
func (e *service) buildEsQuery(alert *v3.GlobalAlert) error {
	aggs := e.buildMetricAggregation(alert.Spec.Field, alert.Spec.Metric)
	aggs = e.buildCompositeAggregation(alert, aggs)

	mustQuery, err := e.convertAlertSpecQueryToEsQuery(alert)
	if err != nil {
		return err
	}
	filterQuery, err := e.buildLookBackRange(alert)
	if err != nil {
		return err
	}
	e.query = JsonObject{
		"query": JsonObject{
			"bool": JsonObject{
				"must":   mustQuery,
				"filter": filterQuery,
			},
		},
	}

	if aggs != nil {
		e.query["size"] = 0
		e.query["aggs"] = aggs
	} else if alert.Spec.Metric == libcalicov3.GlobalAlertMetricCount {
		e.query["size"] = 0
	} else {
		e.query["size"] = QuerySize
	}
	return nil
}

// buildCompositeAggregation builds and returns a composite aggregation query for the GlobalAlert
func (e *service) buildCompositeAggregation(alert *v3.GlobalAlert, aggs JsonObject) JsonObject {
	var src []JsonObject
	if len(alert.Spec.AggregateBy) != 0 {
		for i := len(alert.Spec.AggregateBy) - 1; i >= 0; i-- {
			src = append(src, JsonObject{
				alert.Spec.AggregateBy[i]: JsonObject{
					"terms": JsonObject{
						"field": alert.Spec.AggregateBy[i],
					},
				},
			})
		}
	}
	if len(src) != 0 {
		e.aggs = JsonObject{
			"composite": JsonObject{
				"size":    PaginationSize,
				"sources": src,
			},
		}
		if aggs != nil {
			e.aggs["aggregations"] = aggs
		}
		aggs = JsonObject{
			CompositeAggregationName: e.aggs,
		}
	}
	return aggs
}

// convertAlertSpecQueryToEsQuery converts GlobalAlert's spec.query to Elasticsearch query.
func (e *service) convertAlertSpecQueryToEsQuery(alert *v3.GlobalAlert) (JsonObject, error) {
	q, err := query.ParseQuery(alert.Spec.Query)
	if err != nil {
		log.WithError(err).Errorf("failed to parse spec.query in %s", alert.Name)
		return nil, err
	}

	var converter ElasticQueryConverter
	switch alert.Spec.DataSet {
	case libcalicov3.GlobalAlertDataSetAudit:
		err := query.Validate(q, query.IsValidAuditAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewAuditConverter()
	case libcalicov3.GlobalAlertDataSetDNS:
		err := query.Validate(q, query.IsValidDNSAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewDNSConverter()
	case libcalicov3.GlobalAlertDataSetFlows:
		err := query.Validate(q, query.IsValidFlowsAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewFlowsConverter()
	default:
		return nil, fmt.Errorf("unknown dataset: %s", alert.Spec.DataSet)
	}

	return converter.Convert(q), nil

}

// buildMetricAggregation builds and returns a metric aggregation query for the GlobalAlert
func (e *service) buildMetricAggregation(field string, metric string) JsonObject {
	if metric == libcalicov3.GlobalAlertMetricCount || metric == "" {
		return nil
	}

	return JsonObject{
		field: JsonObject{
			metric: JsonObject{
				"field": field,
			},
		},
	}
}

// buildLookBackRange builds the Elasticsearch range query from GlobalAlert's spec.lookback if it exists,
// else uses the default lookback duration.
func (e *service) buildLookBackRange(alert *v3.GlobalAlert) (JsonObject, error) {
	var timeField string
	switch alert.Spec.DataSet {
	case libcalicov3.GlobalAlertDataSetDNS, libcalicov3.GlobalAlertDataSetFlows:
		timeField = "start_time"
	case libcalicov3.GlobalAlertDataSetAudit:
		timeField = "requestReceivedTimestamp"
	default:
		return nil, fmt.Errorf("unknown dataset %s in GlobalAlert %s", alert.Spec.DataSet, alert.Name)
	}

	var lookback time.Duration
	if alert.Spec.Lookback == nil || alert.Spec.Lookback.Duration <= 0 {
		lookback = DefaultLookback
	} else {
		lookback = alert.Spec.Lookback.Duration
	}

	return JsonObject{
		"range": JsonObject{
			timeField: JsonObject{
				"gte": fmt.Sprintf("now-%ds", int64(lookback.Seconds())),
				"lte": "now",
			},
		},
	}, nil
}

// DeleteElasticWatchers deletes all the service watchers related to the given cluster.
func (e *service) DeleteElasticWatchers(ctx context.Context) {
	res, err := e.esCLI.Search().Index(".watches").Do(ctx)
	if err != nil {
		if eerr, ok := err.(*elastic.Error); ok && eerr.Status == http.StatusNotFound {
			log.Infof("Elasticsearch watches index doesn't exist")
			return
		}
		log.WithError(err).Error("failed to query existing Elasticsearch watches")
		return
	}
	for _, hit := range res.Hits.Hits {
		if strings.HasPrefix(hit.Id, fmt.Sprintf(WatchNamePrefixPattern, e.clusterName)) {
			_, err := e.esCLI.XPackWatchDelete(hit.Id).Do(ctx)
			if err != nil {
				log.WithError(err).Error("failed to delete Elasticsearch watcher")
				return
			}
		}
	}
}

// ExecuteAlert executes the Elasticsearch query built from GlobalAlert, processes the resulting data,
// generates events and update the cached GlobalAlert status.
// If spec.aggregateBy is set, execute Elasticsearch query by paginating over composite aggregation.
// If both spec.metric and spec.aggregateBy are not set, result retried will be documents from the index,
// scroll through them to generate events.
// If spec.metric is set and spec.aggregateBy is not set, the result has only metric aggregation,
// verify it against spec.threshold to generate events.
func (e *service) ExecuteAlert(alert *v3.GlobalAlert) libcalicov3.GlobalAlertStatus {
	log.Infof("Executing Elasticsearch query and processing result for GlobalAlert %s in cluster %s", alert.Name, e.clusterName)

	var status libcalicov3.GlobalAlertStatus
	var err error
	if alert.Spec.AggregateBy != nil {
		status, err = e.executeCompositeQuery(alert)
		if err != nil {
			log.WithError(err).Errorf("failed to query Elasticsearch for GlobalAlert %s", alert.Name)
		}
	} else if alert.Spec.Metric == "" {
		status, err = e.executeQueryWithScroll(alert)
		if err != nil {
			log.WithError(err).Errorf("failed to query elasticsearch for GlobalAlert %s", alert.Name)
		}
	} else if alert.Spec.Metric != "" {
		status, err = e.executeQuery(alert)
		if err != nil {
			log.WithError(err).Errorf("failed to query Elasticsearch for GlobalAlert %s", alert.Name)
		}
	} else {
		log.Errorf("failed to query elasticsearch for GlobalAlert %s", alert.Name)
	}

	return status
}

// executeCompositeQuery executes the composite aggregation Elasticsearch query,
// if resulting data has after_key set, query Elasticsearch again by setting the received after_key to get remaining aggregation buckets.
// Maximum number of buckets retrieved is based on AggregationBucketSize, if there are more buckets left it logs warning.
// For each bucket retrieved, verifies the values against the metrics in GlobalAlert and creates a document in events index if alert conditions are satisfied.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeCompositeQuery(alert *v3.GlobalAlert) (libcalicov3.GlobalAlertStatus, error) {
	query := e.query
	status := alert.Status
	var afterKey JsonObject
	var bucketCounter int

	for bucketCounter = 0; bucketCounter < AggregationBucketSize; {
		searchResult, err := e.esCLI.Search().Index(e.sourceIndexName).Source(query).Do(context.Background())
		status.LastExecuted = &metav1.Time{Time: time.Now()}
		if err != nil {
			log.WithError(err).Errorf("failed to execute Elasticsearch query for GlobalAlert %s", alert.Name)
			status.Healthy = false
			status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
			return e.bulkFlush(status, err)
		}
		aggs := searchResult.Aggregations
		aggBuckets, exists := aggs.Composite(CompositeAggregationName)
		if !exists {
			// If result doesn't have expected aggregation there is nothing to add to events index.
			status.Healthy = true
			status.ErrorConditions = nil
			return e.bulkFlush(status, err)
		}

		afterKey = aggBuckets.AfterKey
		bucketCounter += len(aggBuckets.Buckets)
		if afterKey != nil {
			caggs := e.aggs["composite"].(JsonObject)
			caggs["after"] = afterKey
			query["aggs"] = JsonObject{
				CompositeAggregationName: JsonObject{
					"composite": caggs,
				},
			}
		}

		for _, b := range aggBuckets.Buckets {
			record := JsonObject{}

			// compare bucket value to GlobalAlert metric
			switch alert.Spec.Metric {
			case "":
				// nothing to compare if metric not set.
			case libcalicov3.GlobalAlertMetricCount:
				if compare(float64(b.DocCount), alert.Spec.Threshold, alert.Spec.Condition) {
					record["count"] = b.DocCount
				} else {
					// alert condition not satisfied
					continue
				}
			default:
				metricAggs, exists := b.Terms(alert.Spec.Field)
				if !exists {
					// noting to add to events index for this bucket.
					continue
				}
				var tempMetric float64
				if err := json.Unmarshal(metricAggs.Aggregations["value"], &tempMetric); err != nil {
					log.WithError(err).Errorf("failed to unmarshal GlobalAlert %s Elasticsearch response", alert.Name)
					status.Healthy = false
					status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
					return e.bulkFlush(status, err)
				}
				if compare(tempMetric, alert.Spec.Threshold, alert.Spec.Condition) {
					record[alert.Spec.Metric] = tempMetric
				} else {
					// alert condition not satisfied
					continue
				}
			}
			// Add the bucket names into events document
			for k, v := range b.Key {
				record[k] = v
			}
			doc := e.buildEventsIndexDoc(alert, record)
			r := elastic.NewBulkIndexRequest().Index(e.eventIndexName).Doc(doc)
			e.esBulkProcessor.Add(r)
			status.LastEvent = &metav1.Time{Time: time.Now()}
		}

		if afterKey == nil {
			// we have processed all the buckets
			break
		}
	}
	if err := e.esBulkProcessor.Flush(); err != nil {
		log.WithError(err).Errorf("failed to flush Elasticsearch BulkProcessor")
		status.Healthy = false
		status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
		return status, err
	}
	if bucketCounter > AggregationBucketSize && afterKey != nil {
		log.Warnf("More that %d buckets received in Elasticsearch query result for GlobalAlert %s", AggregationBucketSize, alert.Name)
	}

	status.Healthy = true
	status.ErrorConditions = nil
	return status, nil
}

// executeQueryWithScroll executes the Elasticsearch query using scroll and for each document in the search result adds a document into events index.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeQueryWithScroll(alert *v3.GlobalAlert) (libcalicov3.GlobalAlertStatus, error) {
	status := alert.Status
	scroll := e.esCLI.Scroll(e.sourceIndexName).Body(e.query).Size(PaginationSize)
	for {
		results, err := scroll.Do(context.Background())
		status.LastExecuted = &metav1.Time{Time: time.Now()}

		if err == io.EOF {
			status.Healthy = true
			status.ErrorConditions = nil
			return e.bulkFlush(status, nil)
		}
		if err != nil {
			log.WithError(err).Errorf("failed to execute Elasticsearch query for GlobalAlert %s", alert.Name)
			status.Healthy = false
			status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
			return e.bulkFlush(status, err)
		}

		for _, hit := range results.Hits.Hits {
			var record JsonObject
			err := json.Unmarshal(hit.Source, &record)
			if err != nil {
				log.WithError(err).Errorf("failed to unmarshal GlobalAlert %s Elasticsearch response", alert.Name)
				status.Healthy = false
				status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
				return e.bulkFlush(status, err)
			}
			doc := e.buildEventsIndexDoc(alert, record)
			r := elastic.NewBulkIndexRequest().Index(e.eventIndexName).Doc(doc)
			e.esBulkProcessor.Add(r)
			status.LastEvent = &metav1.Time{Time: time.Now()}
		}
		if err := e.esBulkProcessor.Flush(); err != nil {
			log.WithError(err).Errorf("failed to flush Elasticsearch BulkProcessor")
			return status, err
		}
	}
}

// executeQuery execute the Elasticsearch query, adds a document into events index is query result satisfies alert conditions.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeQuery(alert *v3.GlobalAlert) (libcalicov3.GlobalAlertStatus, error) {
	status := alert.Status
	result, err := e.esCLI.Search().Index(e.sourceIndexName).Source(e.query).Do(context.Background())
	status.LastExecuted = &metav1.Time{Time: time.Now()}

	if err != nil {
		log.WithError(err).Errorf("failed to execute Elasticsearch query for GlobalAlert %s", alert.Name)
		status.Healthy = false
		status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
		return status, err
	}

	var doc JsonObject
	switch alert.Spec.Metric {
	case libcalicov3.GlobalAlertMetricCount:
		if compare(float64(result.Hits.TotalHits.Value), alert.Spec.Threshold, alert.Spec.Condition) {
			record := JsonObject{
				"count": result.Hits.TotalHits.Value,
			}
			doc = e.buildEventsIndexDoc(alert, record)
		}
	default:
		aggs := result.Aggregations
		metricAggs, exists := aggs.Terms(alert.Spec.Field)
		if !exists {
			status.Healthy = true
			status.ErrorConditions = nil
			return status, nil
		}
		var tempMetric float64
		if err := json.Unmarshal(metricAggs.Aggregations["value"], &tempMetric); err != nil {
			log.WithError(err).Errorf("failed to unmarshal GlobalAlert %s Elasticsearch response", alert.Name)
			status.Healthy = false
			status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
			return status, err
		}
		if compare(tempMetric, alert.Spec.Threshold, alert.Spec.Condition) {
			doc = e.buildEventsIndexDoc(alert, JsonObject{alert.Spec.Metric: tempMetric})
		}
	}

	if doc != nil {
		r := elastic.NewBulkIndexRequest().Index(e.eventIndexName).Doc(doc)
		e.esBulkProcessor.Add(r)
		status.LastEvent = &metav1.Time{Time: time.Now()}
		if err := e.esBulkProcessor.Flush(); err != nil {
			status.Healthy = false
			status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
			log.WithError(err).Errorf("failed to flush Elasticsearch BulkProcessor")
			return status, err
		}
	}

	status.Healthy = true
	status.ErrorConditions = nil
	return status, nil
}

// bulkFlush flushes any remaining data in the BulkProcessor, if there is error during flush update the error condition in status before retuning.
func (e *service) bulkFlush(status libcalicov3.GlobalAlertStatus, err error) (libcalicov3.GlobalAlertStatus, error) {
	if err := e.esBulkProcessor.Flush(); err != nil {
		log.WithError(err).Errorf("failed to flush Elasticsearch BulkProcessor")
		status.Healthy = false
		status.ErrorConditions = append(status.ErrorConditions, libcalicov3.ErrorCondition{Message: err.Error()})
		return status, err
	}
	return status, err
}

// buildEventsIndexDoc builds an object that can be sent to events index.
func (e *service) buildEventsIndexDoc(alert *v3.GlobalAlert, record JsonObject) JsonObject {
	description := alert.Spec.Summary
	if alert.Spec.Summary == "" {
		description = alert.Spec.Description
	}
	return JsonObject{
		"type":        AlertEventType,
		"description": description,
		"severity":    alert.Spec.Severity,
		"time":        time.Now().Unix(),
		"record":      record,
		"alert":       alert.Name,
	}
}

// compare returns a boolean after comparing the given input.
func compare(left, right float64, operation string) bool {
	switch operation {
	case "eq":
		return left == right
	case "not_eq":
		return left != right
	case "lt":
		return left < right
	case "lte":
		return left <= right
	case "gt":
		return left > right
	case "gte":
		return left >= right
	default:
		log.Errorf("unexpected comparison operation %s", operation)
		return false
	}
}
