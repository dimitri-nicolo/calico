// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"fmt"
	"net/http"
	goSync "sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v32 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/gc"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
	"github.com/tigera/intrusion-detection/controller/pkg/puller"
	"github.com/tigera/intrusion-detection/controller/pkg/searcher"
	"github.com/tigera/intrusion-detection/controller/pkg/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/sync/globalnetworksets"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const (
	RetryDelay = 10 * time.Second
)

// Watcher accepts updates from threat pullers and synchronizes them to the
// database
type Watcher interface {
	health.Pinger
	health.Readier

	// Run starts the feed synchronization.
	Run(ctx context.Context)
	Close()
}

type watcher struct {
	configMapClient        v1.ConfigMapInterface
	secretsClient          v1.SecretInterface
	globalThreatFeedClient v32.GlobalThreatFeedInterface
	globalNetworkSetClient v32.GlobalNetworkSetInterface
	gnsController          globalnetworksets.Controller
	httpClient             *http.Client
	ipSet                  db.IPSet
	suspiciousIP           db.SuspiciousIP
	events                 db.Events
	feeds                  map[string]*feedWatcher
	feedMutex              goSync.RWMutex
	cancel                 context.CancelFunc
	once                   goSync.Once
	ping                   chan struct{}
	watching               bool
}

type feedWatcher struct {
	feed             *v3.GlobalThreatFeed
	puller           puller.Puller
	garbageCollector gc.GarbageCollector
	searcher         searcher.FlowSearcher
	statser          statser.Statser
}

func NewWatcher(
	configMapClient v1.ConfigMapInterface,
	secretsClient v1.SecretInterface,
	globalThreatFeedInterface v32.GlobalThreatFeedInterface,
	globalNetworkSetController globalnetworksets.Controller,
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
		gnsController:          globalNetworkSetController,
		httpClient:             httpClient,
		ipSet:                  ipSet,
		suspiciousIP:           suspiciousIP,
		events:                 events,
		feeds:                  feeds,
		ping:                   make(chan struct{}),
	}
}

func (s *watcher) Run(ctx context.Context) {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)

		go func() {
			defer s.cancel()
			var w watch.Interface

			// Try forever to establish a Watch
			for {
				var err error
				w, err = s.globalThreatFeedClient.Watch(metav1.ListOptions{
					Watch: true,
				})
				if err == nil {
					log.Info("Watch started")
					defer w.Stop()
					// Set watching to true only after the Watch has returned without an
					// error, and only until this Run loop returns.  No need to bother
					// with a lock since bools are atomic.
					s.watching = true
					defer func() { s.watching = false }()
					break
				}
				log.WithError(err).Error("Failed to start Watch")
				select {
				case <-ctx.Done():
					return
				case <-time.After(RetryDelay):
					// retry
				}
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-s.ping:
					// Used to test liveness of the watcher.
					continue
				case event, ok := <-w.ResultChan():
					if !ok {
						return
					}

					s.handleEvent(ctx, event)
				}
			}
		}()
	})
}

func (s *watcher) handleEvent(ctx context.Context, event watch.Event) {
	switch event.Type {
	case watch.Added, watch.Modified:
		globalThreatFeed, ok := event.Object.(*v3.GlobalThreatFeed)
		if !ok {
			log.WithField("event", event).Error("Received event of unexpected type")
			return
		}

		if _, ok := s.getFeedWatcher(globalThreatFeed.Name); !ok {
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
			return
		}

		if _, ok := s.getFeedWatcher(globalThreatFeed.Name); ok {
			s.stopFeed(globalThreatFeed.Name)
			log.WithField("feed", globalThreatFeed.Name).Info("Feed stopped")
		} else {
			log.WithField("feed", globalThreatFeed.Name).Info("Ignored deletion of non-running feed")
		}
	case watch.Error:
		switch event.Object.(type) {
		case *metav1.Status:
			status := event.Object.(*metav1.Status)
			log.WithField("status", status).Error("Kubernetes API error")

		default:
			log.WithField("event", event).Error("Received kubernetes API error of unexpected type")
		}
	default:
		log.WithField("event", event).Error("Received event with unexpected type")
	}
}

