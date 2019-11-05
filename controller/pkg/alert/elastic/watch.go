// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lithammer/dedent"
	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const (
	AlertEventType    = "alert"
	DefaultPeriod     = time.Minute * 5
	DefaultLookback   = time.Minute * 10
	IndexActionName   = "index_events"
	QueryAggTerms     = "terms"
	QueryAggTermsSize = 10000
	QuerySize         = 10000
)

var (
	comparatorMap = map[string]string{
		"eq":  "=",
		"ne":  "!=",
		"lt":  "<",
		"lte": "<=",
		"gt":  ">",
		"gte": ">=",
	}
)

func Watch(alert v3.GlobalAlert) (*elastic.PutWatchBody, error) {
	input, err := Input(alert)
	if err != nil {
		return nil, err
	}

	return &elastic.PutWatchBody{
		Trigger:   Trigger(Period(alert)),
		Input:     input,
		Condition: Condition(alert),
		Transform: Transform(alert),
		Actions: map[string]elastic.Action{
			IndexActionName: {
				Transform: ActionTransform(alert),
				Index: &elastic.IndexAction{
					Index: elastic.EventIndex,
				},
			},
		},
		Metadata: JsonObject{
			"alert": alert.Spec,
		},
	}, nil
}

func Condition(alert v3.GlobalAlert) *elastic.Condition {
	var valueKey string
	switch alert.Spec.Metric {
	case "":
		return &elastic.Condition{Always: true}
	case libcalicov3.GlobalAlertMetricCount:
		valueKey = "doc_count"
	default:
		valueKey = fmt.Sprintf("get(%q).value", alert.Spec.Field)
	}

	if _, ok := comparatorMap[alert.Spec.Condition]; !ok {
		panic(fmt.Errorf("unknown comparator: %s", alert.Spec.Condition))
	}

	if len(alert.Spec.AggregateBy) == 0 {
		switch alert.Spec.Metric {
		case libcalicov3.GlobalAlertMetricCount:
			return &elastic.Condition{
				Compare: &elastic.Comparison{
					Key:       "ctx.payload.hits.total",
					Operation: alert.Spec.Condition,
					Value:     alert.Spec.Threshold,
				},
			}
		default:
			return &elastic.Condition{
				Script: &elastic.Script{
					Language: "painless",
					Source: fmt.Sprintf("ctx.payload.aggregations.%s %s params.threshold",
						valueKey, comparatorMap[alert.Spec.Condition]),
					Params: JsonObject{
						"threshold": alert.Spec.Threshold,
					},
				},
			}
		}
	}

	/*
		Generates code like:

		ctx.payload.aggregations.client_name.buckets.stream()
		.anyMatch(
			t0 -> t0.client_name_aggr.buckets.stream()
			.anyMatch(
				t1 -> t1.client_namespace.buckets.stream()
				.anyMatch(
					t2 -> t2.doc_count >= params.threshold
				)
			)
		)
	*/
	source := strings.Builder{}
	source.WriteString("ctx.payload.aggregations")

	for idx, term := range alert.Spec.AggregateBy {
		source.WriteString(fmt.Sprintf(`.get(%q).buckets.stream().anyMatch(t%d -> t%d`, term, idx, idx))
	}

	source.WriteString(fmt.Sprintf(".%s %s params.threshold\n", valueKey, comparatorMap[alert.Spec.Condition]))

	for _, _ = range alert.Spec.AggregateBy {
		source.WriteString(")\n")
	}

	return &elastic.Condition{
		Script: &elastic.Script{
			Language: "painless",
			Source:   util.PainlessFmt(source.String()),
			Params: map[string]interface{}{
				"threshold": alert.Spec.Threshold,
			},
		},
	}
}

