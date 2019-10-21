// Copyright 2019 Tigera Inc. All rights reserved.

package watcher

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	apiv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/tigera/intrusion-detection/controller/pkg/alert/job"
	"github.com/tigera/intrusion-detection/controller/pkg/alert/statser"
	"github.com/tigera/intrusion-detection/controller/pkg/controller"
	"github.com/tigera/intrusion-detection/controller/pkg/elastic"
	"github.com/tigera/intrusion-detection/controller/pkg/health"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const DefaultResyncPeriod = 0

// Watcher accepts updates from alert and synchronizes them to the
// database
type Watcher interface {
	health.Pinger

	// Run starts the alert synchronization.
	Run(ctx context.Context)
	Close()
}

type watcher struct {
	globalAlertClient  apiv3.GlobalAlertInterface
	alertsController   controller.Controller
	httpClient         *http.Client
	xPack              elastic.XPackWatcher
	alertWatchers      map[string]*alertWatcher
	alertWatchersMutex sync.RWMutex
	cancel             context.CancelFunc

	// Unfortunately, cache.Controller callbacks can't accept
	// a context, so we need to store this on the watcher so we can pass it
	// to Pullers & Searchers we create.
	ctx context.Context

	once       sync.Once
	ping       chan struct{}
	watching   bool
	controller cache.Controller
	fifo       *cache.DeltaFIFO
	alerts     cache.Store
}

type alertWatcher struct {
	alert   *v3.GlobalAlert
	statser statser.Statser
	job     job.Job
}

func NewWatcher(
	globalAlertInterface apiv3.GlobalAlertInterface,
	alertsController controller.Controller,
	xPack elastic.XPackWatcher,
	httpClient *http.Client,
) Watcher {
	alertWatchers := map[string]*alertWatcher{}

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return globalAlertInterface.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return globalAlertInterface.Watch(options)
		},
	}
	w := &watcher{
		globalAlertClient: globalAlertInterface,
		alertsController:  alertsController,
		httpClient:        httpClient,
		xPack:             xPack,
		alertWatchers:     alertWatchers,
		ping:              make(chan struct{}),
	}

	w.fifo, w.alerts = util.NewPingableFifo()

	cfg := &cache.Config{
		Queue:            w.fifo,
		ListerWatcher:    lw,
		ObjectType:       &v3.GlobalAlert{},
		FullResyncPeriod: DefaultResyncPeriod,
		RetryOnError:     false,
		Process:          w.processQueue,
	}
	w.controller = cache.New(cfg)

	return w
}

func (s *watcher) Run(ctx context.Context) {
	s.once.Do(func() {

		s.ctx, s.cancel = context.WithCancel(ctx)

		go func() {
			// s.watching should only be true while this function is running.  Don't
			// bother with a lock because updates to booleans are always atomic.
			s.watching = true
			defer func() { s.watching = false }()
			s.controller.Run(s.ctx.Done())
		}()

		s.alertsController.Run(s.ctx)
		s.alertsController.StartReconciliation(s.ctx)
	})
}

func (s *watcher) processQueue(obj interface{}) error {
	// In general, this function only operates on local caches and FIFOs, so
	// will never return an error.  We panic on any errors since these indicate
	// programming bugs.

	// from oldest to newest
	for _, d := range obj.(cache.Deltas) {
		// Pings also come as cache updates
		_, ok := d.Object.(util.Ping)
		if ok {
			// Pong on a go routine so we don't block the main loop
			// if no pinger is listening.
			go s.pong()
			continue
		}
		switch d.Type {
		case cache.Sync, cache.Added, cache.Updated:
			old, exists, err := s.alerts.Get(d.Object)
			if err != nil {
				panic(err)
			}
			if exists {
				if err := s.alerts.Update(d.Object); err != nil {
					panic(err)
				}
				s.updateAlertWatcher(s.ctx, old.(*v3.GlobalAlert), d.Object.(*v3.GlobalAlert))
			} else {
				if err := s.alerts.Add(d.Object); err != nil {
					panic(err)
				}
				s.startAlertWatcher(s.ctx, d.Object.(*v3.GlobalAlert))
			}
		case cache.Deleted:
			if err := s.alerts.Delete(d.Object); err != nil {
				panic(err)
			}
			var name string
			switch f := d.Object.(type) {
			case *v3.GlobalAlert:
				name = f.Name
			case cache.DeletedFinalStateUnknown:
				name = f.Key
			default:
				panic(fmt.Sprintf("unknown FIFO delta type %v", d.Object))
			}
			_, exists := s.getAlertWatcher(name)
			if exists {
				s.stopAlertWatcher(s.ctx, name)
			}
		}
	}
	return nil
}

