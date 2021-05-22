// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	lmaelastic "github.com/tigera/lma/pkg/elastic"

	"github.com/projectcalico/libcalico-go/lib/jitter"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
	"github.com/tigera/es-proxy/pkg/k8s"
)

// This file provides a cached interface for pre-correlated flows, logs and events (those returned by the machinery in
// flowl3.go, flowl7.go and events.go). Flow data is filtered based on the users RBAC. Events data is not yet filtered
// by RBAC - instead we only overlay events onto correlated nodes that are in the final service graph response.
//
// The cache is a warm cache. It keeps up to _maxCacheSize_ sets of data stored so that subsequent queries requesting
// the same time range will be gathered from the cache.
// The cache contains raw, but correlated flow and event data. It is filtered after retrieving from the cache, so the
// cached data is applicable to all users.
//
// Data requested using relative times (e.g. now-15m to now) are updated in the background, so that subsequent requests
// using the same relative time interval will return the cached value.  The Force Refresh option in the service graph
// request parameters may be used if the data is not updating fast enough, but this will have an impact on the response
// times.
const (
	// Max number of cached datasets.
	maxCacheSize = 10

	// The length of time we allow for late updates in elastic search. If the end time of a request was within this
	// time we will update the data in the background so subsequent requests may contain more accurate data.
	lateUpdateDuration = 15 * time.Minute
)

type ServiceGraphCache interface {
	GetFilteredServiceGraphData(ctx context.Context, rd *RequestData) (*ServiceGraphData, error)
}

func NewServiceGraphCache(
	ctx context.Context,
	elastic lmaelastic.Client,
	clientSetFactory k8s.ClientSetFactory,
) ServiceGraphCache {
	return NewServiceGraphCacheWithBackend(
		ctx, &realServiceGraphBackend{
			ctx:              ctx,
			elastic:          elastic,
			clientSetFactory: clientSetFactory,
		},
	)
}

func NewServiceGraphCacheWithBackend(
	ctx context.Context,
	backend ServiceGraphBackend,
) ServiceGraphCache {
	sgc := &serviceGraphCache{
		cache:               make(map[cacheKey]*cacheData),
		updateLoopInterval:  2 * time.Minute,
		updateQueryInterval: 5 * time.Second,
		backend:             backend,
	}
	go sgc.backgroundCacheUpdateLoop(ctx)
	return sgc
}

type TimeSeriesFlow struct {
	Edge                 FlowEdge
	AggregatedProtoPorts *v1.AggregatedProtoPorts
	Stats                []v1.GraphStats
}

func (t TimeSeriesFlow) String() string {
	if t.AggregatedProtoPorts == nil {
		return fmt.Sprintf("L3Flow %s", t.Edge)
	}
	return fmt.Sprintf("L3Flow %s (%s)", t.Edge, t.AggregatedProtoPorts)
}

type ServiceGraphData struct {
	TimeIntervals []v1.TimeRange
	FilteredFlows []TimeSeriesFlow
	ServiceGroups ServiceGroups
	Events        []Event
}

type serviceGraphCache struct {
	// The service graph backend.
	backend ServiceGraphBackend

	// We cache a number of different sets of data. When memory usage is to high we'll age out the least used entries.
	lock  sync.Mutex
	cache map[cacheKey]*cacheData

	// Cached data queue in the order of most recently accessed first.
	queue cacheDataQueue

	// The interval at which we trigger background updates, and the minimum interval between each query within a single
	// update loop.
	updateLoopInterval  time.Duration
	updateQueryInterval time.Duration
}

