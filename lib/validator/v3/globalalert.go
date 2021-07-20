// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package v3

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	validator "gopkg.in/go-playground/validator.v9"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
)

func validateGlobalAlertSpec(structLevel validator.StructLevel) {
	validateGlobalAlertPeriod(structLevel)
	validateGlobalAlertLookback(structLevel)
	validateGlobalAlertQuery(structLevel)
	validateGlobalAlertDescriptionAndSummary(structLevel)
	validateGlobalAlertMetric(structLevel)

	/*
		We intentionally do not validate field or aggregation names. Fields do need to be numeric for most of the
		metrics and aggregation keys do need to exist for them to make sense.
	*/
}

func getGlobalAlertSpec(structLevel validator.StructLevel) api.GlobalAlertSpec {
	return structLevel.Current().Interface().(api.GlobalAlertSpec)
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

func validateGlobalAlertQuery(structLevel validator.StructLevel) {
	s := getGlobalAlertSpec(structLevel)

	if q, err := query.ParseQuery(s.Query); err != nil {
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
	if variables, err := extractVariablesFromDescriptionTemplate(description); err != nil {
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

func extractVariablesFromDescriptionTemplate(s string) ([]string, error) {
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
