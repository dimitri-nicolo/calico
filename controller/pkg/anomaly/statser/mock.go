// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"time"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockStatser struct {
	LastSuccessfulSync time.Time
	ErrorConditions    []v3.ErrorCondition
}

func (s *MockStatser) Run(context.Context) {
}

func (s *MockStatser) Close() {
}

func (s *MockStatser) Status() Status {
	return Status{
		LastSuccessfulSync: v1.Time{Time: s.LastSuccessfulSync},
		ErrorConditions:    append(s.ErrorConditions),
	}
}

func (s *MockStatser) SuccessfulSync() {
	s.LastSuccessfulSync = time.Now()
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