// GetFilteredServiceGraphData returns RBAC filtered service graph data:
// -  correlated (source/dest) flow logs and flow stats
// -  service groups calculated from flows
// -  event IDs correlated to endpoints
// TODO(rlb): The events are not RBAC filtered, instead events are overlaid onto the filtered graph view - so the
//            presence of a graph node or not is used to determine whether or not an event is included. This will likely
//            need to be revisited when we refine RBAC control of events.
func (s *serviceGraphCache) GetFilteredServiceGraphData(ctx context.Context, rd *RequestData) (*ServiceGraphData, error) {

	// Get the raw unfiltered data that will satisfy the request.
	data, err := s.getRawDataForRequest(ctx, rd)
	if err != nil {
		return nil, err
	}

	// Construct the service graph data by filtering the L3 and L7 data.
	fd := &ServiceGraphData{
		TimeIntervals: []v1.TimeRange{rd.Request.TimeRange},
		ServiceGroups: NewServiceGroups(),
	}

	// Filter the L3 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range data.l3 {
		if !rd.RBACFilter.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		rf = rd.HostnameHelper.ProcessL3Flow(rf)

		if rf.Edge.ServicePort != nil {
			fd.ServiceGroups.AddMapping(*rf.Edge.ServicePort, rf.Edge.Dest)
		}
		stats := rf.Stats

		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge:                 rf.Edge,
			AggregatedProtoPorts: rf.AggregatedProtoPorts,
			Stats: []v1.GraphStats{{
				L3:        &stats,
				Processes: rf.Processes,
			}},
		})
	}
	fd.ServiceGroups.FinishMappings()

	// Filter the L7 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range data.l7 {
		if !rd.RBACFilter.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		rf = rd.HostnameHelper.ProcessL7Flow(rf)

		stats := rf.Stats
		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge: rf.Edge,
			Stats: []v1.GraphStats{{
				L7: &stats,
			}},
		})
	}

	// Filter the events.
	for _, ev := range data.events {
		// Update the names in the events (if required).
		ev = rd.HostnameHelper.ProcessEvent(ev)
		fd.Events = append(fd.Events, ev)
	}

	return fd, nil
}

// getRawDataForRequest returns the raw data used to fulfill a request.
func (s *serviceGraphCache) getRawDataForRequest(ctx context.Context, rd *RequestData) (*cacheData, error) {
	// Convert the time range to a set of windows that we would cache.
	key, err := s.calculateKey(rd)
	if err != nil {
		return nil, err
	}

	// Lock to access the cache. Grab the current entry or create a new entry and kick off a query. This approach allows
	// multiple concurrent accesses of the same data - but only one goroutine will create a new entry and kick off a
	// query.
	s.lock.Lock()
	var data *cacheData

	if rd.Request.ForceRefresh {
		// Request is forcing a refresh. If there is already cached data that is not pending then delete that data.
		if data = s.cache[key]; data != nil {
			select {
			case <-data.pending:
				// The cached data is not pending, so remove it - this will force a refresh.
				delete(s.cache, key)
			default:
				// The cached data is still pending, so there is no need to force another refresh.
			}
		}
	}

	if data = s.cache[key]; data == nil {
		data = newCacheData(key)
		s.cache[key] = data

		// Kick off the query on a go routine so we can unlock and unblock other go routines.
		go func() {
			defer close(data.pending)
			s.populateData(data)
		}()
	}

	// Just accessed, so move to front of queue.
	s.queue.add(data)

	// Release the lock.
	s.lock.Unlock()

	// Wait for the data to either be ready or to have errored.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-data.pending:
	}
	if data.err != nil {
		return nil, err
	}

	return data, nil
}

// backgroundCacheUpdateLoop loops until done, updating cache entries every tick.
func (s *serviceGraphCache) backgroundCacheUpdateLoop(ctx context.Context) {
	ticker := jitter.NewTicker(s.updateLoopInterval, s.updateLoopInterval/10)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cacheUpdate(ctx)
		}
	}
}

// cacheUpdate is called from backgroundCacheUpdateLoop - it performs the cache updates. It determines which entries
// should be updated and then performs a query for each. These queries are performed consecutively so we are not
// drowning elastic with requests.
func (s *serviceGraphCache) cacheUpdate(ctx context.Context) {
	log.Debug("Starting cache update cycle")

	ticker := jitter.NewTicker(s.updateQueryInterval, s.updateQueryInterval/10)
	defer ticker.Stop()

	// Grab the lock and construct the ordered set of cache datas that need updating.
	var datasToUpdate []*cacheData
	createdCutoff := time.Now().Add(-s.updateLoopInterval / 2)
	endCutoff := time.Now().Add(-lateUpdateDuration)
	s.lock.Lock()
	for data := s.queue.first; data != nil; data = data.next {
		if len(datasToUpdate) < maxCacheSize {
			datasToUpdate = append(datasToUpdate, data)
		} else {
			log.Debugf("Removing cache entry: %s", data.cacheKey)
			delete(s.cache, data.cacheKey)
			s.queue.remove(data)
		}
	}

	// Unlock and then kick off the asynchronous updates for the entries that need updating.
	s.lock.Unlock()
	for _, data := range datasToUpdate {
		log.Debugf("Checking cache entry: %s", data.cacheKey)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if data.needsUpdating(createdCutoff, endCutoff) {
				s.updateData(data.cacheKey)
			}
		}
	}

	log.Debug("Finished cache update cycle")
}