func Transform(alert v3.GlobalAlert) *elastic.Transform {
	if len(alert.Spec.AggregateBy) == 0 {
		switch alert.Spec.Metric {
		case "":
			return &elastic.Transform{
				Script: elastic.Script{
					Language: "painless",
					Source:   "[ '_value': ctx.payload.hits.hits.stream().map(t -> t._source).collect(Collectors.toList()) ]",
				},
			}
		case libcalicov3.GlobalAlertMetricCount:
			return &elastic.Transform{
				Script: elastic.Script{
					Language: "painless",
					Source:   `["_value": [[ "count": ctx.payload.hits.total ]]]`,
				},
			}
		default:
			return &elastic.Transform{
				Script: elastic.Script{
					Language: "painless",
					Source:   fmt.Sprintf(`[ "_value": [[ "%s": ctx.payload.aggregations.get(%q).value ]]]`, alert.Spec.Metric, alert.Spec.Field),
				},
			}
		}
	}

	var description string
	var key string
	switch alert.Spec.Metric {
	case libcalicov3.GlobalAlertMetricCount:
		description = "count"
		key = "doc_count"
	default:
		description = fmt.Sprintf("%s", alert.Spec.Metric)
		key = fmt.Sprintf("get(%q).value", alert.Spec.Field)
	}

	/*
		Generates code like:

		ctx.payload.aggregations.qname0.buckets.stream()
		.map(t0 -> t0.qname1.buckets.stream()
		.filter(t1 -> t1.doc_count >= params.threshold)
		.map(t1 -> ["qname0": t0.key, "qname1": t1.key, "count":t1.doc_count])
		.collect(Collectors.toList()))
		.flatMap(Collection::stream)
		.collect(Collectors.toList())
	*/

	script := strings.Builder{}
	script.WriteString("ctx.payload.aggregations")

	for idx, field := range alert.Spec.AggregateBy {
		if idx > 0 {
			script.WriteString(fmt.Sprintf(`.map(t%d -> t%d`, idx-1, idx-1))
		}
		script.WriteString(fmt.Sprintf(`.get(%q).buckets.stream()`, field))
	}

	maxIdx := len(alert.Spec.AggregateBy) - 1
	if alert.Spec.Metric != "" {
		if c, ok := comparatorMap[alert.Spec.Condition]; !ok {
			panic(fmt.Errorf("unknown comparator: %s", alert.Spec.Condition))
		} else {
			script.WriteString(fmt.Sprintf(`.filter(t%d -> t%d.%s %s params.threshold)`, maxIdx, maxIdx, key, c))
		}
	}

	script.WriteString(fmt.Sprintf(`.map(t%d -> [`, maxIdx))
	var row []string
	for idx, key := range alert.Spec.AggregateBy {
		row = append(row, fmt.Sprintf("%q: t%d.key", key, idx))
	}
	if alert.Spec.Metric != "" {
		row = append(row, fmt.Sprintf("%q: t%d.%s", description, maxIdx, key))
	}
	script.WriteString(strings.Join(row, ", "))
	script.WriteString("]).collect(Collectors.toList())")

	for i := 0; i < len(alert.Spec.AggregateBy)-1; i++ {
		script.WriteString(").flatMap(Collection::stream).collect(Collectors.toList())")
	}

	return &elastic.Transform{
		Script: elastic.Script{
			Language: "painless",
			Source:   util.PainlessFmt(script.String()),
			Params: map[string]interface{}{
				"threshold": alert.Spec.Threshold,
			},
		},
	}
}

func Trigger(period time.Duration) elastic.Trigger {
	return elastic.Trigger{
		Schedule: elastic.Schedule{
			Interval: &elastic.Interval{
				Duration: period,
			},
		},
	}
}

func Period(alert v3.GlobalAlert) time.Duration {
	if alert.Spec.Period == nil {
		return DefaultPeriod
	}
	period := alert.Spec.Period.Duration
	if period <= 0 {
		return DefaultPeriod
	}
	return period
}

func Lookback(alert v3.GlobalAlert) time.Duration {
	if alert.Spec.Lookback == nil {
		return DefaultLookback
	}
	lookback := alert.Spec.Lookback.Duration
	if lookback <= 0 {
		return DefaultLookback
	}
	return lookback
}

func Input(alert v3.GlobalAlert) (*elastic.Input, error) {
	aggs := MetricQueryAggs(alert.Spec.Field, alert.Spec.Metric, nil)
	aggs = TermQueryAggs(alert.Spec.AggregateBy, aggs)

	indices, err := Indices(alert.Spec.DataSet)
	if err != nil {
		return nil, err
	}
	q, err := Query(alert)
	if err != nil {
		return nil, err
	}

	var timeField string
	switch alert.Spec.DataSet {
	case libcalicov3.GlobalAlertDataSetDNS, libcalicov3.GlobalAlertDataSetFlows:
		timeField = "start_time"
	case libcalicov3.GlobalAlertDataSetAudit:
		timeField = "timestamp"
	default:
		return nil, fmt.Errorf("unknown dataset: %s", alert.Spec.DataSet)
	}

	queryObj := JsonObject{
		"bool": JsonObject{
			"must":   q,
			"filter": LookbackFilter(Lookback(alert), timeField),
		},
	}
	body := JsonObject{
		"query": queryObj,
	}
	if aggs != nil {
		body["size"] = 0
		body["aggs"] = aggs
	} else if alert.Spec.Metric == libcalicov3.GlobalAlertMetricCount {
		body["size"] = 0
	} else {
		body["size"] = QuerySize
	}

	return &elastic.Input{
		Search: &elastic.Search{
			Request: elastic.SearchRequest{
				Indices: indices,
				Body:    body,
				IndicesOptions: &elastic.IndicesOptions{
					AllowNoIndices: util.BoolPtr(true),
				},
			},
		},
	}, nil
}

func LookbackFilter(lookback time.Duration, timeField string) JsonObject {
	return JsonObject{
		"range": JsonObject{
			timeField: JsonObject{
				"gte":    fmt.Sprintf("{{ctx.trigger.scheduled_time}}||-%ds", lookback.Milliseconds()/1000),
				"lte":    "{{ctx.trigger.scheduled_time}}",
				"format": "strict_date_optional_time||epoch_millis",
			},
		},
	}
}

