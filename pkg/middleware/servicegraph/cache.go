// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/jitter"

	lmav1 "github.com/tigera/lma/pkg/apis/v1"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file provides a cache-backed interface for service graph data.
//
// The cache is a warm cache. It keeps a number of data sets cached so that subsequent queries requiring the same
// underlying data will be handled from the cache. The cache contains unfiltered correlated L3 and L7 flows, and events.
// Each set of cached data may be accessed by any user because the raw cached data is post-processed to provide a
// user-specific subset of data.
//
// Data requested using relative times (e.g. now-15m to now) are updated in the background, so that subsequent requests
// using the same relative time interval will return regularly updated cached values for the same relative range.
// The Force Refresh option in the service graph request parameters may be used if the data is not updating fast enough,
// but that will obviously impact response times because the cache would then be cold for that request.
//
// There are a number of different configuration parameters available to configure the size, refresh interval and max
// age of cache entries. See pkg/server/config.go for details.
// TODO(rlb): Future iterations may use runtime stats to determine how the cache grows and ages out, and perhaps control
//            garbage collection.

type ServiceGraphCache interface {
	GetFilteredServiceGraphData(ctx context.Context, rd *RequestData) (*ServiceGraphData, error)
	GetCacheSize() int
}

func NewServiceGraphCache(
	ctx context.Context,
	backend ServiceGraphBackend,
	cfg *Config,
) ServiceGraphCache {
	sgc := &serviceGraphCache{
		cache:   make(map[cacheKey]*cacheData),
		backend: backend,
		cfg:     cfg,
	}
	go sgc.backgroundCacheUpdateLoop(ctx)
	return sgc
}

type TimeSeriesFlow struct {
	Edge                 FlowEdge
	AggregatedProtoPorts *v1.AggregatedProtoPorts
	Stats                []v1.GraphStats
}

type TimeSeriesDNS struct {
	Endpoint FlowEndpoint
	Stats    []v1.GraphStats
}

func (t TimeSeriesFlow) String() string {
	if t.AggregatedProtoPorts == nil {
		return fmt.Sprintf("L3Flow %s", t.Edge)
	}
	return fmt.Sprintf("L3Flow %s (%s)", t.Edge, t.AggregatedProtoPorts)
}

type ServiceGraphData struct {
	TimeIntervals         []lmav1.TimeRange
	FilteredFlows         []TimeSeriesFlow
	FilteredDNSClientLogs []TimeSeriesDNS
	ServiceGroups         ServiceGroups
	NameHelper            NameHelper
	Events                []Event
}

type serviceGraphCache struct {
	// The service graph backend.
	backend ServiceGraphBackend

	// The service graph config
	cfg *Config

	// We cache a number of different sets of data. When memory usage is to high we'll age out the least used entries.
	lock  sync.Mutex
	cache map[cacheKey]*cacheData

	// Cached data queue in the order of most recently accessed first.
	queue cacheDataQueue
}

// GetCacheSize returns the current number of entries in the cache. Mostly for testing purposes.
func (s *serviceGraphCache) GetCacheSize() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.cache)
}