func (s *watcher) startFeed(ctx context.Context, f v3.GlobalThreatFeed) {
	if _, ok := s.getFeedWatcher(f.Name); ok {
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

	s.setFeedWatcher(f.Name, &fw)

	if fCopy.Spec.Pull != nil && fCopy.Spec.Pull.HTTP != nil {
		fw.puller = puller.NewHTTPPuller(fCopy, s.ipSet, s.configMapClient, s.secretsClient, s.httpClient, s.gnsController)
		failFunc := fw.puller.Run(ctx, fw.statser)
		s.gnsController.RegisterFailFunc(fCopy.Name, failFunc)
	} else {
		fw.puller = nil
	}

	if fCopy.Spec.GlobalNetworkSet != nil {
		s.gnsController.NoGC(util.NewGlobalNetworkSet(fCopy.Name))
	}

	fw.garbageCollector.Run(ctx, fw.statser)
	fw.searcher.Run(ctx, fw.statser)
}

func (s *watcher) updateFeed(ctx context.Context, f v3.GlobalThreatFeed) {
	fw, ok := s.getFeedWatcher(f.Name)
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
		}
	} else {
		if fw.puller != nil {
			fw.puller.Close()
		}
		fw.puller = nil
	}

	gns := util.NewGlobalNetworkSet(fw.feed.Name)
	if fw.feed.Spec.GlobalNetworkSet != nil {
		s.gnsController.NoGC(gns)
	} else {
		s.gnsController.Delete(gns)
	}

	fw.searcher.SetFeed(fw.feed)
	fw.garbageCollector.SetFeed(fw.feed)
}

func (s *watcher) restartPuller(ctx context.Context, f v3.GlobalThreatFeed) {
	name := f.Name

	fw, ok := s.getFeedWatcher(name)
	if !ok {
		panic(fmt.Sprintf("feed %s not running", name))
	}

	fw.feed = f.DeepCopy()
	if fw.puller != nil {
		fw.puller.Close()
	}

	if fw.feed.Spec.Pull != nil && fw.feed.Spec.Pull.HTTP != nil {
		fw.puller = puller.NewHTTPPuller(fw.feed, s.ipSet, s.configMapClient, s.secretsClient, s.httpClient, s.gnsController)
		failFunc := fw.puller.Run(ctx, fw.statser)
		s.gnsController.RegisterFailFunc(name, failFunc)
	} else {
		fw.puller = nil
	}
}

func (s *watcher) stopFeed(name string) {
	fw, ok := s.getFeedWatcher(name)
	if !ok {
		panic(fmt.Sprintf("feed %s not running", name))
	}

	log.WithField("feed", name).Info("Stopping feed")

	if fw.puller != nil {
		fw.puller.Close()
	}
	gns := util.NewGlobalNetworkSet(name)
	s.gnsController.Delete(gns)

	fw.garbageCollector.Close()
	fw.searcher.Close()
	s.deleteFeedWatcher(name)
}

func (s *watcher) Close() {
	s.cancel()
}

func (s *watcher) getFeedWatcher(name string) (fw *feedWatcher, ok bool) {
	s.feedMutex.RLock()
	defer s.feedMutex.RUnlock()
	fw, ok = s.feeds[name]
	return
}

func (s *watcher) setFeedWatcher(name string, fw *feedWatcher) {
	s.feedMutex.Lock()
	defer s.feedMutex.Unlock()
	s.feeds[name] = fw
	return
}

func (s *watcher) deleteFeedWatcher(name string) {
	s.feedMutex.Lock()
	defer s.feedMutex.Unlock()
	delete(s.feeds, name)
}

func (s *watcher) listFeedWatchers() []*feedWatcher {
	s.feedMutex.RLock()
	defer s.feedMutex.RUnlock()
	var out []*feedWatcher
	for _, fw := range s.feeds {
		out = append(out, fw)
	}
	return out
}

// Ping is used to ensure the watcher's main loop is running and not blocked.
func (s *watcher) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	// Since this channel is unbuffered, this will block if the main loop is not
	// running, or has itself blocked.
	case s.ping <- struct{}{}:
		return nil
	}
}

// Ready determines whether we are watching GlobalThreatFeeds and they are all
// functioning correctly.
func (s *watcher) Ready() bool {
	if !s.watching {
		return false
	}

	// Loop over all the active feeds and return false if any have errors.
	for _, fw := range s.listFeedWatchers() {
		status := fw.statser.Status()
		if len(status.ErrorConditions) > 0 {
			return false
		}
	}
	return true
}
