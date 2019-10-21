// Copyright 2019 Tigera Inc. All rights reserved.

package statser

import (
	"context"
	"sync"
	"time"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"
	v32 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	statserCommon "github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const (
	ElasticSyncFailed          = "ElasticSyncFailed"
	GlobalNetworkSetSyncFailed = "GlobalNetworkSetSyncFailed"
	GarbageCollectionFailed    = "GarbageCollectionFailed"
	PullFailed                 = "PullFailed"
	SearchFailed               = "SearchFailed"
)

type Statser interface {
	Run(context.Context)
	Close()
	Status() v3.GlobalThreatFeedStatus
	SuccessfulSync()
	SuccessfulSearch()
	Error(string, error)
	ClearError(string)
}

func NewStatser(name string, globalThreatFeedClient v32.GlobalThreatFeedInterface) Statser {
	l, err := statserCommon.NewErrorConditions(statserCommon.MaxErrors)
	if err != nil {
		panic(err)
	}

	return &statser{
		name:                   name,
		globalThreatFeedClient: globalThreatFeedClient,
		errorConditions:        l,
	}
}

type statser struct {
	name                   string
	globalThreatFeedClient v32.GlobalThreatFeedInterface
	lastSuccessfulSync     time.Time
	lastSuccessfulSearch   time.Time
	errorConditions        *statserCommon.ErrorConditions
	lock                   sync.RWMutex
	once                   sync.Once
	cancel                 context.CancelFunc
	enqueue                runloop.EnqueueFunc
}

func (s *statser) Run(ctx context.Context) {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)

		run, enqueue := runloop.OnDemand()
		s.enqueue = enqueue

		go run(ctx, func(ctx context.Context, i interface{}) {
			s.updateStatus(i.(v3.GlobalThreatFeedStatus))
		})
	})
}

func (s *statser) Close() {
	s.cancel()
}

func (s *statser) Status() v3.GlobalThreatFeedStatus {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.status()
}

func (s *statser) status() v3.GlobalThreatFeedStatus {
	res := v3.GlobalThreatFeedStatus{
		LastSuccessfulSync:   v1.Time{Time: s.lastSuccessfulSync},
		LastSuccessfulSearch: v1.Time{Time: s.lastSuccessfulSearch},
		ErrorConditions:      s.errorConditions.Errors(),
	}

	return res
}

func (s *statser) SuccessfulSync() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lastSuccessfulSync = time.Now()
	s.enqueue(s.status())
}

func (s *statser) SuccessfulSearch() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lastSuccessfulSearch = time.Now()
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

func (s *statser) updateStatus(status v3.GlobalThreatFeedStatus) {
	log.WithField("name", s.name).Debug("Updating status")
	gtf, err := s.globalThreatFeedClient.Get(s.name, v1.GetOptions{})
	if err != nil {
		log.WithError(err).WithField("name", s.name).Error("Could not get global threat feed")
		return
	}
	gtf.Status = status
	_, err = s.globalThreatFeedClient.UpdateStatus(gtf)
	if err != nil {
		log.WithError(err).WithField("name", s.name).Error("Could not update global threat feed status")
		return
	}
	log.WithField("name", s.name).Debug("Updated global threat feed status")
}
