// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package v3

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	validator "gopkg.in/go-playground/validator.v9"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/validator/v3/query"
)

func validateGlobalAlertSpec(structLevel validator.StructLevel) {
	validateGlobalAlertPeriod(structLevel)
	validateGlobalAlertLookback(structLevel)
	validateGlobalAlertQuery(structLevel)
	validateGlobalAlertDescription(structLevel)
	validateGlobalAlertMetric(structLevel)

	/*
		We intentionally do not validate field or aggregation names. Fields do need to be numeric for most of the
		metrics and aggregation keys do need to exist for them to make sense.
	*/
}

func validateGlobalAlertPeriod(structLevel validator.StructLevel) {
	s := structLevel.Current().Interface().(api.GlobalAlertSpec)

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
	s := structLevel.Current().Interface().(api.GlobalAlertSpec)

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
	s := structLevel.Current().Interface().(api.GlobalAlertSpec)

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

// validateGlobalAlertDescription validates that there are no unreferenced fields in the description
func validateGlobalAlertDescription(structLevel validator.StructLevel) {
	s := structLevel.Current().Interface().(api.GlobalAlertSpec)
	if variables, err := extractVariablesFromDescriptionTemplate(s.Description); err != nil {
		structLevel.ReportError(
			reflect.ValueOf(s.DataSet),
			"Description",
			"",
			reason(fmt.Sprintf("invalid description: %s: %s", s.Description, err)),
			"",
		)
	} else {
		for _, key := range variables {
			if key == "" {
				structLevel.ReportError(
					reflect.ValueOf(s.Description),
					"Description",
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
					reflect.ValueOf(s.Description),
					"Description",
					"",
					reason("invalid description: "+s.Description),
					"",
				)
			}
		}
	}
}

func validateGlobalAlertMetric(structLevel validator.StructLevel) {
	s := structLevel.Current().Interface().(api.GlobalAlertSpec)
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
