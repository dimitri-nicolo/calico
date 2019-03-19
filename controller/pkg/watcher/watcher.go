// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"fmt"
	"net/http"
	goSync "sync"
	"time"

	"github.com/tigera/intrusion-detection/controller/pkg/util"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v32 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/gc"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/searcher"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/sync"
)

// Watcher accepts updates from threat pullers and synchronizes them to the
// database
type Watcher interface {
	// Run starts the feed synchronization.
	Run(ctx context.Context) error
	Close()
}

type watcher struct {
	configMapClient        v1.ConfigMapInterface
	secretsClient          v1.SecretInterface
	globalThreatFeedClient v32.GlobalThreatFeedInterface
	globalNetworkSetClient v32.GlobalNetworkSetInterface
	httpClient             *http.Client
	ipSet                  db.IPSet
	suspiciousIP           db.SuspiciousIP
	events                 db.Events
	feeds                  map[string]*feedWatcher
	cancel                 context.CancelFunc
	once                   goSync.Once
	err                    error
}

type feedWatcher struct {
	feed             *v3.GlobalThreatFeed
	puller           puller.Puller
	syncer           sync.Syncer
	garbageCollector gc.GarbageCollector
	searcher         searcher.FlowSearcher
	statser          statser.Statser
}

func NewWatcher(
	configMapClient v1.ConfigMapInterface,
	secretsClient v1.SecretInterface,
	globalThreatFeedInterface v32.GlobalThreatFeedInterface,
	globalNetworkSetInterface v32.GlobalNetworkSetInterface,
	httpClient *http.Client,
	ipSet db.IPSet,
	suspiciousIP db.SuspiciousIP,
	events db.Events,
) Watcher {
	feeds := map[string]*feedWatcher{}

	return &watcher{
		configMapClient:        configMapClient,
		secretsClient:          secretsClient,
		globalThreatFeedClient: globalThreatFeedInterface,
		globalNetworkSetClient: globalNetworkSetInterface,
		httpClient:             httpClient,
		ipSet:                  ipSet,
		suspiciousIP:           suspiciousIP,
		events:                 events,
		feeds:                  feeds,
	}
}

func (s *watcher) Run(ctx context.Context) error {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)
		defer s.cancel()

		w, err := s.globalThreatFeedClient.Watch(metav1.ListOptions{
			Watch: true,
		})
		if err != nil {
			s.err = err
			return
		}
		defer w.Stop()

		for {
			select {
			case <-ctx.Done():
				s.err = ctx.Err()
				return
			case event, ok := <-w.ResultChan():
				if !ok {
					return
				}

				switch event.Type {
				case watch.Added, watch.Modified:
					globalThreatFeed, ok := event.Object.(*v3.GlobalThreatFeed)
					if !ok {
						log.WithField("event", event).Error("Received event of unexpected type")
						continue
					}

					if _, ok := s.feeds[globalThreatFeed.Name]; !ok {
						s.startFeed(ctx, *globalThreatFeed)
						log.WithField("feed", globalThreatFeed.Name).Info("Feed started")
					} else {
						s.updateFeed(ctx, *globalThreatFeed)
						log.WithField("feed", globalThreatFeed.Name).Info("Feed updated")

					}
				case watch.Deleted:
					globalThreatFeed, ok := event.Object.(*v3.GlobalThreatFeed)
					if !ok {
						log.WithField("event", event).Error("Received event of unexpected type")
						continue
					}

					s.stopFeed(globalThreatFeed.Name)
					log.WithField("feed", globalThreatFeed.Name).Info("Feed stopped")
				case watch.Error:
					switch event.Object.(type) {
					case *metav1.Status:
						status := event.Object.(*metav1.Status)
						log.WithField("status", status).Error("Kubernetes API error")

					default:
						log.WithField("event", event).Error("Received kubernetes API error of unexpected type")
					}
				}
			}
		}
	})
	return s.err
}

