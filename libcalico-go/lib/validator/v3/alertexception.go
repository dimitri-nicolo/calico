// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package v3

import (
	"fmt"
	"reflect"

	validator "gopkg.in/go-playground/validator.v9"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
)

func validateAlertExceptionSpec(structLevel validator.StructLevel) {
	validateAlertExceptionPeriod(structLevel)
	validateAlertExceptionQuery(structLevel)
}

func getAlertExceptionSpec(structLevel validator.StructLevel) api.AlertExceptionSpec {
	return structLevel.Current().Interface().(api.AlertExceptionSpec)
}

func validateAlertExceptionPeriod(structLevel validator.StructLevel) {
	s := getAlertExceptionSpec(structLevel)

	if s.Period != nil && s.Period.Duration != 0 && s.Period.Duration < api.AlertExceptionMinPeriod {
		structLevel.ReportError(
			reflect.ValueOf(s.Period),
			"Period",
			"",
			reason(fmt.Sprintf("period %s < %s", s.Period, api.AlertExceptionMinPeriod)),
			"",
		)
	}
}

func validateAlertExceptionQuery(structLevel validator.StructLevel) {
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
