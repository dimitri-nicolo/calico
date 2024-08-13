// Copyright (c) 2018 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statscache

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultStatsFlushInterval = 5 * time.Second
)

// Tuple encapsulates the 5-tuple connection information.
type Tuple struct {
	SrcIp    string
	DstIp    string
	SrcPort  int32
	DstPort  int32
	Protocol string
}

func (t Tuple) String() string {
	return fmt.Sprintf("Stats(%s %s:%d to %s:%d)", t.Protocol, t.SrcIp, t.SrcPort, t.DstIp, t.DstPort)
}

// Values contains a set of statistic values that can be aggregated.
type Values struct {
	HTTPRequestsAllowed int64
	HTTPRequestsDenied  int64
}

func (v Values) String() string {
	return fmt.Sprintf("{DeltaHTTPReqAllowed: %d; DeltaHTTPReqDenied: %d}", v.HTTPRequestsAllowed, v.HTTPRequestsDenied)
}

func (v Values) add(v2 Values) Values {
	return Values{
		HTTPRequestsDenied:  v.HTTPRequestsDenied + v2.HTTPRequestsDenied,
		HTTPRequestsAllowed: v.HTTPRequestsAllowed + v2.HTTPRequestsAllowed,
	}
}

// DPStats contains dataplane statistics for a single connection.
type DPStats struct {
	Tuple  Tuple
	Values Values
}

func (d DPStats) String() string {
	return fmt.Sprintf("%v=%v", d.Tuple, d.Values)
}

// The statscache interface.
type StatsCache interface {
	Start(context.Context)
	Add(DPStats)
	Flush()
	RegisterFlushCallback(StatsCacheFlushCallback)
}

type StatsCacheFlushCallback func(map[Tuple]Values)

// New() creates a new statscache.
func New(opts ...StatsCacheOption) StatsCache {
	return NewWithFlushInterval(DefaultStatsFlushInterval, opts...)
}

type StatsCacheOption func(*statsCache)

func WithTicker(t LazyTicker) StatsCacheOption {
	return func(s *statsCache) {
		s.ticker = t
	}
}

func WithStatsFlushCallbacks(callbacks ...StatsCacheFlushCallback) StatsCacheOption {
	return func(s *statsCache) {
		s.callbacks = callbacks
	}
}

func NewWithFlushInterval(flushInterval time.Duration, opts ...StatsCacheOption) StatsCache {
	res := &statsCache{
		ticker:    NewLazyTicker(flushInterval),
		stats:     make(map[Tuple]Values),
		callbacks: make([]StatsCacheFlushCallback, 0),
	}
	for _, o := range opts {
		o(res)
	}
	return res
}

// Start starts the statscache collection, aggregation and reporting.
func (s *statsCache) Start(cxt context.Context) {
	log.Debug("Starting statistics consolidation and reporting")
	go s.ticker.Start(cxt)
	for {

		select {
		case <-s.ticker.C():
			log.Debug("ticker fired")
			s.Flush()
		case <-cxt.Done():
			return
		}
	}
}

type statsCache struct {
	ticker    LazyTicker
	events    []DPStats
	stats     map[Tuple]Values
	mu        sync.Mutex
	callbacks []StatsCacheFlushCallback
}

func (s *statsCache) RegisterFlushCallback(cb StatsCacheFlushCallback) {
	s.callbacks = append(s.callbacks, cb)
}

// Add adds the supplied DPStats to the current cache, either creating a new entry, or aggregating
// into the existing entry.
func (s *statsCache) Add(d DPStats) {
	s.events = append(s.events, d)
}

func (s *statsCache) processEvents() {
	for _, d := range s.events {
		log.Debugf("Caching statistic: %v", d)
		if v, ok := s.stats[d.Tuple]; ok {
			// Entry already exists in cache, increment stats in existing entry.
			s.stats[d.Tuple] = v.add(d.Values)
			return
		}
		// Entry does not exist, add new entry.
		s.stats[d.Tuple] = d.Values
	}
}

// Flush sends the current aggregated cache, and creates a new empty cache.
func (s *statsCache) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Trace("Flushing cached statistics")

	s.processEvents()

	if len(s.stats) == 0 {
		// No stats, so nothing to report.
		log.Debug("No statistics to report")
		return
	}

	log.Trace("Reporting cached statistics")
	for _, cb := range s.callbacks {
		cb(s.stats)
	}
	log.Trace("Clearing cached statistics")
	s.events = nil
	s.stats = map[Tuple]Values{}
}