// calculateKey calculates the cache data key for the reqeust.
func (s *serviceGraphCache) calculateKey(rd *RequestData) (cacheKey, error) {
	if rd.Request.TimeRange.Now == nil {
		return cacheKey{
			relative: false,
			start:    rd.Request.TimeRange.From.Unix(),
			end:      rd.Request.TimeRange.To.Unix(),
			cluster:  rd.Request.Cluster,
		}, nil
	}
	return cacheKey{
		relative: true,
		start:    int64(rd.Request.TimeRange.Now.Sub(rd.Request.TimeRange.From) / time.Second),
		end:      int64(rd.Request.TimeRange.Now.Sub(rd.Request.TimeRange.From) / time.Second),
		cluster:  rd.Request.Cluster,
	}, nil
}

// updateData performs a new query for a cache entry and then replaces the existing entry with the update.
func (s *serviceGraphCache) updateData(key cacheKey) {
	log.Debugf("Updating cache entry: %s", key)

	dNew := newCacheData(key)
	defer close(dNew.pending)
	s.populateData(dNew)

	// grab the lock while we update the cache.
	s.lock.Lock()
	defer s.lock.Unlock()

	// Get the existing entry. If this is nil then we have removed the entry and therefore do not need to keep it
	// updated anymore, so just exit.
	dOld := s.cache[key]
	if dOld == nil {
		return
	}

	if dNew.err != nil && dOld.err == nil {
		// The latest attempt to get data resulted in an error. There is a non-errored entry in the cache, so just keep
		// that one.
		log.Debugf("Error retrieving data, keep existing data: %s", key)
		return
	}

	// Replace the entry with the new one.
	s.cache[key] = dNew
	s.queue.replace(dOld, dNew)
}

// populateData performs the various queries to get raw log data and updates the cacheData.
func (s *serviceGraphCache) populateData(d *cacheData) {
	log.Debugf("Populating data from elastic and k8s queries: %s", d.cacheKey)

	// At the moment there is no cache and only a single data point in the flow. Kick off the L3 and L7 queries at the
	// same time.
	wg := sync.WaitGroup{}
	var rawL3 []L3Flow
	var rawL7 []L7Flow
	var rawEvents []Event
	var errL3, errL7, errEvents error

	// Determine the flow config - we need this to process some of the flow data correctly.
	flowConfig, err := s.backend.GetFlowConfig(d.cluster)
	if err != nil {
		log.WithError(err).Error("failed to get felix flow configuration")
		d.err = err
		return
	}

	// Construct a time range for this data.
	var tr v1.TimeRange
	if d.relative {
		now := time.Now().UTC()
		tr.From = now.Add(time.Duration(-d.start) * time.Second)
		tr.To = now.Add(time.Duration(-d.end) * time.Second)
	} else {
		tr.From = time.Unix(d.start, 0)
		tr.To = time.Unix(d.end, 0)
	}

	// Set the actual time range in the data.
	d.timeRange = tr

	// Run simultaneous queries to get the L3, L7 and events data.
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawL3, errL3 = s.backend.GetL3FlowData(d.cluster, tr, flowConfig)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawL7, errL7 = s.backend.GetL7FlowData(d.cluster, tr)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawEvents, errEvents = s.backend.GetEvents(d.cluster, tr)
	}()
	wg.Wait()
	if errL3 != nil {
		log.WithError(err).Error("failed to get l3 logs")
		d.err = errL3
		return
	} else if errL7 != nil {
		log.WithError(err).Error("failed to get l7 logs")
		d.err = errL7
		return
	} else if errEvents != nil {
		log.WithError(err).Error("failed to get event logs")
		d.err = errEvents
		return
	}

	d.l3 = rawL3
	d.l7 = rawL7
	d.events = rawEvents
}