func Indices(dataSet string) ([]string, error) {
	switch dataSet {
	case libcalicov3.GlobalAlertDataSetAudit:
		return []string{elastic.AuditIndex}, nil
	case libcalicov3.GlobalAlertDataSetDNS:
		return []string{elastic.DNSLogIndex}, nil
	case libcalicov3.GlobalAlertDataSetFlows:
		return []string{elastic.FlowLogIndex}, nil
	default:
		return nil, fmt.Errorf("unknown dataset: %s", dataSet)
	}
}

func Query(alert v3.GlobalAlert) (interface{}, error) {
	q, err := query.ParseQuery(alert.Spec.Query)
	if err != nil {
		return nil, err
	}

	var converter ElasticQueryConverter
	switch alert.Spec.DataSet {
	case libcalicov3.GlobalAlertDataSetAudit:
		err := query.Validate(q, query.IsValidAuditAtom)
		if err != nil {
			return nil, err
		}
		converter = NewAuditConverter()
	case libcalicov3.GlobalAlertDataSetDNS:
		err := query.Validate(q, query.IsValidDNSAtom)
		if err != nil {
			return nil, err
		}
		converter = NewDNSConverter()
	case libcalicov3.GlobalAlertDataSetFlows:
		err := query.Validate(q, query.IsValidFlowsAtom)
		if err != nil {
			return nil, err
		}
		converter = NewFlowsConverter()
	default:
		return nil, fmt.Errorf("unknown dataset: %s", alert.Spec.DataSet)
	}

	return converter.Convert(q), nil
}

type QueryAgg struct {
	Field       string
	Aggregation string
	Child       *QueryAgg
}

func (q QueryAgg) MarshalJSON() ([]byte, error) {
	aggregation := JsonObject{
		"field": q.Field,
	}
	if q.Aggregation == QueryAggTerms {
		aggregation["size"] = QueryAggTermsSize
	}
	res := JsonObject{
		q.Aggregation: aggregation,
	}
	if q.Child != nil {
		res["aggs"] = q.Child
	}
	return json.Marshal(JsonObject{q.Field: res})
}

func TermQueryAggs(terms []string, agg *QueryAgg) *QueryAgg {
	for i := len(terms) - 1; i >= 0; i-- {
		term := terms[i]
		agg = &QueryAgg{Field: term, Aggregation: QueryAggTerms, Child: agg}
	}
	return agg
}

func MetricQueryAggs(field, metric string, agg *QueryAgg) *QueryAgg {
	switch metric {
	case "", libcalicov3.GlobalAlertMetricCount:
		return nil
	}

	return &QueryAgg{
		Field:       field,
		Aggregation: metric,
		Child:       agg,
	}
}

func GenerateDescriptionFunction(s string) string {
	var sb strings.Builder
	sb.WriteString("String description(def t) {\n\tStringBuffer sb = new StringBuffer();\n")

	for s != "" {
		start := strings.Index(s, "${")
		if start < 0 {
			break
		}
		sb.WriteString(fmt.Sprintf("\tsb.append(%q);\n", s[:start]))
		s = s[start+2:]
		end := strings.Index(s, "}")
		if end < 0 {
			s = "${" + s
			break
		}

		sb.WriteString(fmt.Sprintf("\tsb.append(resolve(%q, t));\n", s[:end]))
		s = s[end+1:]
	}

	if s != "" {
		sb.WriteString(fmt.Sprintf("\tsb.append(%q);\n", s))
	}

	sb.WriteString("\tsb.toString();\n}\n")

	return sb.String()
}

var ResolveCode = strings.TrimSpace(dedent.Dedent(`
String resolve(String key, def t) {
	if (t instanceof ArrayList) {
		return t.stream().map(s -> resolve(key, s)).collect(Collectors.toList()).toString()
	}
	if (!(t instanceof HashMap)) {
		return "null"
	}
	if (t.containsKey(key)) {
		return String.valueOf(t[key])
	}
	String[] parts = key.splitOnToken(".", 2);
	if (parts.length > 1 && t.containsKey(parts[0])) {
		return resolve(parts[1], t[parts[0]])
	}
	"null"
}
`)) + "\n"

var actionTransformCode = strings.TrimSpace(dedent.Dedent(`
	return [
		'_doc': ctx.payload._value.stream()
			.map(t -> {[
				'time': ctx.trigger.triggered_time,
				'type': params.type,
				'description': description(t),
				'severity': params.severity,
				'record': t
			]})
			.collect(Collectors.toList())
	]
`))

func ActionTransform(alert v3.GlobalAlert) *elastic.Transform {
	return &elastic.Transform{
		elastic.Script{
			// This doesn't work for nested
			Language: "painless",
			Source:   util.PainlessFmt(ResolveCode + GenerateDescriptionFunction(alert.Spec.Description) + actionTransformCode),
			Params: JsonObject{
				"type":        AlertEventType,
				"description": alert.Spec.Description,
				"severity":    alert.Spec.Severity,
			},
		},
	}
}
