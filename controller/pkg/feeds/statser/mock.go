// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"time"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockStatser struct {
	lastSuccessfulSync   time.Time
	lastSuccessfulSearch time.Time
	errorConditions      []v3.ErrorCondition
}

func (s *MockStatser) Run(context.Context) {
}

func (s *MockStatser) Close() {
}

func (s *MockStatser) Status() v3.GlobalThreatFeedStatus {
	return v3.GlobalThreatFeedStatus{
		LastSuccessfulSync:   v1.Time{Time: s.lastSuccessfulSync},
		LastSuccessfulSearch: v1.Time{Time: s.lastSuccessfulSearch},
		ErrorConditions:      append(s.errorConditions),
	}
}

func (s *MockStatser) SuccessfulSync() {
	s.lastSuccessfulSync = time.Now()
}

func (s *MockStatser) SuccessfulSearch() {
	s.lastSuccessfulSearch = time.Now()
}

func (s *MockStatser) Error(t string, err error) {
	s.errorConditions = append(s.errorConditions, v3.ErrorCondition{Type: t, Message: err.Error()})
}

func (s *MockStatser) ClearError(t string) {
	ec := []v3.ErrorCondition{}

	for _, errorCondition := range s.errorConditions {
		if errorCondition.Type != t {
			ec = append(ec, errorCondition)
		}
	}

	s.errorConditions = ec
}
