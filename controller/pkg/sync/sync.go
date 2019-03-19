// Copyright 2019 Tigera Inc. All rights reserved.

package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	retry "github.com/avast/retry-go"
	v33 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v32 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
)

const (
	retryAttempts = 3
	retryDelay    = 5 * time.Second
)

type Syncer interface {
	Run(context.Context, <-chan db.IPSetSpec, puller.SyncFailFunction, statser.Statser)
	SetFeed(*v3.GlobalThreatFeed)
	Close()
}

type syncer struct {
	feed                   *v3.GlobalThreatFeed
	ipSet                  db.IPSet
	globalNetworkSetClient v32.GlobalNetworkSetInterface
	cancel                 context.CancelFunc
	once                   sync.Once
	lock                   sync.RWMutex
}

func NewSyncer(feed *v3.GlobalThreatFeed, ipSet db.IPSet, globalNetworkSetClient v32.GlobalNetworkSetInterface) Syncer {
	return &syncer{feed: feed.DeepCopy(), ipSet: ipSet, globalNetworkSetClient: globalNetworkSetClient}
}

func (s *syncer) Run(ctx context.Context, c <-chan db.IPSetSpec, failFunc puller.SyncFailFunction, st statser.Statser) {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)
		go func() {
			defer s.cancel()
			s.runSyncGlobalNetworkSet(ctx, st)

			_ = runloop.RunLoopRecvChannel(ctx, func(x interface{}) {
				_ = s.sync(ctx, x.(db.IPSetSpec), failFunc, st, retryAttempts, retryDelay)
			}, c)
		}()
	})
}

func (s *syncer) sync(ctx context.Context, set db.IPSetSpec, failFunction puller.SyncFailFunction, st statser.Statser, attempts uint, delay time.Duration) error {
	s.lock.RLock()
	name := s.feed.Name
	gnss := s.feed.Spec.GlobalNetworkSet
	s.lock.RUnlock()

	err := retry.Do(func() error {
		return s.ipSet.PutIPSet(ctx, name, set)
	}, retry.Attempts(attempts), retry.Delay(delay))
	if err != nil {
		failFunction()
		log.WithError(err).WithField("name", name).Error("could not put FeedPuller set from feed")
		st.Error(statser.ElasticSyncFailed, err)
		return err
	}
	log.WithField("name", name).Info("Stored IPSet to database")
	st.ClearError(statser.ElasticSyncFailed)

	if gnss != nil {
		err = s.syncGlobalNetworkSet(ctx, name, set, gnss, st)
		if err != nil {
			failFunction()
			log.WithError(err).WithField("name", name).Error("could not sync FeedPuller set to GlobalNetworkSet")
			st.Error(statser.GlobalNetworkSetSyncFailed, err)
			return err
		}
	}
	st.ClearError(statser.GlobalNetworkSetSyncFailed)
	st.SuccessfulSync()

	return nil
}

func (s *syncer) runSyncGlobalNetworkSet(ctx context.Context, st statser.Statser) {
	s.lock.RLock()
	gnss := s.feed.Spec.GlobalNetworkSet
	name := s.feed.Name
	s.lock.RUnlock()

	if gnss == nil {
		st.ClearError(statser.GlobalNetworkSetSyncFailed)
		return
	}

	ipSet, err := s.ipSet.GetIPSet(ctx, name)
	if err != nil {
		log.WithError(err).WithField("name", name).Error("could not get IPSet from DB")
		st.Error(statser.GlobalNetworkSetSyncFailed, err)
		return
	}
	err = s.syncGlobalNetworkSet(ctx, name, ipSet, gnss, st)
	if err != nil {
		log.WithError(err).WithField("name", name).Error("could not sync IPSet to GlobalNetworkSet")
		st.Error(statser.GlobalNetworkSetSyncFailed, err)
		return
	}

	st.ClearError(statser.GlobalNetworkSetSyncFailed)
}

func (s *syncer) syncGlobalNetworkSet(ctx context.Context, name string, set db.IPSetSpec, gnss *v33.GlobalNetworkSetSync, st statser.Statser) error {
	// prefix name with "threatfeed."
	name = fmt.Sprintf("threatfeed.%s", name)

	// This does a Create to create the feed. k8s returns an error if it already exists
	// so we Get the old record, update the set and labels, and Update it. This preserves
	// metadata and any changes that may have been made to other parts of the record.
	gns := &v3.GlobalNetworkSet{
		ObjectMeta: v1.ObjectMeta{
			Name:   name,
			Labels: gnss.Labels,
			Annotations: map[string]string{
				"tigera.io/creator": "intrusion-detection-controller",
			},
		},
		Spec: v33.GlobalNetworkSetSpec{
			Nets: set,
		},
	}
	_, err := s.globalNetworkSetClient.Create(gns)
	if err != nil {
		if kerrors.IsAlreadyExists(err) {
			gns, err := s.globalNetworkSetClient.Get(name, v1.GetOptions{})
			if err != nil {
				if !kerrors.IsNotFound(err) {
					log.WithError(err).WithField("name", name).Error("Could not get GlobalNetworkSet")
					return err
				}

			}

			gns.Labels = gnss.Labels
			gns.Spec.Nets = set

			_, err = s.globalNetworkSetClient.Update(gns)
			if err != nil {
				log.WithError(err).WithField("name", name).Error("Could not update GlobalNetworkSet")
				return err
			}
			log.WithField("name", name).Info("Successfully updated GlobalNetworkSet")

			return nil
		}
		log.WithError(err).WithField("name", name).Error("Could not create GlobalNetworkSet")
		return err
	}
	log.WithField("name", name).Info("Successfully created GlobalNetworkSet")
	return nil
}

func (s *syncer) SetFeed(f *v3.GlobalThreatFeed) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.feed = f.DeepCopy()
}

func (s *syncer) Close() {
	s.cancel()
}
