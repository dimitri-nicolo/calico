// Package data is responsible for aggregating Felix updates, and ensuring
// that the consumers of the data receive it only when it is safe. This
// is done using *roughly* the observer pattern.
package data

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/egress-gateway/proto"
	protoutil "github.com/tigera/egress-gateway/util/proto"
)

// RouteObserver allows a module to be notified of updates regarding routes
type RouteObserver interface {
	// NotifyResync notifies an observer of a full datastore refresh
	NotifyResync(RouteStore)
}

// RouteStore encapsulates the inner datastore to protect from arbitraty reads/writes
type RouteStore interface {
	Routes(readFn func(map[string]*proto.RouteUpdate))
	Subscribe(RouteObserver)     // allows Observers to subscribe to store updates
	SyncForever(context.Context) // begin a sync loop (blocks until context.Done)
}

// routeStore stores all information needed to program the Egress Gateway's return routes to workloads
type routeStore struct {
	// will be notified of updates
	observers []RouteObserver

	// RWMutex should be used when reading/writing store data
	sync.RWMutex

	routeUpdatesByWorkloadCIDR map[string]*proto.RouteUpdate

	inSync bool

	// getUpdatesPipeline is a means of getting a new updates pipeline if one is closed for any reason
	getUpdatesPipeline func() <-chan *proto.ToDataplane
}

// NewRouteStore instantiates a new store for route updates
func NewRouteStore(getUpdatesPipeline func() <-chan *proto.ToDataplane) RouteStore {
	return &routeStore{
		observers:                  make([]RouteObserver, 0),
		RWMutex:                    sync.RWMutex{},
		routeUpdatesByWorkloadCIDR: make(map[string]*proto.RouteUpdate),
		inSync:                     false,
		getUpdatesPipeline:         getUpdatesPipeline,
	}
}

// Routes wraps the read function to allow for thread-safe access to RouteUpdates.
func (s *routeStore) Routes(readFn func(map[string]*proto.RouteUpdate)) {
	s.read(func(s *routeStore) {
		readFn(s.routeUpdatesByWorkloadCIDR)
	})
}

// Subscribe allows datastore consumers to subscribe to store updates
func (s *routeStore) Subscribe(o RouteObserver) {
	s.observers = append(s.observers, o)
}

// Sync aggregates payloads from the sync client into safe, condensed 'notifications' for store observers to program
func (s *routeStore) SyncForever(ctx context.Context) {
	s.inSync = false
	updates := s.getUpdatesPipeline()

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				log.Debug("updates channel closed by upstream, fetching new channel...")
				// if the updates pipeline closes, get a fresh pipline and start resyncing from scratch
				updates = s.getUpdatesPipeline()
				s.inSync = false
				s.clear()
			} else {
				log.WithField("update", update).Debug("parsing new update from upstream...")
				// otherwise begin parsing pipeline updates
				switch payload := update.Payload.(type) {

				case *proto.ToDataplane_RouteUpdate:
					ru := payload.RouteUpdate
					if ru.Type == proto.RouteType_REMOTE_WORKLOAD ||
						ru.Type == proto.RouteType_LOCAL_WORKLOAD ||
						protoutil.IsWireguardTunnel(ru) {

						log.Debugf("received RouteUpdate for a workload: %v", update.Payload.(*proto.ToDataplane_RouteUpdate))
						wCIDR := ru.Dst
						s.write(func(rs *routeStore) {
							rs.routeUpdatesByWorkloadCIDR[wCIDR] = ru
						})
						s.maybeNotifyResync()
					}

				case *proto.ToDataplane_RouteRemove:
					log.Debugf("RouteRemove Received: %v", update.Payload.(*proto.ToDataplane_RouteRemove))

					rm := payload.RouteRemove
					wCIDR := rm.Dst
					s.write(func(rs *routeStore) {
						delete(rs.routeUpdatesByWorkloadCIDR, wCIDR)
					})
					s.maybeNotifyResync()

				case *proto.ToDataplane_InSync:
					// After receiving an `inSync`, all future updates over this channel will immediately notify observers
					s.inSync = true
					s.maybeNotifyResync()
				default:
					log.Debugf("Unexpected update received: %v", update)
				}
			}
		}
	}
}

// write allows for thread-safe writes to the store via a write-callback
func (s *routeStore) write(writeFn func(*routeStore)) {
	log.Debug("Acquiring write lock for egress-gateway store")
	s.RWMutex.Lock()
	defer func() {
		s.RWMutex.Unlock()
		log.Debug("Released write lock for egress-gateway store")
	}()
	log.Debug("Acquired write lock for egress-gateway store")
	writeFn(s)
}

// read allows for thread-safe reads from the store via a read-callback
func (s *routeStore) read(readFn func(*routeStore)) {
	log.Debug("Acquiring read lock for the datastore")
	s.RWMutex.RLock()
	defer func() {
		s.RWMutex.RUnlock()
		log.Debug("Released read lock for the datastore")
	}()
	log.Debug("Acquired read lock for the datastore")
	readFn(s)
}

// clear drops all data in the routeStore
func (s *routeStore) clear() {
	s.write(func(rs *routeStore) {
		rs.routeUpdatesByWorkloadCIDR = make(map[string]*proto.RouteUpdate)
	})
}

// notify datastore Observers of a full resync
func (s *routeStore) maybeNotifyResync() {
	if s.inSync {
		for _, o := range s.observers {
			// use a goroutine so as not to be blocked by potentially long downstream operations
			go o.NotifyResync(s)
		}
	}
}
