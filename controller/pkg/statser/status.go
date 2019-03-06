package statser

import "time"

type Status struct {
	LastSuccessfulSync time.Time
	LastSuccessfulSearch time.Time
	ErrorConditions []ErrorCondition
}

type ErrorCondition struct {
	Type string
	Message string
}

func (s *Status) Status() *Status {
	return s
}

func (s *Status) SuccessfulSync() {
	s.LastSuccessfulSync = time.Now()
}

func (s *Status) SuccessfulSearch() {
	s.LastSuccessfulSearch = time.Now()
}

func (s *Status) Error(t string, err error) {
	s.ErrorConditions = append(s.ErrorConditions, ErrorCondition{Type: t, Message: err.Error()})
}

func (s *Status) ClearError(t string) {
	ec := []ErrorCondition{}

	for _, errorCondition := range s.ErrorConditions {
		if errorCondition.Type != t {
			ec = append(ec, errorCondition)
		}
	}

	s.ErrorConditions = ec
}
