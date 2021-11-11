// Package data is responsible for aggregating Felix updates, and ensuring
// that the consumers of the data receive it only when it is safe. This
// is done using *roughly* the observer pattern.
package data

import (
	"context"
	"net"
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
	// Workloads wraps the read function to allow for thread-safe access to workload RouteUpdates.
	Workloads(readFn func(map[string]*proto.RouteUpdate))
	// WorkloadsByNodeName wraps the Workloads function to allow for thread-safe access to workload RouteUpdates, sorted by node.
	WorkloadsByNodeName(readFn func(map[string][]proto.RouteUpdate))
	// Tunnels wraps the read function to allow for thread-safe access to host tunnel RouteUpdates.
	Tunnels(readFn func(map[string]*proto.RouteUpdate))
	// TunnelsByNodeName wraps the Tunnels function to allow for thread-safe access to host tunnel RouteUpdates sorted by node.
	TunnelsByNodeName(readFn func(map[string][]proto.RouteUpdate))
	// GatewayWorkload fetches the RouteUpdate describing the instance of the egress gateway workload itself (thread-safe)
	GatewayWorkload(readFn func(*proto.RouteUpdate))
	// Subscribe allows Observers to subscribe to store updates
	Subscribe(RouteObserver)
	// SyncForever begins a sync loop (blocks until context.Done)
	SyncForever(context.Context)
}

// routeStore stores all information needed to program the Egress Gateway's return routes to workloads
type routeStore struct {
	// will be notified of updates
	observers []RouteObserver

	// RWMutex should be used when reading/writing store data
	sync.RWMutex

	// RouteUpdates describing workloads on other nodes (local workloads do not require tunneling and should use default routing)
	remoteWorkloadUpdatesByDst map[string]*proto.RouteUpdate

	// RouteUpdates describing host-ns tunnel devices per-node. It's important we recongise these IP's as they may
	// be used instead of a node's default IP by outbound egress packets.
	tunnelUpdatesByDst map[string]*proto.RouteUpdate

	// the latest RouteUpdate describing this gateway - can contain information about what encap is being used for the gateway's ippool
	latestGatewayUpdate *proto.RouteUpdate

	// this gateway's own IP
	gatewayIP net.IP

	// observers of this store will not be notified of updates until the store is inSync
	inSync bool

	// getUpdatesPipeline is a means of getting a new updates pipeline if one is closed for any reason
	getUpdatesPipeline func() <-chan *proto.ToDataplane
}

// NewRouteStore instantiates a new store for route updates
func NewRouteStore(getUpdatesPipeline func() <-chan *proto.ToDataplane, egressPodIP net.IP) RouteStore {
	return &routeStore{
		observers:                  make([]RouteObserver, 0),
		RWMutex:                    sync.RWMutex{},
		remoteWorkloadUpdatesByDst: make(map[string]*proto.RouteUpdate),
		tunnelUpdatesByDst:         make(map[string]*proto.RouteUpdate),
		gatewayIP:                  egressPodIP,
		inSync:                     false,
		getUpdatesPipeline:         getUpdatesPipeline,
	}
}

// Workloads wraps the read function to allow for thread-safe access to workload RouteUpdates.
func (s *routeStore) Workloads(readFn func(map[string]*proto.RouteUpdate)) {
	s.read(func(s *routeStore) {
		readFn(s.remoteWorkloadUpdatesByDst)
	})
}

// WorkloadsByNodeName wraps the Workloads function to allow for thread-safe access to workload RouteUpdates, sorted by node.
func (s *routeStore) WorkloadsByNodeName(readFn func(map[string][]proto.RouteUpdate)) {
	s.Workloads(func(workloads map[string]*proto.RouteUpdate) {
		workloadsByNodeName := make(map[string][]proto.RouteUpdate)

		for _, workload := range workloads {
			nodeName := workload.DstNodeName
			if _, ok := workloadsByNodeName[nodeName]; !ok {
				workloadsByNodeName[nodeName] = make([]proto.RouteUpdate, 0)
			}
			workloadsByNodeName[nodeName] = append(workloadsByNodeName[nodeName], *workload)
		}
		log.Debugf("constructed workload map sorted by node name: %+v", workloadsByNodeName)
		readFn(workloadsByNodeName)
	})
}

// Tunnels wraps the read function to allow for thread-safe access to host tunnel RouteUpdates.
func (s *routeStore) Tunnels(readFn func(map[string]*proto.RouteUpdate)) {
	s.read(func(s *routeStore) {
		readFn(s.tunnelUpdatesByDst)
	})
}

