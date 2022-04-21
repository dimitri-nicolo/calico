// Copyright 2019 Tigera Inc. All rights reserved.

package errorcondition

import (
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const MaxErrors = 10

func AddError(status *v3.GlobalThreatFeedStatus, errType string, err error) {
	errorConditions := status.ErrorConditions
	errorConditions = append(errorConditions, v3.ErrorCondition{Type: errType, Message: err.Error()})
	if len(errorConditions) > MaxErrors {
		errorConditions = errorConditions[1:]
	}
	status.ErrorConditions = errorConditions
}

func ClearError(status *v3.GlobalThreatFeedStatus, errType string) {
	errorConditions := make([]v3.ErrorCondition, 0)
	for _, errorCondition := range status.ErrorConditions {
		if errorCondition.Type != errType {
			errorConditions = append(errorConditions, errorCondition)
		}
	}
	status.ErrorConditions = errorConditions
}
