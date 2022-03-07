// Copyright 2021-2022 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/intrusion-detection/controller/pkg/maputil"
	lmaAPI "github.com/tigera/lma/pkg/api"
	lma "github.com/tigera/lma/pkg/elastic"
)

const (
	AggregationBucketSize    = 10000
	DefaultLookback          = time.Minute * 10
	QuerySize                = 10000
	AutoBulkFlush            = 500
	PaginationSize           = 500
	MaxErrorsSize            = 10
	AlertEventType           = "global_alert"
	EventIndexPattern        = "tigera_secure_ee_events.%s"
	AuditIndexPattern        = "tigera_secure_ee_audit_*.%s.*"
	FlowLogIndexPattern      = "tigera_secure_ee_flows.%s.*"
	DNSLogIndexPattern       = "tigera_secure_ee_dns.%s.*"
	L7LogIndexPattern        = "tigera_secure_ee_l7.%s.*"
	WAFLogIndexPattern       = "tigera_secure_ee_waf.%s.*"
	CompositeAggregationName = "composite_aggs"
)

type Service interface {
	DeleteElasticWatchers(context.Context)
	ExecuteAlert(*v3.GlobalAlert) v3.GlobalAlertStatus
}

type service struct {
	// esCLI is an Elasticsearch client.
	lmaESClient lma.Client
	// http client for making calls to the vulnerability api.
	httpClient *http.Client
	// clusterName is name of the cluster.
	clusterName string
	// sourceIndexName is name of the Elasticsearch index to query data from.
	sourceIndexName string
	// query has the entire Elasticsearch query based on GlobalAlert fields.
	query JsonObject
	// aggs has the composite aggregation query, this will be altered and reused during pagination with `after` key.
	aggs JsonObject
	// globalAlert has the copy of GlobalAlert, it is updated periodically when Elasticsearch is queried for alert.
	globalAlert *v3.GlobalAlert
	// bulkFlushErrored is true if flushing to Elasticsearch encounters an error, this is reset every time the Elasticsearch is queried.
	bulkFlushErrored bool
}

// NewService builds Elasticsearch query that will be used periodically to query Elasticsearch data.
func NewService(lmaESClient lma.Client, clusterName string, alert *v3.GlobalAlert) (Service, error) {
	e := &service{
		clusterName: clusterName,
		lmaESClient: lmaESClient,
	}

	// We query the base API for the vulnerability dataset and ES for others.
	var err error
	if alert.Spec.DataSet == v3.GlobalAlertDataSetVulnerability {
		cfg, cfgErr := getImageAssuranceTLSConfig()
		if cfgErr != nil {
			return nil, cfgErr
		}
		e.httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}
		err = e.buildVulnerabilityQuery(alert)
	} else {
		e.buildIndexName(alert)
		err = e.buildEsQuery(alert)
	}

	if err != nil {
		return nil, err
	}

	AfterFn := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
		if response.Errors {
			e.bulkFlushErrored = true
			for _, v := range response.Items {
				for _, i := range v {
					log.Errorf("Error during bulk flush for GlobalAlert %s : %#v", alert.Name, i.Error)
					e.globalAlert.Status.Healthy = false
					e.globalAlert.Status.ErrorConditions = appendError(e.globalAlert.Status.ErrorConditions,
						v3.ErrorCondition{Message: i.Error.Reason, Type: i.Error.Type})
				}
			}
		}
	}

	err = lmaESClient.BulkProcessorInitialize(context.Background(), AfterFn)
	if err != nil {
		log.WithError(err).Errorf("failed to initialize Elasticsearch bulk processor for GlobalAlert %s", alert.Name)
		return nil, err
	}

	return e, nil
}

// buildVulnerabilityQuery builds Vulnerability API query parameters from the fields in GlobalAlert spec.
func (e *service) buildVulnerabilityQuery(alert *v3.GlobalAlert) error {
	res, err := e.convertAlertSpecQueryToVulnerabilityQueryParameters(alert)
	if err != nil {
		return err
	}
	e.query = res

	return nil
}