// cacheData contains data for a requested window.
type cacheData struct {
	cacheKey

	// Channel is closed once data has been fetched for the first time.
	pending chan struct{}

	// ==== lock required for accessing the following data ====

	// The previous and next most recently accessed entries. A nil entry indicates the end of the queue (see first and
	// last in the serviceGraphCache struct).
	prev *cacheData
	next *cacheData

	// ==== cached data:  This is read safe without any locks once the pending channel is closed ====

	// The time this entry was created.
	created time.Time

	// Error obtained attempting to fetch the data. If a failure occurred the data may be re-queried in the background,
	// this will result in a new cacheData entry that will replace this entry - the access position will remain the
	// same though - so aging-out processing can still occur based on the user access of the data. Pre-loaded data
	// is always added to the end of the queue.
	err error

	// The time range for this data.
	timeRange v1.TimeRange

	// The L3, L7 and events data.
	l3     []L3Flow
	l7     []L7Flow
	events []Event
}

func newCacheData(key cacheKey) *cacheData {
	return &cacheData{
		cacheKey: key,
		pending:  make(chan struct{}),
		created:  time.Now().UTC(),
	}
}

// needsUpdating returns true if this particular cache data should be updated.
func (d *cacheData) needsUpdating(createdCutoff, endCutoff time.Time) bool {
	select {
	case <-d.pending:
		// Data is populated.
		if d.err != nil {
			// Failed to previously fetch the data and so does need updating.
			return true
		} else if createdCutoff.Before(d.created) {
			// This entry was created recently and so does not need updating.
			return false
		} else if d.relative {
			// This indicates a time relative to "now". This entry should be updated.
			return true
		} else if endCutoff.Before(time.Unix(d.end, 0)) {
			// The entry is not relative to now and the end time of the entry is sufficiently recent we should do an
			// update to allow for late arriving data.
			return true
		}
		return false
	default:
		// Still pending an update, so does not need updating.
		return false
	}
}

// cacheDataQueue is a queue struct used for queueing cacheData for access order.
type cacheDataQueue struct {
	// Track the order these cached intervals are accessed.
	first *cacheData
	last  *cacheData
}

func (q *cacheDataQueue) add(d *cacheData) {
	if q.first == d {
		// Already the most recently accessed entry.
		return
	}
	if d.next != nil || d.prev != nil {
		// Already in the queue, so remove from the queue first.
		q.remove(d)
	}
	if q.first == nil {
		// The first entry to be added.
		q.first = d
		q.last = d
		return
	}
	q.first.prev, q.first, d.next = d, d, q.first
}

func (q *cacheDataQueue) remove(d *cacheData) {
	prev := d.prev
	next := d.next

	if prev != nil {
		prev.next = next
	} else if q.first == d {
		q.first = next
	}

	if next != nil {
		next.prev = prev
	} else if q.last == d {
		q.last = prev
	}

	d.prev = nil
	d.next = nil
}

func (q *cacheDataQueue) replace(dOld, dNew *cacheData) {
	dNew.prev = dOld.prev
	dNew.next = dOld.next

	if dNew.prev != nil {
		dNew.prev.next = dNew
	} else if q.first == dOld {
		q.first = dNew
	}
	if dNew.next != nil {
		dNew.next.prev = dNew
	} else if q.last == dOld {
		q.last = dNew
	}

	dOld.prev = nil
	dOld.next = nil
}

// cacheKey is a key for accessing cacheData. It is basically a time and window combination, allowing for times
// relative to "now".   A time range "now-15m to now" will have the same key irrespective of the actual time (now).
type cacheKey struct {
	// Whether the time is absolute or relative to now.
	relative bool

	// If "relative" is true these are the start and end Unix time in seconds.
	// If "relative" is false, these are the offsets from "now" in seconds.
	start int64
	end   int64

	// The cluster name.
	cluster string
}

func (k cacheKey) String() string {
	if k.relative {
		start := time.Duration(k.start) * time.Second
		end := time.Duration(k.end) * time.Second
		return fmt.Sprintf("%s(now-%s->now-%s)", k.cluster, start, end)
	}
	start := time.Unix(k.start, 0)
	end := time.Unix(k.end, 0)
	return fmt.Sprintf("%s(%s->%s)", k.cluster, start, end)
}