// GetFilteredServiceGraphData returns RBAC filtered service graph data:
// -  correlated (source/dest) flow logs and flow stats
// -  service groups calculated from flows
// -  event IDs correlated to endpoints
// TODO(rlb): The events are not RBAC filtered, instead events are overlaid onto the filtered graph view - so the
//            presence of a graph node or not is used to determine whether or not an event is included. This will likely
//            need to be revisited when we refine RBAC control of events.
func (s *serviceGraphCache) GetFilteredServiceGraphData(ctx context.Context, rd *RequestData) (*ServiceGraphData, error) {
	// Run the following queries in parallel.
	// - Get the RBAC filter
	// - Get the host name mapping helper
	// - Get the raw data.
	log.Debugf("GetFilteredServiceGraphData called with time range: %s", rd.ServiceGraphRequest.TimeRange)
	var cacheData *cacheData
	var rbacFilter RBACFilter
	var nameHelper NameHelper
	var errCacheData, errRBACFilter, errNameHelper error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		rbacFilter, errRBACFilter = s.backend.NewRBACFilter(ctx, rd)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		nameHelper, errNameHelper = s.backend.NewNameHelper(ctx, rd)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		cacheData, errCacheData = s.getRawDataForRequest(ctx, rd)
	}()
	wg.Wait()
	if errRBACFilter != nil {
		log.WithError(errRBACFilter).Error("Failed to load users permissions")
		return nil, errRBACFilter
	} else if errNameHelper != nil {
		log.WithError(errNameHelper).Error("Failed to load name mappings")
		return nil, errNameHelper
	} else if errCacheData != nil {
		log.WithError(errNameHelper).Error("Failed to load raw graph data")
		return nil, errCacheData
	}

	// Construct the service graph data by filtering the L3 and L7 data. Return the time range of the actual data
	// rather than the request.
	fd := &ServiceGraphData{
		TimeIntervals: []lmav1.TimeRange{cacheData.timeRange},
		ServiceGroups: NewServiceGroups(),
		NameHelper:    nameHelper,
	}

	// Filter the L3 flows based on RBAC. All other graph content is removed through graph pruning.
	for _, rf := range cacheData.l3 {
		if !rbacFilter.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		rf = nameHelper.ConvertL3Flow(rf)

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
	for _, rf := range cacheData.l7 {
		if !rbacFilter.IncludeFlow(rf.Edge) {
			continue
		}

		// Update the names in the flow (if required).
		rf = nameHelper.ConvertL7Flow(rf)

		stats := rf.Stats
		fd.FilteredFlows = append(fd.FilteredFlows, TimeSeriesFlow{
			Edge: rf.Edge,
			Stats: []v1.GraphStats{{
				L7: &stats,
			}},
		})
	}

	// Filter the DNS logs based on RBAC. All other graph content is removed through graph pruning.
	for _, dl := range cacheData.dns {
		if !rbacFilter.IncludeEndpoint(dl.Endpoint) {
			continue
		}

		stats := dl.Stats
		fd.FilteredDNSClientLogs = append(fd.FilteredDNSClientLogs, TimeSeriesDNS{
			Endpoint: dl.Endpoint,
			Stats: []v1.GraphStats{{
				DNS: &stats,
			}},
		})
	}

	// Filter the events.
	for _, ev := range cacheData.events {
		// Update the names in the events (if required).
		ev = nameHelper.ConvertEvent(ev)
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
	log.Debugf("Getting raw data for %s", key)

	// Lock to access the cache. Grab the current entry or create a new entry and kick off a query. This approach allows
	// multiple concurrent accesses of the same data - but only one goroutine will create a new entry and kick off a
	// query.
	s.lock.Lock()
	var data *cacheData

	if rd.ServiceGraphRequest.ForceRefresh {
		// Requested a forced refresh. If there is already cached data that is not pending then delete that data.
		if data = s.getData(key); data != nil {
			select {
			case <-data.pending:
				// The cached data is not pending, so remove it - this will force a refresh.
				s.removeData(data)
			default:
				// The cached data is still pending, so there is no need to force another refresh.
			}
		}
	}

	if data = s.getData(key); data == nil {
		data = newCacheData(key, time.Now().UTC())
		s.addData(data)

		// Kick off the query on a go routine so we can unlock and unblock other go routines.
		go func() {
			s.populateData(data)

			// If the entry needs discarding then remove it...
			s.lock.Lock()
			defer s.lock.Unlock()
			if data.discard() {
				s.removeData(data)
			}

			// and tidy the cache to maintain cache size and age out old entries.
			s.tidyCache()
		}()
	} else {
		s.touchData(data)
	}

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

// getData returns the cached data for the specified key.
//
// Lock is held by caller.
func (s *serviceGraphCache) getData(k cacheKey) *cacheData {
	return s.cache[k]
}

// touchData updates the accessed time and moves the data to the front of the queue.
//
// Lock is held by caller.
func (s *serviceGraphCache) touchData(d *cacheData) {
	d.accessed = time.Now().UTC()
	s.queue.add(d)
}

// removeData removes the data from the cache.
//
// Lock is held by caller.
func (s *serviceGraphCache) removeData(d *cacheData) {
	delete(s.cache, d.cacheKey)
	s.queue.remove(d)
}

// addData adds the data to the cache.
//
// Lock is held by caller.
func (s *serviceGraphCache) addData(d *cacheData) {
	s.cache[d.cacheKey] = d
	s.queue.add(d)
}

// replaceData replaces the data in the cache and ensures it maintains its position in the access queue.
//
// Lock is held by caller.
func (s *serviceGraphCache) replaceData(d *cacheData) {
	old := s.cache[d.cacheKey]
	if old != nil {
		s.cache[d.cacheKey] = d
		s.queue.replace(old, d)
	}
}

// tidyCache is called after adding new entries to the cache, or during the update poll. It removes oldest entries from
// the cache to maintain cache size and removes polled entries that have not been accessed for a long time (to avoid
// continuously polling).
//
// Lock is held by caller.
func (s *serviceGraphCache) tidyCache() {
	// Access cutoff time.
	cutoff := time.Now().UTC().Add(-s.cfg.ServiceGraphCachePolledEntryAgeOut)

	// Remove all aged-out relative time entries - this avoid unnecessary polling of elastic.
	data := s.queue.first
	for data != nil {
		next := data.next
		if data.relative && data.accessed.Before(cutoff) {
			log.Debugf("Removing aged out cache entry: %s", data.cacheKey)
			s.removeData(data)
		}
		data = next
	}

	// Remove oldest entries to maintain cache size.  It is fine if this removes entries that are still pending - any
	// API call that is waiting for the data already has the data pointer and that data will be still be updated.
	for len(s.cache) > s.cfg.ServiceGraphCacheMaxEntries {
		log.Debugf("Removing cache entry to keep cache size maintained: %s", s.queue.last.cacheKey)
		s.removeData(s.queue.last)
	}
}

// backgroundCacheUpdateLoop loops until done, updating cache entries every tick.
func (s *serviceGraphCache) backgroundCacheUpdateLoop(ctx context.Context) {
	loopTicker := jitter.NewTicker(s.cfg.ServiceGraphCachePollLoopInterval, s.cfg.ServiceGraphCachePollLoopInterval/10)
	defer loopTicker.Stop()
	queryTicker := jitter.NewTicker(s.cfg.ServiceGraphCachePollQueryInterval, s.cfg.ServiceGraphCachePollQueryInterval/10)
	defer queryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-loopTicker.C:
			log.Debug("Starting cache update cycle")
		}

		// Grab the lock and construct the set of cache datas that need updating.
		var datasToUpdate []*cacheData
		createdCutoff := time.Now().Add(-s.cfg.ServiceGraphCachePollLoopInterval / 2)
		settleCutoff := time.Now().Add(-s.cfg.ServiceGraphCacheDataSettleTime)

		// Start by tidying the cache and then loop through remaining cache entries to see which need updating.
		s.lock.Lock()
		s.tidyCache()
		for data := s.queue.first; data != nil; data = data.next {
			if data.needsUpdating(createdCutoff, settleCutoff) {
				// This cache entry needs updating.
				datasToUpdate = append(datasToUpdate, data)
			}
		}
		s.lock.Unlock()

		for _, data := range datasToUpdate {
			log.Debugf("Checking cache entry: %s", data.cacheKey)
			select {
			case <-ctx.Done():
				return
			case <-queryTicker.C:
				s.updateCachedData(data)
			}
		}

		log.Debug("Finished cache update cycle")
	}
}