func (s *watcher) startAlertWatcher(ctx context.Context, f *v3.GlobalAlert) {
	log.WithFields(log.Fields{
		"name": f.Name,
	}).Debug("Starting alert")
	if _, ok := s.getAlertWatcher(f.Name); ok {
		panic(fmt.Sprintf("Alert %s already started", f.Name))
	}

	fCopy := f.DeepCopy()
	st := statser.NewStatser(fCopy)
	st.Run(ctx)

	aw := alertWatcher{
		alert:   f,
		job:     job.NewJob(f, st, s.alertsController),
		statser: st,
	}

	s.alertsController.NoGC(ctx, f.Name)

	aw.job.Run(ctx)
	s.setAlertWatcher(f.Name, &aw)
}

func (s *watcher) updateAlertWatcher(ctx context.Context, oldAlert, newAlert *v3.GlobalAlert) {
	log.WithFields(log.Fields{
		"name": newAlert.Name,
	}).Debug("Updating alert")
	aw, ok := s.getAlertWatcher(newAlert.Name)
	if !ok {
		panic(fmt.Sprintf("Alert %s not started", newAlert.Name))
	}

	aw.alert = newAlert.DeepCopy()
	aw.job.SetAlert(aw.alert)
}

func (s *watcher) stopAlertWatcher(ctx context.Context, name string) {
	log.WithFields(log.Fields{
		"name": name,
	}).Debug("Stopping alert")
	aw, ok := s.getAlertWatcher(name)
	if !ok {
		panic(fmt.Sprintf("Alert %s not started", name))
	}

	aw.statser.Close()
	aw.job.Close()

	s.alertsController.Delete(ctx, name)
	s.deleteAlertWatcher(name)
}

func (s *watcher) Close() {
	s.cancel()
}

func (s *watcher) getAlertWatcher(name string) (fw *alertWatcher, ok bool) {
	s.alertWatchersMutex.RLock()
	defer s.alertWatchersMutex.RUnlock()
	fw, ok = s.alertWatchers[name]
	return
}

func (s *watcher) setAlertWatcher(name string, fw *alertWatcher) {
	s.alertWatchersMutex.Lock()
	defer s.alertWatchersMutex.Unlock()
	s.alertWatchers[name] = fw
	return
}

func (s *watcher) deleteAlertWatcher(name string) {
	s.alertWatchersMutex.Lock()
	defer s.alertWatchersMutex.Unlock()
	delete(s.alertWatchers, name)
}

func (s *watcher) listAlertWatchers() []*alertWatcher {
	s.alertWatchersMutex.RLock()
	defer s.alertWatchersMutex.RUnlock()
	var out []*alertWatcher
	for _, fw := range s.alertWatchers {
		out = append(out, fw)
	}
	return out
}

// Ping is used to ensure the watcher's main loop is running and not blocked.
func (s *watcher) Ping(ctx context.Context) error {
	// Enqueue a ping
	err := s.fifo.Update(util.Ping{})
	if err != nil {
		// Local fifo & cache should never error.
		panic(err)
	}

	// Wait for the ping to be processed, or context to expire.
	select {
	case <-ctx.Done():
		return ctx.Err()

	// Since this channel is unbuffered, this will block if the main loop is not
	// running, or has itself blocked.
	case <-s.ping:
		return nil
	}
}

// pong is called from the main processing loop to reply to a ping.
func (s *watcher) pong() {
	// Nominally, a sync.Cond would work nicely here rather than a channel,
	// which would allow us to wake up all pingers at once. However, sync.Cond
	// doesn't allow timeouts, so we stick with channels and one pong() per ping.
	s.ping <- struct{}{}
}
