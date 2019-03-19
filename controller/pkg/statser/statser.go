// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"sync"
	"time"
)

const (
	ElasticSyncFailed          = "ElasticSyncFailed"
	GlobalNetworkSetSyncFailed = "GlobalNetworkSetSyncFailed"
	GarbageCollectionFailed    = "GarbageCollectionFailed"
	PullFailed                 = "PullFailed"
	SearchFailed               = "SearchFailed"
)

type Statser interface {
	Status() Status
	SuccessfulSync()
	SuccessfulSearch()
	Error(string, error)
	ClearError(string)
}

type Status struct {
	LastSuccessfulSync   time.Time
	LastSuccessfulSearch time.Time
	ErrorConditions      []ErrorCondition
}

type ErrorCondition struct {
	Type    string
	Message string
}

func NewStatser() Statser {
	return &statser{errorConditions: make(map[string][]ErrorCondition)}
}

type statser struct {
	lastSuccessfulSync   time.Time
	lastSuccessfulSearch time.Time
	errorConditions      map[string][]ErrorCondition
	lock                 sync.RWMutex
}

func (s *statser) Status() Status {
	s.lock.RLock()
	defer s.lock.RUnlock()

	res := Status{
		LastSuccessfulSync:   s.lastSuccessfulSync,
		LastSuccessfulSearch: s.lastSuccessfulSearch,
		ErrorConditions:      make([]ErrorCondition, len(s.errorConditions)),
	}

	for _, conditions := range s.errorConditions {
		res.ErrorConditions = append(res.ErrorConditions, conditions...)
	}

	return res
}

func (s *statser) SuccessfulSync() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lastSuccessfulSync = time.Now()
}

func (s *statser) SuccessfulSearch() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lastSuccessfulSearch = time.Now()
}

func (s *statser) Error(t string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.errorConditions[t] = append(s.errorConditions[t], ErrorCondition{Type: t, Message: err.Error()})
}

func (s *statser) ClearError(t string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.errorConditions, t)
}
