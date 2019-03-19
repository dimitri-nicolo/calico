// Copyright 2019 Tigera Inc. All rights reserved.

package mock

import (
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

type Statser struct {
	lastSuccessfulSync   time.Time
	lastSuccessfulSearch time.Time
	errorConditions      []statser.ErrorCondition
}

func (s *Statser) Status() statser.Status {
	return statser.Status{
		LastSuccessfulSync:   s.lastSuccessfulSync,
		LastSuccessfulSearch: s.lastSuccessfulSearch,
		ErrorConditions:      append(s.errorConditions),
	}
}

func (s *Statser) SuccessfulSync() {
	s.lastSuccessfulSync = time.Now()
}

func (s *Statser) SuccessfulSearch() {
	s.lastSuccessfulSearch = time.Now()
}

func (s *Statser) Error(t string, err error) {
	s.errorConditions = append(s.errorConditions, statser.ErrorCondition{Type: t, Message: err.Error()})
}

func (s *Statser) ClearError(t string) {
	ec := []statser.ErrorCondition{}

	for _, errorCondition := range s.errorConditions {
		if errorCondition.Type != t {
			ec = append(ec, errorCondition)
		}
	}

	s.errorConditions = ec
}