// calculateKey calculates the cache data key for the reqeust.
func (s *serviceGraphCache) calculateKey(rd *RequestData) (cacheKey, error) {
	if rd.ServiceGraphRequest.TimeRange.Now == nil {
		return cacheKey{
			relative: false,
			start:    rd.ServiceGraphRequest.TimeRange.From.Unix(),
			end:      rd.ServiceGraphRequest.TimeRange.To.Unix(),
			cluster:  rd.ServiceGraphRequest.Cluster,
		}, nil
	}
	return cacheKey{
		relative: true,
		start:    int64(rd.ServiceGraphRequest.TimeRange.Now.Sub(rd.ServiceGraphRequest.TimeRange.From) / time.Second),
		end:      int64(rd.ServiceGraphRequest.TimeRange.Now.Sub(rd.ServiceGraphRequest.TimeRange.To) / time.Second),
		cluster:  rd.ServiceGraphRequest.Cluster,
	}, nil
}

// updateCachedData performs a new query for a cache entry and then replaces the existing entry with the update.
func (s *serviceGraphCache) updateCachedData(dataOld *cacheData) {
	log.Debugf("Updating cache entry: %s", dataOld.cacheKey)

	dataNew := newCacheData(dataOld.cacheKey, dataOld.accessed)
	s.populateData(dataNew)

	// grab the lock while we update the cache.
	s.lock.Lock()
	defer s.lock.Unlock()

	if dataNew.discard() || (dataNew.err != nil && dataOld.err == nil) {
		// The latest attempt to get data resulted in an error. There is a non-errored entry in the cache, so just keep
		// that one.
		log.Debugf("Error retrieving data, keep existing data: %s", dataOld.cacheKey)
		return
	}

	// Replace the entry with the new one.
	s.replaceData(dataNew)
}