// TunnelsForHost provides thread-safe read access to host-ns tunnel RouteUpdates for the given node
func (s *routeStore) TunnelsByNodeName(readFn func(map[string][]proto.RouteUpdate)) {
	s.Tunnels(func(tunnels map[string]*proto.RouteUpdate) {
		tunnelsByNodeName := make(map[string][]proto.RouteUpdate)

		for _, tunnel := range tunnels {
			nodeName := tunnel.DstNodeName
			if _, ok := tunnelsByNodeName[nodeName]; !ok {
				tunnelsByNodeName[nodeName] = make([]proto.RouteUpdate, 0)
			}
			tunnelsByNodeName[nodeName] = append(tunnelsByNodeName[nodeName], *tunnel)
		}
		log.Debugf("constructed tunnels map sorted by node name: %+v", tunnelsByNodeName)
		readFn(tunnelsByNodeName)
	})
}

// GatewayWorkload fetches the RouteUpdate describing the instance of the egress gateway workload itself (thread-safe)
func (s *routeStore) GatewayWorkload(readFn func(*proto.RouteUpdate)) {
	s.read(func(s *routeStore) {
		readFn(s.latestGatewayUpdate)
	})
}

// Subscribe allows datastore consumers to subscribe to store updates
func (s *routeStore) Subscribe(o RouteObserver) {
	s.observers = append(s.observers, o)
}

// SyncForever aggregates payloads from the sync client into safe, condensed 'notifications' for store observers to program
func (s *routeStore) SyncForever(ctx context.Context) {
	s.inSync = false
	updates := s.getUpdatesPipeline()

	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				// if the updates pipeline closes, get a fresh pipline and start resyncing from scratch
				log.Debug("updates channel closed by upstream, fetching new channel...")
				updates = s.getUpdatesPipeline()
				s.inSync = false
				s.clear()
			} else {
				// begin parsing pipeline updates
				log.WithField("update", update).Debug("parsing new update from upstream...")

				switch payload := update.Payload.(type) {
				case *proto.ToDataplane_RouteUpdate:
					ru := payload.RouteUpdate
					if ru.Type == proto.RouteType_REMOTE_WORKLOAD {
						log.Debugf("received RouteUpdate for a remote workload: %+v", ru)
						s.write(func(rs *routeStore) {
							rs.remoteWorkloadUpdatesByDst[ru.Dst] = ru
							s.maybeNotifyResync()
						})

					} else if ru.Type == proto.RouteType_LOCAL_WORKLOAD {
						// we only care about local workloads describing this gateway, check if that's what we have
						_, dstCIDR, err := net.ParseCIDR(ru.Dst)
						if err != nil {
							log.WithError(err).Warnf("could not parse dst CIDR of RouteUpdate: %+v", ru)
							continue

						} else {
							if dstCIDR.Contains(s.gatewayIP) {
								log.Debugf("received RouteUpdate describing this gateway: %+v", ru)
								s.write(func(rs *routeStore) {
									rs.latestGatewayUpdate = ru
								})
								s.maybeNotifyResync()
							}
						}

					} else if protoutil.IsHostTunnel(ru) {
						log.Debugf("received RouteUpdate for host tunnel: %+v", ru)
						s.write(func(rs *routeStore) {
							rs.tunnelUpdatesByDst[ru.Dst] = ru
						})
						s.maybeNotifyResync()
					}

				case *proto.ToDataplane_RouteRemove:
					log.Debugf("received RouteRemove: %+v", update.Payload.(*proto.ToDataplane_RouteRemove))

					rm := payload.RouteRemove
					s.write(func(rs *routeStore) {
						delete(rs.remoteWorkloadUpdatesByDst, rm.Dst)
						delete(rs.tunnelUpdatesByDst, rm.Dst)
					})
					s.maybeNotifyResync()

				case *proto.ToDataplane_InSync:
					log.Debugf("received InSync, notifying observers...")
					// After receiving an `inSync`, all future updates over this channel will immediately notify observers
					s.inSync = true
					s.maybeNotifyResync()
				default:
					log.Debugf("Unexpected update received: %+v", update)
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
		rs.remoteWorkloadUpdatesByDst = make(map[string]*proto.RouteUpdate)
		rs.tunnelUpdatesByDst = make(map[string]*proto.RouteUpdate)
		rs.latestGatewayUpdate = nil
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
