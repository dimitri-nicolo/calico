// Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.

package v3

import (
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	validator "gopkg.in/go-playground/validator.v9"
)

func getGlobalAlertTemplate(structLevel validator.StructLevel) api.GlobalAlertTemplate {
	return structLevel.Current().Interface().(api.GlobalAlertTemplate)
}

func validateGlobalAlertTemplate(structLevel validator.StructLevel) {
	globalAlertTemplate := getGlobalAlertTemplate(structLevel)
	validateGlobalAlertSpec(structLevel, globalAlertTemplate.Name, globalAlertTemplate.Spec)
}