// populateData performs the various queries to get raw log data and updates the cacheData.
func (s *serviceGraphCache) populateData(d *cacheData) {
	log.Debugf("Populating data from elastic and k8s queries: %s", d.cacheKey)

	// When this finishes, close the pending channel so threads waiting for this to populate can complete.
	defer close(d.pending)

	// At the moment there is no cache and only a single data point in the flow. Kick off the L3 and L7 queries at the
	// same time.
	wg := sync.WaitGroup{}
	var rawL3 []L3Flow
	var rawL7 []L7Flow
	var rawDNS []DNSLog
	var rawEvents []Event
	var errL3, errL7, errDNS, errEvents error

	// Determine the flow config - we need this to process some of the flow data correctly.
	flowConfig, err := s.backend.GetFlowConfig(d.cluster)
	if err != nil {
		log.WithError(err).Error("failed to get felix flow configuration")
		d.err = err
		return
	}

	// Construct a time range for this data.
	var tr lmav1.TimeRange
	if d.relative {
		tr.From = d.created.Add(time.Duration(-d.start) * time.Second)
		tr.To = d.created.Add(time.Duration(-d.end) * time.Second)
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
		rawDNS, errDNS = s.backend.GetDNSData(d.cluster, tr)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		rawEvents, errEvents = s.backend.GetEvents(d.cluster, tr)
	}()
	wg.Wait()
	if errL3 != nil {
		log.WithError(errL3).Error("failed to get l3 logs")
		d.err = errL3
	} else if errL7 != nil {
		log.WithError(errL7).Error("failed to get l7 logs")
		d.err = errL7
	} else if errDNS != nil {
		log.WithError(errDNS).Error("failed to get DNS logs")
		d.err = errDNS
	} else if errEvents != nil {
		log.WithError(errEvents).Error("failed to get event logs")
		d.err = errEvents
	}

	d.l3 = rawL3
	d.l7 = rawL7
	d.dns = rawDNS
	d.events = rawEvents

	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug(" ========= Tracing output from raw queries ========= ")
		if b, err := json.Marshal(d.l3); err == nil {
			log.Debugf("RawL3: %s", b)
		}
		if b, err := json.Marshal(d.l7); err == nil {
			log.Debugf("RawL7: %s", b)
		}
		if b, err := json.Marshal(d.dns); err == nil {
			log.Debugf("RawDNS: %s", b)
		}
		if b, err := json.Marshal(d.events); err == nil {
			log.Debugf("RawEvents: %s", b)
		}
		log.Debug(" ========= End of tracing ========= ")
	}

	log.Debugf("Updated data: %s", d.cacheKey)
}

// cacheData contains data for a requested window.
type cacheData struct {
	cacheKey

	// Channel is closed once data has been fetched for the first time.
	pending chan struct{}

	// ==== lock required for accessing the following data ====

	// The previous and next most recently accessed entries. A nil entry indicates the end of the queue (see
	// cacheDataQueue below).
	prev *cacheData
	next *cacheData

	// The time this entry was last accessed. We use this to start removing relative time entries that have not been
	// accessed for some amount of time so that we don't just keep querying for ever. Fixed time entries can remain in
	// the cache until they are aged out through cache size and access order.
	accessed time.Time

	// ==== cached data:  This is read safe without any locks once the pending channel is closed ====

	// The time this entry was created.
	created time.Time

	// Error obtained attempting to fetch the data. If a failure occurred the data may be re-queried in the background,
	// this will result in a new cacheData entry that will replace this entry - the access position will remain the
	// same though - so aging-out processing can still occur based on the user access of the data. Pre-loaded data
	// is always added to the end of the queue.
	err error

	// The time range for this data.
	timeRange lmav1.TimeRange

	// The L3, L7 and events data.
	l3     []L3Flow
	l7     []L7Flow
	dns    []DNSLog
	events []Event
}

func newCacheData(key cacheKey, accessed time.Time) *cacheData {
	return &cacheData{
		cacheKey: key,
		pending:  make(chan struct{}),
		created:  time.Now().UTC(),
		accessed: accessed,
	}
}

// discard returns true if this entry only contains errors and is therefore not worth caching.
func (d *cacheData) discard() bool {
	return d.err != nil && len(d.l3) == 0 && len(d.l7) == 0 && len(d.events) == 0
}

// needsUpdating returns true if this particular cache data should be updated.
func (d *cacheData) needsUpdating(createdCutoff, settleCutoff time.Time) bool {
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
		} else if settleCutoff.Before(time.Unix(d.end, 0)) {
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

// add adds the cached data to the front of the queue. This may be called with data already in the queue.
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

// add removes the cached data from the queue. This may be called with data not in the queue.
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

// replace replaces the old cached data with the new data maintaining the position in the queue. This may be called
// with data not in the queue.
func (q *cacheDataQueue) replace(dataOld, dataNew *cacheData) {
	dataNew.prev = dataOld.prev
	dataNew.next = dataOld.next

	if dataNew.prev != nil {
		dataNew.prev.next = dataNew
	} else if q.first == dataOld {
		q.first = dataNew
	}
	if dataNew.next != nil {
		dataNew.next.prev = dataNew
	} else if q.last == dataOld {
		q.last = dataNew
	}

	dataOld.prev = nil
	dataOld.next = nil
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
