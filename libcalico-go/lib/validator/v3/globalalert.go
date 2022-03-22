// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package v3

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	validator "gopkg.in/go-playground/validator.v9"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

const (
	GlobalAlertDetectorTemplateNamePrefix = "tigera.io.detector."
)

var (
	ADDetectorsGlobalAlertTemplateNameSet = func() map[string]bool {
		return map[string]bool{
			GlobalAlertDetectorTemplateNamePrefix + "dga":                   true,
			GlobalAlertDetectorTemplateNamePrefix + "http-connection-spike": true,
			GlobalAlertDetectorTemplateNamePrefix + "http-response-codes":   true,
			GlobalAlertDetectorTemplateNamePrefix + "http-verbs":            true,
			GlobalAlertDetectorTemplateNamePrefix + "ip-sweep":              true,
			GlobalAlertDetectorTemplateNamePrefix + "port-scan":             true,
			GlobalAlertDetectorTemplateNamePrefix + "generic-dns":           true,
			GlobalAlertDetectorTemplateNamePrefix + "time-series-dns":       true,
			GlobalAlertDetectorTemplateNamePrefix + "generic-flows":         true,
			GlobalAlertDetectorTemplateNamePrefix + "time-series-flows":     true,
			GlobalAlertDetectorTemplateNamePrefix + "generic-l7":            true,
			GlobalAlertDetectorTemplateNamePrefix + "time-series-l7":        true,
			GlobalAlertDetectorTemplateNamePrefix + "dns-latency":           true,
			GlobalAlertDetectorTemplateNamePrefix + "l7-bytes":              true,
			GlobalAlertDetectorTemplateNamePrefix + "l7-latency":            true,
			GlobalAlertDetectorTemplateNamePrefix + "process-restarts":      true,
		}
	}

	ADDetectorsSet = func() map[string]bool {
		return map[string]bool{
			"dga":                   true,
			"http_connection_spike": true,
			"http_response_codes":   true,
			"http_verbs":            true,
			"ip_sweep":              true,
			"port_scan":             true,
			"generic_dns":           true,
			"time_series_dns":       true,
			"generic_flows":         true,
			"time_series_flows":     true,
			"generic_l7":            true,
			"time_series_l7":        true,
			"dns_latency":           true,
			"l7_bytes":              true,
			"l7_latency":            true,
			"process_restarts":      true,
		}
	}
)

func validateGlobalAlertSpec(structLevel validator.StructLevel) {
	validateGlobalAlertDetector(structLevel)
	validateGlobalAlertPeriod(structLevel)
	validateGlobalAlertLookback(structLevel)
	validateGlobalAlertDataSet(structLevel)
	validateGlobalAlertQuery(structLevel)
	validateGlobalAlertDescriptionAndSummary(structLevel)
	validateGlobalAlertAggregateBy(structLevel)
	validateGlobalAlertMetric(structLevel)
}

func getGlobalAlertSpec(structLevel validator.StructLevel) api.GlobalAlertSpec {
	return structLevel.Current().Interface().(api.GlobalAlertSpec)
}

func validateGlobalAlertDetector(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)
	_, isValidJob := ADDetectorsSet()[s.Detector]
	if s.Type == api.GlobalAlertTypeAnomalyDetection && !isValidJob {
		structLevel.ReportError(
			reflect.ValueOf(s.Detector),
			"Detector",
			"",
			reason(fmt.Sprintf("unaccepted Detector for GlobalAlert of Type %s", s.Type)),
			"",
		)
	}
}

func validateGlobalAlertDataSet(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	if (len(s.Type) == 0 || s.Type == api.GlobalAlertTypeUserDefined) && len(s.DataSet) == 0 {
		structLevel.ReportError(
			reflect.ValueOf(s.DataSet),
			"DataSet",
			"",
			reason(fmt.Sprintf("empty DataSet for GlobalAlert of Type %s", api.GlobalAlertTypeUserDefined)),
			"",
		)
	}
}

