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
	"time"
)

// Tuple encapsulates the 5-tuple connection information.
type Tuple struct {
	SrcIp    string
	DstIp    string
	SrcPort  int32
	DstPort  int32
	Protocol string
}

// Values contains a set of statistic values that can be aggregated.
type Values struct {
	HTTPRequestsAllowed int64
	HTTPRequestsDenied  int64
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

// The statscache interface.
type StatsCache interface {
	Start(cxt context.Context, dpStats <-chan DPStats)
	Aggregated() <-chan map[Tuple]Values
}

// New() creates a new statscache.
func New(flushInterval time.Duration) StatsCache {
	return &statsCache{
		flushInterval: flushInterval,
		stats:         make(map[Tuple]Values),
		aggregated:    make(chan map[Tuple]Values),
	}
}

// Start starts the statscache collection, aggregation and reporting.
// Statistics are fed in through the dpStats channel, and the aggregated statistics are reported through
// the aggregated channel with a flush interval defined by `flushInterval`.
func (s *statsCache) Start(cxt context.Context, dpStats <-chan DPStats) {
	go s.run(cxt, dpStats)
}

// Aggregated returns the aggregated stats channel. If no goroutine is actively reading from this channel
// when the stats are flushed, the stats will be dropped.
func (s *statsCache) Aggregated() <-chan map[Tuple]Values {
	return s.aggregated
}

type statsCache struct {
	flushInterval time.Duration
	stats         map[Tuple]Values
	aggregated    chan map[Tuple]Values
}

// run is the main loop that pulls stats from the dsStats channel and periodically reports aggregated
// stats through the aggregated channel.
func (s *statsCache) run(cxt context.Context, dpStats <-chan DPStats) {
	flushStatsTicker := time.NewTicker(s.flushInterval)
	defer flushStatsTicker.Stop()

	for {
		select {
		case d := <-dpStats:
			s.add(d)
		case <-flushStatsTicker.C:
			s.flush()
		case <-cxt.Done():
			return
		}
	}
}

// add adds the supplied DPStats to the current cache, either creating a new entry, or aggregating
// into the existing entry.
func (s *statsCache) add(d DPStats) {
	if v, ok := s.stats[d.Tuple]; ok {
		// Entry already exists in cache, increment stats in existing entry.
		s.stats[d.Tuple] = v.add(d.Values)
		return
	}
	// Entry does not exist, add new entry.
	s.stats[d.Tuple] = d.Values
}

// flush sends the current aggregated cache, and creates a new empty cache.
func (s *statsCache) flush() {
	if len(s.stats) == 0 {
		// No stats, so nothing to report.
		return
	}

	// Report the current set of stats, and create a new empty stats cache.
	select {
	case s.aggregated <- s.stats:
		// Aggregated stats successfully received by remote end.
	default:
		// No go-routine waiting on the stats. Drop them.
	}
	s.stats = make(map[Tuple]Values)
}
