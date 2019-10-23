// Copyright (c) 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"sync"
	"time"

	libcalicov3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"
	generatedV3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	idsElastic "github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	statserCommon "github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const (
	ElasticSyncFailed  = "ElasticSyncFailed"
	CompilationFailed  = "CompilationFailed"
	InstallationFailed = "InstallationFailed"
)

type Statser interface {
	Run(context.Context)
	Close()
	Status() libcalicov3.GlobalAlertStatus
	Error(string, error)
	ClearError(string)
}

type statser struct {
	name              string
	xPack             idsElastic.XPackWatcher
	globalAlertClient generatedV3.GlobalAlertInterface
	lastUpdate        *v1.Time
	active            bool
	healthy           bool
	executionState    string
	lastExecuted      *v1.Time
	lastEvent         *v1.Time
	errorConditions   *statserCommon.ErrorConditions
	lock              sync.RWMutex
	once              sync.Once
	cancel            context.CancelFunc
	enqueue           runloop.EnqueueFunc
}

func NewStatser(name string, xPack idsElastic.XPackWatcher, globalAlertClient generatedV3.GlobalAlertInterface) Statser {
	errorConditions, err := statserCommon.NewErrorConditions(statserCommon.MaxErrors)
	if err != nil {
		panic(err)
	}

	return &statser{
		name:              name,
		xPack:             xPack,
		globalAlertClient: globalAlertClient,
		errorConditions:   errorConditions,
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

		go runloop.RunLoop(ctx, func() {
			if err := s.pollStatus(ctx); err != nil {
				log.WithFields(log.Fields{
					"name": s.name,
				}).WithError(err).Error("status poll failed")
			} else {
				s.enqueue(s.Status())
			}
		}, time.Minute)
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
		LastUpdate:      s.lastUpdate,
		Active:          s.active,
		Healthy:         s.healthy,
		ExecutionState:  s.executionState,
		LastExecuted:    s.lastExecuted,
		LastEvent:       s.lastEvent,
		ErrorConditions: s.errorConditions.Errors(),
	}

	return res
}

func (s *statser) pollStatus(ctx context.Context) error {
	res, err := s.xPack.GetWatchStatus(ctx, s.name)
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.executionState = res.ExecutionState
	switch res.ExecutionState {
	case idsElastic.WatchExecutionStateExecuted, idsElastic.WatchExecutionStateExecutionNotNeeded,
		idsElastic.WatchExecutionStateThrottled, idsElastic.WatchExecutionStateAcknowledged:
		// TODO sort of...
		s.healthy = true
	default:
		s.healthy = false
	}
	if res.LastChecked != nil {
		s.lastExecuted = &v1.Time{*res.LastChecked}
	}
	if action, ok := res.Actions["index_events"]; ok { // TODO import cycle
		if action.LastSuccessfulExecution != nil {
			s.lastEvent = &v1.Time{action.LastSuccessfulExecution.Timestamp}
		}
	}
	if res.State != nil {
		s.active = res.State.Active
		s.lastUpdate = &v1.Time{res.State.Timestamp}
	}

	return nil
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
	log.WithField("name", s.name).Debug("Updating status")
	alert, err := s.globalAlertClient.Get(s.name, v1.GetOptions{})
	if err != nil {
		log.WithError(err).WithField("name", s.name).Error("Could not get global alert")
		return
	}
	alert.Status = status
	_, err = s.globalAlertClient.UpdateStatus(alert)
	if err != nil {
		log.WithError(err).WithField("name", s.name).Error("Could not update global alert status")
		return
	}
	log.WithField("name", s.name).Debug("Updated global alert status")
}
