// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package v3

import (
	"reflect"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	validator "gopkg.in/go-playground/validator.v9"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

func validateAlertExceptionSpec(structLevel validator.StructLevel) {
	validateAlertExceptionTime(structLevel)
	validateAlertExceptionSelector(structLevel)
}

func getAlertExceptionSpec(structLevel validator.StructLevel) api.AlertExceptionSpec {
	return structLevel.Current().Interface().(api.AlertExceptionSpec)
}

func validateAlertExceptionTime(structLevel validator.StructLevel) {
	s := getAlertExceptionSpec(structLevel)

	if s.StartTime.IsZero() {
		structLevel.ReportError(
			reflect.ValueOf(s.EndTime),
			"StartTime",
			"",
			reason("invalid StartTime"),
			"",
		)
	} else if s.EndTime != nil {
		if s.EndTime.IsZero() {
			structLevel.ReportError(
				reflect.ValueOf(s.EndTime),
				"EndTime",
				"",
				reason("invalid EndTime"),
				"",
			)
		} else if s.EndTime.Before(&s.StartTime) {
			structLevel.ReportError(
				reflect.ValueOf(s.EndTime),
				"EndTime",
				"",
				reason("EndTime can not be earlier than StartTime"),
				"",
			)
		}
	}
}

func validateAlertExceptionSelector(structLevel validator.StructLevel) {
	s := getAlertExceptionSpec(structLevel)

	if q, err := query.ParseQuery(s.Selector); err != nil {
		structLevel.ReportError(
			reflect.ValueOf(s.Selector),
			"Selector",
			"",
			reason("invalid selector: "+err.Error()),
			"",
		)
	} else if err := query.Validate(q, query.IsValidEventsKeysAtom); err != nil {
		structLevel.ReportError(
			reflect.ValueOf(s.Selector),
			"Selector",
			"",
			reason("invalid selector: "+err.Error()),
			"",
		)
	}
}
