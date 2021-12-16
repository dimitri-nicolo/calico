// Copyright 2021 Tigera Inc. All rights reserved.

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

	lmaAPI "github.com/tigera/lma/pkg/api"
	lma "github.com/tigera/lma/pkg/elastic"

	"github.com/tigera/intrusion-detection/controller/pkg/maputil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"

	"github.com/olivere/elastic/v7"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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
	CompositeAggregationName = "composite_aggs"
)

type Service interface {
	DeleteElasticWatchers(context.Context)
	ExecuteAlert(*v3.GlobalAlert) v3.GlobalAlertStatus
}

type service struct {
	// esCLI is an Elasticsearch client.
	lmaESClient lma.Client
	// clusterName is name of the cluster.
	clusterName string
	// sourceIndexName is name of the Elasticsearch index to query data from.
	sourceIndexName string
	// query has the entire Elasticsearch query based on GlobalAlert fields.
	query map[string]interface{}
	// aggs has the composite aggregation query, this will be altered and reused during pagination with `after` key.
	aggs map[string]interface{}
	// globalAlert has the copy of GlobalAlert, it is updated periodically when Elasticsearch is queried for alert.
	globalAlert *v3.GlobalAlert
	// bulkFlushErrored is true if flushing to Elasticsearch encounters an error, this is reset every time the Elasticsearch is queried.
	bulkFlushErrored bool
}

// NewService builds Elasticsearch query that will be used periodically to query Elasticsearch data.
func NewService(lmaESClient lma.Client, clusterName string, alert *v3.GlobalAlert) (Service, error) {
	e := &service{
		lmaESClient: lmaESClient,
		clusterName: clusterName,
	}
	e.buildIndexName(alert)

	err := e.buildEsQuery(alert)
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
	case v3.GlobalAlertDataSetAudit:
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
	if e.globalAlert.Spec.AggregateBy != nil {
		e.executeCompositeQuery()
	} else if e.globalAlert.Spec.Metric == "" {
		e.executeQueryWithScroll()
	} else if e.globalAlert.Spec.Metric != "" {
		e.executeQuery()
	} else {
		log.Errorf("failed to query elasticsearch for GlobalAlert %s", e.globalAlert.Name)
	}

	return e.globalAlert.Status
}

// executeCompositeQuery executes the composite aggregation Elasticsearch query,
// if resulting data has after_key set, query Elasticsearch again by setting the received after_key to get remaining aggregation buckets.
// Maximum number of buckets retrieved is based on AggregationBucketSize, if there are more buckets left it logs warning.
// For each bucket retrieved, verifies the values against the metrics in GlobalAlert and creates a document in events index if alert conditions are satisfied.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeCompositeQuery() {
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
	return
}

// executeQueryWithScroll executes the Elasticsearch query using scroll and for each document in the search result adds a document into events index.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeQueryWithScroll() {
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

// executeQuery execute the Elasticsearch query, adds a document into events index is query result satisfies alert conditions.
// It sets and returns a GlobalAlert status with the last executed query time, last time an event was generated, health status and error conditions if unhealthy.
func (e *service) executeQuery() {
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
		e.globalAlert.Status.Healthy = true
		e.globalAlert.Status.ErrorConditions = nil
		e.lmaESClient.BulkProcessorFlush()
		return
	}

	if e.bulkFlushErrored {
		return
	}
	e.globalAlert.Status.Healthy = true
	e.globalAlert.Status.ErrorConditions = nil
	return
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
			switch value.(type) {
			case string:
				description = strings.Replace(description, fmt.Sprintf("${%s}", v), value.(string), 1)
			case int64:
				description = strings.Replace(description, fmt.Sprintf("${%s}", v), strconv.FormatInt(value.(int64), 10), 1)
			case float64:
				description = strings.Replace(description, fmt.Sprintf("${%s}", v), strconv.FormatFloat(value.(float64), 'f', 1, 64), 1)
			default:
				log.Warnf("failed to build summary or description for alert %s due to unsupported value type for variable %s", e.globalAlert.Name, v)
			}
		}
	}
	return description
}

// extractEventData checks the given record object for keys that are defined in lmaAPI.EventsData,
// for each key found, it assigns them to lmaAPI.EventsData and removes it from the record object.
func extractEventData(record JsonObject) lmaAPI.EventsData {
	var e lmaAPI.EventsData
	if val, ok := record["source_ip"].(string); ok {
		e.SourceIP = &val
		delete(record, "source_ip")
	}
	if val, ok := record["source_port"].(int64); ok {
		e.SourcePort = &val
		delete(record, "source_port")
	}
	if val, ok := record["source_namespace"].(string); ok {
		e.SourceNamespace = val
		delete(record, "source_namespace")
	}
	if val, ok := record["source_name"].(string); ok {
		e.SourceName = val
		delete(record, "source_name")
	}
	if val, ok := record["source_name_aggr"].(string); ok {
		e.SourceNameAggr = val
		delete(record, "source_name_aggr")
	}
	if val, ok := record["dest_ip"].(string); ok {
		e.DestIP = &val
		delete(record, "dest_ip")
	}
	if val, ok := record["dest_port"].(int64); ok {
		e.DestPort = &val
		delete(record, "dest_port")
	}
	if val, ok := record["dest_namespace"].(string); ok {
		e.DestNamespace = val
		delete(record, "dest_namespace")
	}
	if val, ok := record["dest_name"].(string); ok {
		e.DestName = val
		delete(record, "dest_name")
	}
	if val, ok := record["dest_name_aggr"].(string); ok {
		e.DestNameAggr = val
		delete(record, "dest_name_aggr")
	}
	e.Record = record
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