// convertAlertSpecQueryToVulnerabilityQueryParameters converts GlobalAlert's spec.query to Vulnerability API query parameters.
func (e *service) convertAlertSpecQueryToVulnerabilityQueryParameters(alert *v3.GlobalAlert) (JsonObject, error) {
	q, err := query.ParseQuery(e.substituteVariables((alert)))
	if err != nil {
		log.WithError(err).Errorf("failed to parse spec.query in %s", alert.Name)
		return nil, err
	}

	err = query.Validate(q, query.IsValidVulnerabilityAtom)
	if err != nil {
		log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
		return nil, err
	}

	res := JsonObject{}
	for _, a := range q.Atoms() {
		res[a.Key] = a.Value
	}

	return res, nil
}

// buildIndexName updates the events index name and name of the source index to query.
func (e *service) buildIndexName(alert *v3.GlobalAlert) {
	switch alert.Spec.DataSet {
	case v3.GlobalAlertDataSetAudit:
		e.sourceIndexName = fmt.Sprintf(AuditIndexPattern, e.clusterName)
	case v3.GlobalAlertDataSetDNS:
		e.sourceIndexName = fmt.Sprintf(DNSLogIndexPattern, e.clusterName)
	case v3.GlobalAlertDataSetFlows:
		e.sourceIndexName = fmt.Sprintf(FlowLogIndexPattern, e.clusterName)
	case v3.GlobalAlertDataSetL7:
		e.sourceIndexName = fmt.Sprintf(L7LogIndexPattern, e.clusterName)
	case v3.GlobalAlertDataSetWAF:
		e.sourceIndexName = fmt.Sprintf(WAFLogIndexPattern, e.clusterName)
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
	} else if alert.Spec.Metric == v3.GlobalAlertMetricCount {
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
	q, err := query.ParseQuery(e.substituteVariables(alert))
	if err != nil {
		log.WithError(err).Errorf("failed to parse spec.query in %s", alert.Name)
		return nil, err
	}

	var converter ElasticQueryConverter
	switch alert.Spec.DataSet {
	case v3.GlobalAlertDataSetAudit:
		err := query.Validate(q, query.IsValidAuditAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewAuditConverter()
	case v3.GlobalAlertDataSetDNS:
		err := query.Validate(q, query.IsValidDNSAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewDNSConverter()
	case v3.GlobalAlertDataSetFlows:
		err := query.Validate(q, query.IsValidFlowsAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewFlowsConverter()
	case v3.GlobalAlertDataSetL7:
		err := query.Validate(q, query.IsValidL7LogsAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewL7Converter()
	case v3.GlobalAlertDataSetWAF:
		err := query.Validate(q, query.IsValidWAFAtom)
		if err != nil {
			log.WithError(err).Errorf("failed to validate spec.query in %s", alert.Name)
			return nil, err
		}
		converter = NewWAFConverter()
	default:
		return nil, fmt.Errorf("unknown dataset: %s", alert.Spec.DataSet)
	}

	return converter.Convert(q), nil

}

// buildMetricAggregation builds and returns a metric aggregation query for the GlobalAlert
func (e *service) buildMetricAggregation(field string, metric string) JsonObject {
	if metric == v3.GlobalAlertMetricCount || metric == "" {
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
	case v3.GlobalAlertDataSetDNS, v3.GlobalAlertDataSetFlows, v3.GlobalAlertDataSetL7:
		timeField = "start_time"
	case v3.GlobalAlertDataSetWAF:
		timeField = "@timestamp"
	case v3.GlobalAlertDataSetAudit:
		timeField = "requestReceivedTimestamp"
	case v3.GlobalAlertDataSetVulnerability:
		timeField = "start_date"
	default:
		return nil, fmt.Errorf("unknown dataset %s in GlobalAlert %s", alert.Spec.DataSet, alert.Name)
	}

	var lookback time.Duration
	if alert.Spec.Lookback == nil || alert.Spec.Lookback.Duration <= 0 {
		lookback = DefaultLookback
	} else {
		lookback = alert.Spec.Lookback.Duration
	}

	if alert.Spec.DataSet == v3.GlobalAlertDataSetVulnerability {
		now := time.Now()
		return JsonObject{
			timeField:  now.Add(-lookback).Unix(),
			"end_date": now.Unix(),
		}, nil
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
	res, err := e.lmaESClient.Backend().Search().Index(".watches").Do(ctx)
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
			_, err := e.lmaESClient.Backend().XPackWatchDelete(hit.Id).Do(ctx)
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
func (e *service) ExecuteAlert(alert *v3.GlobalAlert) v3.GlobalAlertStatus {
	log.Infof("Executing Elasticsearch query and processing result for GlobalAlert %s in cluster %s", alert.Name, e.clusterName)

	e.globalAlert = alert
	e.bulkFlushErrored = false
	if e.globalAlert.Spec.DataSet == v3.GlobalAlertDataSetVulnerability {
		e.executeVulnerabilityQuery()
	} else if e.globalAlert.Spec.AggregateBy != nil {
		e.executeEsCompositeQuery()
	} else if e.globalAlert.Spec.Metric == "" {
		e.executeEsQueryWithScroll()
	} else if e.globalAlert.Spec.Metric != "" {
		e.executeEsQuery()
	} else {
		log.Errorf("failed to query elasticsearch for GlobalAlert %s", e.globalAlert.Name)
	}

	return e.globalAlert.Status
}

// executeEsCompositeQuery executes the composite aggregation Elasticsearch query,
// if resulting data has after_key set, query Elasticsearch again by setting the received after_key to get remaining aggregation buckets.
// Maximum number of buckets retrieved is based on AggregationBucketSize, if there are more buckets left it logs warning.
// For each bucket retrieved, verifies the values against the metrics in GlobalAlert and creates a document in events index if alert conditions are satisfied.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeEsCompositeQuery() {
	query, err := maputil.Copy(e.query)
	if err != nil {
		log.WithError(err).Errorf("failed to copy Elasticsearch query for GlobalAlert %s", e.globalAlert.Name)
		return
	}

	compositeAggs, err := maputil.Copy(e.aggs["composite"].(JsonObject))
	if err != nil {
		log.WithError(err).Errorf("failed to copy Elasticsearch query for GlobalAlert %s", e.globalAlert.Name)
		return
	}

	var afterKey JsonObject
	var bucketCounter int

	for bucketCounter = 0; bucketCounter < AggregationBucketSize; {
		searchResult, err := e.lmaESClient.Backend().Search().Index(e.sourceIndexName).Source(query).Do(context.Background())
		e.globalAlert.Status.LastExecuted = &metav1.Time{Time: time.Now()}
		if err != nil {
			log.WithError(err).Errorf("failed to execute Elasticsearch query for GlobalAlert %s", e.globalAlert.Name)
			e.setErrorAndFlush(v3.ErrorCondition{Message: err.Error()})
			return
		}
		aggs := searchResult.Aggregations
		aggBuckets, exists := aggs.Composite(CompositeAggregationName)
		if !exists {
			// If result doesn't have expected aggregation there is nothing to add to events index.
			if e.bulkFlushErrored {
				return
			}
			e.globalAlert.Status.Healthy = true
			e.globalAlert.Status.ErrorConditions = nil
			return
		}

		afterKey = aggBuckets.AfterKey
		bucketCounter += len(aggBuckets.Buckets)
		if afterKey != nil {
			compositeAggs["after"] = afterKey
			query["aggs"] = JsonObject{
				CompositeAggregationName: JsonObject{
					"composite": compositeAggs,
				},
			}
		}

		for _, b := range aggBuckets.Buckets {
			record := JsonObject{}

			// compare bucket value to GlobalAlert metric
			switch e.globalAlert.Spec.Metric {
			case "":
				// nothing to compare if metric not set.
			case v3.GlobalAlertMetricCount:
				if compare(float64(b.DocCount), e.globalAlert.Spec.Threshold, e.globalAlert.Spec.Condition) {
					record["count"] = b.DocCount
				} else {
					// alert condition not satisfied
					continue
				}
			default:
				metricAggs, exists := b.Terms(e.globalAlert.Spec.Field)
				if !exists {
					// noting to add to events index for this bucket.
					continue
				}
				var tempMetric float64
				if err := json.Unmarshal(metricAggs.Aggregations["value"], &tempMetric); err != nil {
					log.WithError(err).Errorf("failed to unmarshal GlobalAlert %s Elasticsearch response", e.globalAlert.Name)
					e.setErrorAndFlush(v3.ErrorCondition{Message: err.Error()})
					return
				}
				if compare(tempMetric, e.globalAlert.Spec.Threshold, e.globalAlert.Spec.Condition) {
					record[e.globalAlert.Spec.Metric] = tempMetric
				} else {
					// alert condition not satisfied
					continue
				}
			}
			// Add the bucket names into events document
			for k, v := range b.Key {
				record[k] = v
			}
			doc := e.buildEventsIndexDoc(record)
			if err = e.lmaESClient.PutBulkSecurityEvent(doc); err != nil {
				log.WithError(err).Errorf("failed to add event for GlobalAlert %s", e.globalAlert.Name)
				continue
			}

			e.globalAlert.Status.LastEvent = &metav1.Time{Time: time.Now()}
		}
		e.lmaESClient.BulkProcessorFlush()
		if afterKey == nil {
			// we have processed all the buckets.
			break
		}
	}

	if bucketCounter > AggregationBucketSize && afterKey != nil {
		log.Warnf("More that %d buckets received in Elasticsearch query result for GlobalAlert %s", AggregationBucketSize, e.globalAlert.Name)
	}

	if e.bulkFlushErrored {
		return
	}

	e.globalAlert.Status.Healthy = true
	e.globalAlert.Status.ErrorConditions = nil
}

// executeEsQueryWithScroll executes the Elasticsearch query using scroll and for each document in the search result adds a document into events index.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeEsQueryWithScroll() {
	scroll := e.lmaESClient.Backend().Scroll(e.sourceIndexName).Body(e.query).Size(PaginationSize)
	for {
		results, err := scroll.Do(context.Background())
		e.globalAlert.Status.LastExecuted = &metav1.Time{Time: time.Now()}

		if err == io.EOF {
			if e.bulkFlushErrored {
				return
			}
			e.globalAlert.Status.Healthy = true
			e.globalAlert.Status.ErrorConditions = nil
			return
		}
		if err != nil {
			log.WithError(err).Errorf("failed to execute Elasticsearch query for GlobalAlert %s", e.globalAlert.Name)
			e.setErrorAndFlush(v3.ErrorCondition{Message: err.Error()})
			return
		}

		for _, hit := range results.Hits.Hits {
			var record JsonObject
			err := json.Unmarshal(hit.Source, &record)
			if err != nil {
				log.WithError(err).Errorf("failed to unmarshal GlobalAlert %s Elasticsearch response", e.globalAlert.Name)
				e.setErrorAndFlush(v3.ErrorCondition{Message: err.Error()})
				return
			}
			doc := e.buildEventsIndexDoc(record)
			e.lmaESClient.PutBulkSecurityEvent(doc)
			e.globalAlert.Status.LastEvent = &metav1.Time{Time: time.Now()}
		}

		e.lmaESClient.BulkProcessorFlush()
	}
}

// executeEsQuery execute the Elasticsearch query, adds a document into events index is query result satisfies alert conditions.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeEsQuery() {
	result, err := e.lmaESClient.Backend().Search().Index(e.sourceIndexName).Source(e.query).Do(context.Background())
	e.globalAlert.Status.LastExecuted = &metav1.Time{Time: time.Now()}

	if err != nil {
		log.WithError(err).Errorf("failed to execute Elasticsearch query for GlobalAlert %s", e.globalAlert.Name)
		e.globalAlert.Status.Healthy = false
		e.globalAlert.Status.ErrorConditions = appendError(e.globalAlert.Status.ErrorConditions, v3.ErrorCondition{Message: err.Error()})
		return
	}

	var doc lmaAPI.EventsData
	switch e.globalAlert.Spec.Metric {
	case v3.GlobalAlertMetricCount:
		if compare(float64(result.Hits.TotalHits.Value), e.globalAlert.Spec.Threshold, e.globalAlert.Spec.Condition) {
			record := JsonObject{
				"count": result.Hits.TotalHits.Value,
			}
			doc = e.buildEventsIndexDoc(record)
		}
	default:
		aggs := result.Aggregations
		metricAggs, exists := aggs.Terms(e.globalAlert.Spec.Field)
		if !exists {
			e.globalAlert.Status.Healthy = true
			e.globalAlert.Status.ErrorConditions = nil
			return
		}
		var tempMetric float64
		if err := json.Unmarshal(metricAggs.Aggregations["value"], &tempMetric); err != nil {
			log.WithError(err).Errorf("failed to unmarshal GlobalAlert %s Elasticsearch response", e.globalAlert.Name)
			e.globalAlert.Status.Healthy = false
			e.globalAlert.Status.ErrorConditions = appendError(e.globalAlert.Status.ErrorConditions, v3.ErrorCondition{Message: err.Error()})
			return
		}
		if compare(tempMetric, e.globalAlert.Spec.Threshold, e.globalAlert.Spec.Condition) {
			doc = e.buildEventsIndexDoc(JsonObject{e.globalAlert.Spec.Metric: tempMetric})
		}
	}

	if doc.Type != "" {
		if err := e.lmaESClient.PutBulkSecurityEvent(doc); err != nil {
			log.WithError(err).Error("failed to add events to bulk processor")
			return
		}
		e.globalAlert.Status.LastEvent = &metav1.Time{Time: time.Now()}
		e.lmaESClient.BulkProcessorFlush()
	}

	if e.bulkFlushErrored {
		return
	}
	e.globalAlert.Status.Healthy = true
	e.globalAlert.Status.ErrorConditions = nil
}

func (e *service) executeVulnerabilityQuery() {
	e.globalAlert.Status.LastExecuted = &metav1.Time{Time: time.Now()}

	if lookBackRange, err := e.buildLookBackRange(e.globalAlert); err != nil {
		return
	} else {
		for k, v := range lookBackRange {
			e.query[k] = v
		}
	}

	params := make(VulnerabilityQueryParameterMap)
	for k, v := range e.query {
		switch v := v.(type) {
		case string:
			params[k] = v
		case int64:
			params[k] = strconv.FormatInt(v, 10)
		default:
			log.Warnf("invalid image assurance query parameter type for %s=%v", k, v)
		}
	}

	events, err := queryVulnerabilityDataset(e.httpClient, params)
	if err != nil {
		log.WithError(err).Error("failed to query image assurance API")
		e.globalAlert.Status.Healthy = false
		e.globalAlert.Status.ErrorConditions = appendError(e.globalAlert.Status.ErrorConditions, v3.ErrorCondition{Message: err.Error()})
		return
	}

	switch e.globalAlert.Spec.Metric {
	case v3.GlobalAlertMetricCount:
		if compare(float64(len(events)), e.globalAlert.Spec.Threshold, e.globalAlert.Spec.Condition) {
			doc := e.buildEventsIndexDoc(JsonObject{"count": len(events)})
			if err := e.lmaESClient.PutBulkSecurityEvent(doc); err != nil {
				log.WithError(err).Error("failed to add events to bulk processor")
				e.globalAlert.Status.Healthy = false
				e.globalAlert.Status.ErrorConditions = appendError(e.globalAlert.Status.ErrorConditions, v3.ErrorCondition{Message: err.Error()})
				return
			}
		}
	case v3.GlobalAlertMetricMax:
		field := e.globalAlert.Spec.Field
		for _, event := range events {
			v, ok := event[field]
			if !ok {
				log.Warnf("field %s is not found in response; skipping", field)
				continue
			}

			val, ok := v.(float64)
			if !ok {
				log.Warnf("failed to convert %s: %v to float64", field, v)
				continue
			}

			if compare(val, e.globalAlert.Spec.Threshold, e.globalAlert.Spec.Condition) {
				doc := e.buildEventsIndexDoc(event)
				if err := e.lmaESClient.PutBulkSecurityEvent(doc); err != nil {
					log.WithError(err).Error("failed to add events to bulk processor")
					e.setErrorAndFlush(v3.ErrorCondition{Message: err.Error()})
					return
				}
				e.globalAlert.Status.LastEvent = &metav1.Time{Time: time.Now()}
			}
		}
	default:
		for _, event := range events {
			doc := e.buildEventsIndexDoc(event)
			if err := e.lmaESClient.PutBulkSecurityEvent(doc); err != nil {
				log.WithError(err).Error("failed to add events to bulk processor")
				e.setErrorAndFlush(v3.ErrorCondition{Message: err.Error()})
				return
			}
			e.globalAlert.Status.LastEvent = &metav1.Time{Time: time.Now()}
		}
	}

	e.lmaESClient.BulkProcessorFlush()
	if e.bulkFlushErrored {
		return
	}
	e.globalAlert.Status.Healthy = true
	e.globalAlert.Status.ErrorConditions = nil
}

// setErrorAndFlush sets the Status.Healthy to false, appends the given error to the Status
// and flushes the BulkProcessor.
func (e *service) setErrorAndFlush(err v3.ErrorCondition) {
	e.globalAlert.Status.Healthy = false
	e.globalAlert.Status.ErrorConditions = appendError(e.globalAlert.Status.ErrorConditions, err)
	e.lmaESClient.BulkProcessorFlush()
}

// buildEventsIndexDoc builds an object that can be sent to events index.
func (e *service) buildEventsIndexDoc(record JsonObject) lmaAPI.EventsData {
	description := e.substituteDescriptionOrSummaryContents(record)
	eventData := extractEventData(record)

	eventData.Time = time.Now().Unix()
	eventData.Type = AlertEventType
	eventData.Description = description
	eventData.Severity = e.globalAlert.Spec.Severity
	eventData.Origin = e.globalAlert.Name

	return eventData
}

// substituteVariables finds variables in the query string and replace them with values from GlobalAlertSpec.Substitutions.
func (e *service) substituteVariables(alert *v3.GlobalAlert) string {
	out := alert.Spec.Query
	variables, err := extractVariablesFromTemplate(out)
	if err != nil {
		log.WithError(err).Warnf("failed to build IN template for alert %s due to invalid formatting of bracketed variables", alert.Name)
		return out
	}
	if len(variables) > 0 {
		for _, variable := range variables {
			sub, err := findSubstitutionByVariableName(alert, variable)
			if err != nil {
				log.Warnf("failed to build IN template for alert %s due to wrong variable name %s", alert.Name, variable)
				return out
			}

			// Translate Substitution.Values into the set notation.
			patterns := []string{}
			for _, v := range sub.Values {
				if v != "" {
					patterns = append(patterns, strconv.Quote(v))
				}
			}
			if len(patterns) > 0 {
				out = strings.Replace(out, fmt.Sprintf("${%s}", variable), "{"+strings.Join(patterns, ",")+"}", 1)
			}
		}
	}
	return out
}

// substituteDescriptionOrSummaryContents substitute bracketed variables in summary/description with it's value.
// If there is an error in substitution log error and return the partly substituted value.
func (e *service) substituteDescriptionOrSummaryContents(record JsonObject) string {
	description := e.globalAlert.Spec.Summary
	if e.globalAlert.Spec.Summary == "" {
		description = e.globalAlert.Spec.Description
	}

	vars, err := extractVariablesFromTemplate(description)
	if err != nil {
		log.WithError(err).Warnf("failed to build summary or description for alert %s due to invalid formatting of bracketed variables", e.globalAlert.Name)
	}

	// replace extracted variables with it's value
	for _, v := range vars {
		if value, ok := record[v]; !ok {
			log.Warnf("failed to build summary or description for alert %s due to missing value for variable %s", e.globalAlert.Name, v)
		} else {
			switch value := value.(type) {
			case string:
				description = strings.Replace(description, fmt.Sprintf("${%s}", v), value, 1)
			case int64:
				description = strings.Replace(description, fmt.Sprintf("${%s}", v), strconv.FormatInt(value, 10), 1)
			case float64:
				description = strings.Replace(description, fmt.Sprintf("${%s}", v), strconv.FormatFloat(value, 'f', 1, 64), 1)
			default:
				log.Warnf("failed to build summary or description for alert %s due to unsupported value type for variable %s", e.globalAlert.Name, v)
			}
		}
	}
	return description
}

// extractEventData checks the given record object for keys that are defined in lmaAPI.EventsData,
// for each key found, it assigns them to lmaAPI.EventsData.
func extractEventData(record JsonObject) lmaAPI.EventsData {
	var e lmaAPI.EventsData
	if val, ok := record["source_ip"].(string); ok {
		e.SourceIP = &val
	}
	if val, ok := record["source_port"].(float64); ok {
		v := int64(val)
		e.SourcePort = &v
	}
	if val, ok := record["source_namespace"].(string); ok {
		e.SourceNamespace = val
	}
	if val, ok := record["source_name"].(string); ok {
		e.SourceName = val
	}
	if val, ok := record["source_name_aggr"].(string); ok {
		e.SourceNameAggr = val
	}
	if val, ok := record["dest_ip"].(string); ok {
		e.DestIP = &val
	}
	if val, ok := record["dest_port"].(float64); ok {
		v := int64(val)
		e.DestPort = &v
	}
	if val, ok := record["dest_namespace"].(string); ok {
		e.DestNamespace = val
	}
	if val, ok := record["dest_name"].(string); ok {
		e.DestName = val
	}
	if val, ok := record["dest_name_aggr"].(string); ok {
		e.DestNameAggr = val
	}
	if val, ok := record["host"].(string); ok {
		e.Host = val
	}
	if val, ok := record["host.keyword"].(string); ok {
		e.Host = val
	}

	e.Record = record

	// Allow for nested structures specifically for WAF logs.
	if val, ok := record["source"].(map[string]interface{}); ok {
		if nestedVal, ok := val["ip"].(string); ok {
			e.SourceIP = &nestedVal
		}
		if nestedVal, ok := val["hostname"].(string); ok {
			e.SourceName = nestedVal
		}
		if nestedVal, ok := val["port_num"].(float64); ok {
			v := int64(nestedVal)
			e.SourcePort = &v
		}
	}
	if val, ok := record["destination"].(map[string]interface{}); ok {
		if nestedVal, ok := val["ip"].(string); ok {
			e.DestIP = &nestedVal
		}
		if nestedVal, ok := val["hostname"].(string); ok {
			e.DestName = nestedVal
		}
		if nestedVal, ok := val["port_num"].(float64); ok {
			v := int64(nestedVal)
			e.DestPort = &v
		}
	}

	return e
}

// extractVariablesFromTemplate extracts and returns array of variables in the template string.
func extractVariablesFromTemplate(s string) ([]string, error) {
	var res []string
	for s != "" {
		start := strings.Index(s, "${")
		if start < 0 {
			break
		}
		s = s[start+2:]
		end := strings.Index(s, "}")
		if end < 0 {
			return nil, fmt.Errorf("unterminated }")
		}
		res = append(res, s[:end])
		s = s[end+1:]
	}
	return res, nil
}

// findSubstitutionByVariableName finds the substitution from spec by variable name.
func findSubstitutionByVariableName(alert *v3.GlobalAlert, variable string) (*v3.GlobalAlertSubstitution, error) {
	var substitution *v3.GlobalAlertSubstitution
	for _, sub := range alert.Spec.Substitutions {
		if strings.EqualFold(variable, sub.Name) {
			if substitution != nil {
				return nil, fmt.Errorf("found more than one substitution for variable %s", variable)
			} else {
				substitution = sub.DeepCopy()
			}
		}
	}
	if substitution != nil {
		return substitution, nil
	}
	return nil, fmt.Errorf("variable %s not found", variable)
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

// appendError appends the given error to the list of errors, ensures there are only `MaxErrorsSize` recent errors.
func appendError(errs []v3.ErrorCondition, err v3.ErrorCondition) []v3.ErrorCondition {
	errs = append(errs, err)
	if len(errs) > MaxErrorsSize {
		errs = errs[1:]
	}
	return errs
}