func (s *watcher) startFeed(ctx context.Context, f v3.GlobalThreatFeed) {
	if _, ok := s.feeds[f.Name]; ok {
		panic(fmt.Sprintf("Feed %s already started", f.Name))
	}

	fCopy := f.DeepCopy()
	st := statser.NewStatser()
	fw := feedWatcher{
		feed:             fCopy,
		garbageCollector: gc.NewGarbageCollector(fCopy, time.Hour),
		searcher:         searcher.NewFlowSearcher(fCopy, time.Minute, s.suspiciousIP, s.events),
		statser:          st,
	}

	s.feeds[f.Name] = &fw

	if fCopy.Spec.Pull != nil && fCopy.Spec.Pull.HTTP != nil {
		fw.puller = puller.NewHTTPPuller(fCopy, s.ipSet, s.configMapClient, s.secretsClient, s.httpClient)
		c, failFunc := fw.puller.Run(ctx, fw.statser)
		fw.syncer = sync.NewSyncer(fCopy, s.ipSet, s.globalNetworkSetClient)
		fw.syncer.Run(ctx, c, failFunc, fw.statser)
	} else {
		// TODO log
		fw.puller = nil
		fw.syncer = nil
	}

	fw.garbageCollector.Run(ctx, fw.statser)
	fw.searcher.Run(ctx, fw.statser)
}

func (s *watcher) updateFeed(ctx context.Context, f v3.GlobalThreatFeed) {
	fw, ok := s.feeds[f.Name]
	if !ok {
		panic(fmt.Sprintf("Feed %s not started", f.Name))
	}

	oldFeed := fw.feed
	fw.feed = f.DeepCopy()
	if fw.feed.Spec.Pull != nil && fw.feed.Spec.Pull.HTTP != nil {
		if util.FeedNeedsRestart(oldFeed, fw.feed) {
			s.restartPuller(ctx, f)
		} else {
			fw.puller.SetFeed(fw.feed)
			fw.syncer.SetFeed(fw.feed)
		}
	} else {
		if fw.puller != nil {
			fw.puller.Close()
		}
		fw.puller = nil
		if fw.syncer != nil {
			fw.syncer.Close()
		}
		fw.syncer = nil
	}

	fw.searcher.SetFeed(fw.feed)
	fw.garbageCollector.SetFeed(fw.feed)
}

func (s *watcher) restartPuller(ctx context.Context, f v3.GlobalThreatFeed) {
	name := f.Name

	fw, ok := s.feeds[name]
	if !ok {
		panic(fmt.Sprintf("feed %s not running", name))
	}

	fw.feed = f.DeepCopy()
	if fw.puller != nil {
		fw.puller.Close()
	}
	if fw.syncer != nil {
		fw.syncer.Close()
	}

	if fw.feed.Spec.Pull != nil && fw.feed.Spec.Pull.HTTP != nil {
		fw.puller = puller.NewHTTPPuller(fw.feed, s.ipSet, s.configMapClient, s.secretsClient, s.httpClient)
		c, failFunc := fw.puller.Run(ctx, fw.statser)
		fw.syncer = sync.NewSyncer(fw.feed, s.ipSet, s.globalNetworkSetClient)
		fw.syncer.Run(ctx, c, failFunc, fw.statser)
	} else {
		// TODO log
		fw.puller = nil
		fw.syncer = nil
	}
}

func (s *watcher) stopFeed(name string) {
	fw, ok := s.feeds[name]
	if !ok {
		panic(fmt.Sprintf("feed %s not running", name))
	}

	log.WithField("feed", name).Info("Stopping feed")

	if fw.puller != nil {
		fw.puller.Close()
	}
	if fw.syncer != nil {
		fw.syncer.Close()
	}
	fw.garbageCollector.Close()
	fw.searcher.Close()
	delete(s.feeds, name)
}

func (s *watcher) Close() {
	s.cancel()
}
