// Copyright (c) 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

type MockStatser struct {
	ErrorConditions []v3.ErrorCondition
}

var _ = Statser(&MockStatser{})

func (s *MockStatser) Run(context.Context) {
}

func (s *MockStatser) Close() {
}

func (s *MockStatser) Status() v3.GlobalAlertStatus {
	return v3.GlobalAlertStatus{
		ErrorConditions: append(s.ErrorConditions),
	}
}

func (s *MockStatser) Error(t string, err error) {
	s.ErrorConditions = append(s.ErrorConditions, v3.ErrorCondition{Type: t, Message: err.Error()})
}

func (s *MockStatser) ClearError(t string) {
	ec := []v3.ErrorCondition{}

	for _, errorCondition := range s.ErrorConditions {
		if errorCondition.Type != t {
			ec = append(ec, errorCondition)
		}
	}

	s.ErrorConditions = ec
}