func validateGlobalAlertPeriod(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	if s.Period != nil && s.Period.Duration != 0 && s.Period.Duration < api.GlobalAlertMinPeriod {
		structLevel.ReportError(
			reflect.ValueOf(s.Period),
			"Period",
			"",
			reason(fmt.Sprintf("period %s < %s", s.Period, api.GlobalAlertMinPeriod)),
			"",
		)
	}
}

func validateGlobalAlertLookback(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	if s.Lookback != nil && s.Lookback.Duration != 0 && s.Lookback.Duration < api.GlobalAlertMinLookback {
		structLevel.ReportError(
			reflect.ValueOf(s.Lookback),
			"Lookback",
			"",
			reason(fmt.Sprintf("lookback %s < %s", s.Period, api.GlobalAlertMinLookback)),
			"",
		)
	}
}

// substituteVariables finds variables in the query string and replace them with values from GlobalAlertSpec.Substitutions.
func substituteVariables(s api.GlobalAlertSpec) (string, error) {
	out := s.Query
	variables, err := extractVariablesFromTemplate(out)
	if err != nil {
		return out, err
	}

	if len(variables) > 0 {
		for _, variable := range variables {
			sub, err := findSubstitutionByVariableName(s, variable)
			if err != nil {
				return out, err
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
	return out, nil
}

// validateGlobalAlertQuery substitutes all variables in the query string and validates it by the query parser.
func validateGlobalAlertQuery(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	if qs, err := substituteVariables(s); err != nil {
		structLevel.ReportError(
			reflect.ValueOf(s.Query),
			"Query",
			"",
			reason("invalid query: "+err.Error()),
			"",
		)
	} else if q, err := query.ParseQuery(qs); err != nil {
		structLevel.ReportError(
			reflect.ValueOf(s.Query),
			"Query",
			"",
			reason("invalid query: "+err.Error()),
			"",
		)
	} else {
		switch s.DataSet {
		case api.GlobalAlertDataSetAudit:
			if err := query.Validate(q, query.IsValidAuditAtom); err != nil {
				structLevel.ReportError(
					reflect.ValueOf(s.Query),
					"Query",
					"",
					reason("invalid query: "+err.Error()),
					"",
				)
			}
		case api.GlobalAlertDataSetDNS:
			if err := query.Validate(q, query.IsValidDNSAtom); err != nil {
				structLevel.ReportError(
					reflect.ValueOf(s.Query),
					"Query",
					"",
					reason("invalid query: "+err.Error()),
					"",
				)
			}
		case api.GlobalAlertDataSetFlows:
			if err := query.Validate(q, query.IsValidFlowsAtom); err != nil {
				structLevel.ReportError(
					reflect.ValueOf(s.Query),
					"Query",
					"",
					reason("invalid query: "+err.Error()),
					"",
				)
			}
		case api.GlobalAlertDataSetWAF:
			if err := query.Validate(q, query.IsValidWAFAtom); err != nil {
				structLevel.ReportError(
					reflect.ValueOf(s.Query),
					"Query",
					"",
					reason("invalid query: "+err.Error()),
					"",
				)
			}
		case api.GlobalAlertDataSetVulnerability:
			if err := query.Validate(q, query.IsValidVulnerabilityAtom); err != nil {
				structLevel.ReportError(
					reflect.ValueOf(s.Query),
					"Query",
					"",
					reason("invalid query: "+err.Error()),
					"",
				)
			}
		}
	}
}

// validateGlobalAlertDescriptionAndSummary validates that there are no unreferenced fields in the description and summary
func validateGlobalAlertDescriptionAndSummary(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	validateGlobalAlertDescriptionOrSummaryContents(s.Description, "Description", structLevel, s)
	if s.Summary != "" {
		validateGlobalAlertDescriptionOrSummaryContents(s.Summary, "Summary", structLevel, s)
	}
}

func validateGlobalAlertDescriptionOrSummaryContents(description, fieldName string, structLevel validator.StructLevel, s api.GlobalAlertSpec) {
	if variables, err := extractVariablesFromTemplate(description); err != nil {
		structLevel.ReportError(
			reflect.ValueOf(s.DataSet),
			fieldName,
			"",
			reason(fmt.Sprintf("invalid %s: %s: %s", strings.ToLower(fieldName), description, err)),
			"",
		)
	} else {
		for _, key := range variables {
			if key == "" {
				structLevel.ReportError(
					reflect.ValueOf(description),
					fieldName,
					"",
					reason("empty variable name"),
					"",
				)
				break
			}

			if key == s.Metric {
				continue
			}
			var found bool
			for _, ag := range s.AggregateBy {
				if key == ag {
					found = true
					break
				}
			}
			if !found {
				structLevel.ReportError(
					reflect.ValueOf(description),
					fieldName,
					"",
					reason(fmt.Sprintf("invalid %s: %s", strings.ToLower(fieldName), description)),
					"",
				)
			}
		}
	}
}

func validateGlobalAlertAggregateBy(structLevel validator.StructLevel) {
	// We intentionally do not validate field or aggregation names. Fields do need to be numeric for most of the
	// metrics and aggregation keys do need to exist for them to make sense.
	s := getGlobalAlertSpec(structLevel)

	if s.DataSet == api.GlobalAlertDataSetVulnerability {
		if len(s.AggregateBy) > 0 {
			structLevel.ReportError(
				reflect.ValueOf(s.AggregateBy),
				"AggregateBy",
				"",
				reason("vulnerability dataset doesn't support aggregateBy field"),
				"",
			)
		}
	}
}

func validateGlobalAlertMetric(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	switch s.Metric {
	case api.GlobalAlertMetricAvg, api.GlobalAlertMetricMax, api.GlobalAlertMetrixMin, api.GlobalAlertMetricSum:
		if s.Field == "" {
			structLevel.ReportError(
				reflect.ValueOf(s.Field),
				"Field",
				"",
				reason(fmt.Sprintf("metric %s requires a field", s.Metric)),
				"",
			)
		}
	case api.GlobalAlertMetricCount:
		if s.Field != "" {
			structLevel.ReportError(
				reflect.ValueOf(s.Field),
				"Field",
				"",
				reason(fmt.Sprintf("metric %s cannot be applied to a field", s.Metric)),
				"",
			)
		}
	case "":
		if s.Field != "" {
			structLevel.ReportError(
				reflect.ValueOf(s.Field),
				"Field",
				"",
				reason("field without metric is invalid"),
				"",
			)
		}
	default:
		structLevel.ReportError(
			reflect.ValueOf(s.Metric),
			"Metric",
			"",
			reason(fmt.Sprintf("invalid metric: %s", s.Metric)),
			"",
		)
	}

	if s.Metric != "" && s.Condition == "" {
		structLevel.ReportError(
			reflect.ValueOf(s.Metric),
			"Metric",
			"",
			reason(fmt.Sprintf("metric %s without condition", s.Metric)),
			"",
		)
	}
}

// extractVariablesFromTemplate extracts variables from a template string.
// Variables are defined by starting with a dollar sign and enclosed by curly braces.
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
			return nil, errors.New("unterminated }")
		}
		res = append(res, s[:end])
		s = s[end+1:]
	}
	return res, nil
}

// findSubstitutionByVariableName finds the substitution from GlobalAlertSpec.Substitutions by the variable name.
// Only one substitution will be returned. If no substitution or more than one substitution is found,
// an error will be returned.
func findSubstitutionByVariableName(s api.GlobalAlertSpec, variable string) (*api.GlobalAlertSubstitution, error) {
	var substitution *api.GlobalAlertSubstitution
	for _, sub := range s.Substitutions {
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
	return nil, fmt.Errorf("substition not found for variable %s", variable)
}
