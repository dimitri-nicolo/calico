// Copyright (c) 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"sync"

	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	statserCommon "github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const (
	SyncFailed         = "SyncFailed"
	CompilationFailed  = "CompilationFailed"
	InstallationFailed = "InstallationFailed"
)

type Statser interface {
	Run(context.Context)
	Close()
	Status() libcalicov3.GlobalAlertStatus
	SuccessfulSync()
	SuccessfulSearch()
	Error(string, error)
	ClearError(string)
}

type statser struct {
	alert           *v3.GlobalAlert
	errorConditions *statserCommon.ErrorConditions
	lock            sync.RWMutex
	once            sync.Once
	cancel          context.CancelFunc
	enqueue         runloop.EnqueueFunc
}

func NewStatser(alert *v3.GlobalAlert) Statser {
	errorConditions, err := statserCommon.NewErrorConditions(statserCommon.MaxErrors)
	if err != nil {
		panic(err)
	}

	return &statser{
		alert:           alert,
		errorConditions: errorConditions,
	}
}

func (s *statser) Run(ctx context.Context) {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)

		run, enqueue := runloop.OnDemand()
		s.enqueue = enqueue

		go run(ctx, func(ctx context.Context, i interface{}) {
			s.updateStatus(i.(libcalicov3.GlobalAlertStatus))
		})
	})
}

func (s *statser) Close() {
	s.cancel()
}

func (s *statser) Status() libcalicov3.GlobalAlertStatus {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.status()
}

func (s *statser) status() libcalicov3.GlobalAlertStatus {
	res := libcalicov3.GlobalAlertStatus{
		ErrorConditions: s.errorConditions.Errors(),
	}

	return res
}

func (s *statser) SuccessfulSync() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.enqueue(s.status())
}

func (s *statser) SuccessfulSearch() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.enqueue(s.status())
}

func (s *statser) Error(t string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.errorConditions.Add(t, err)
	s.enqueue(s.status())
}

func (s *statser) ClearError(t string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.errorConditions.Clear(t)
	s.enqueue(s.status())
}

func (s *statser) updateStatus(status libcalicov3.GlobalAlertStatus) {
	// TODO CNX-9645
}
