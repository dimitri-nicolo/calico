// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"sync"
	"time"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

const (
	XPackRecordsFailed = "XPackRecordsFailed"
	FilterFailed       = "FilterFailed"
	StoreEventsFailed  = "StoreEventsFailed"
)

type Statser interface {
	Run(context.Context)
	Close()
	Status() Status
	SuccessfulSync()
	Error(string, error)
	ClearError(string)
}

type Status struct {
	LastSuccessfulSync metav1.Time
	ErrorConditions    []v3.ErrorCondition `json:"errorConditions"`
}

func NewStatser(name string) Statser {
	return &statser{
		name:            name,
		errorConditions: make(map[string][]v3.ErrorCondition),
	}
}

type statser struct {
	name                 string
	lastSuccessfulSync   time.Time
	lastSuccessfulSearch time.Time
	errorConditions      map[string][]v3.ErrorCondition
	lock                 sync.RWMutex
	once                 sync.Once
	cancel               context.CancelFunc
	enqueue              runloop.EnqueueFunc
}

func (s *statser) Run(ctx context.Context) {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)

		run, enqueue := runloop.OnDemand()
		s.enqueue = enqueue

		go run(ctx, func(ctx context.Context, i interface{}) {
			s.updateStatus(i.(Status))
		})
	})
}

func (s *statser) Close() {
	s.cancel()
}

func (s *statser) Status() Status {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.status()
}

func (s *statser) status() Status {
	res := Status{
		LastSuccessfulSync: metav1.Time{Time: s.lastSuccessfulSync},
		ErrorConditions:    make([]v3.ErrorCondition, 0),
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
	s.enqueue(s.status())
}

func (s *statser) Error(t string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.errorConditions[t] = append(s.errorConditions[t], v3.ErrorCondition{Type: t, Message: err.Error()})
	s.enqueue(s.status())
}

func (s *statser) ClearError(t string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.errorConditions, t)
	s.enqueue(s.status())
}

func (s *statser) updateStatus(status Status) {
	// noop
}
