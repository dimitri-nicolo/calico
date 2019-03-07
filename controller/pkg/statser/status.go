package statser

import (
	"sync"
	"time"
)

type Status struct {
	LastSuccessfulSync   time.Time
	LastSuccessfulSearch time.Time
	ErrorConditions      []ErrorCondition
	lock                 sync.Mutex
}

type ErrorCondition struct {
	Type    string
	Message string
}

func (s *Status) Status() *Status {
	return s
}

func (s *Status) SuccessfulSync() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.LastSuccessfulSync = time.Now()
}

func (s *Status) SuccessfulSearch() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.LastSuccessfulSearch = time.Now()
}

func (s *Status) Error(t string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.ErrorConditions = append(s.ErrorConditions, ErrorCondition{Type: t, Message: err.Error()})
}

func (s *Status) ClearError(t string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ec := []ErrorCondition{}

	for _, errorCondition := range s.ErrorConditions {
		if errorCondition.Type != t {
			ec = append(ec, errorCondition)
		}
	}

	s.ErrorConditions = ec
}
