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
package statscache_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/projectcalico/app-policy/statscache"
)

var (
	tuple1 = statscache.Tuple{
		SrcIp:    "1.2.3.4",
		DstIp:    "2.3.4.5",
		SrcPort:  10,
		DstPort:  10020,
		Protocol: "TCP",
	}

	tuple2 = statscache.Tuple{
		SrcIp:    "10.20.30.40",
		DstIp:    "20.30.40.50",
		SrcPort:  100,
		DstPort:  20020,
		Protocol: "TCP",
	}

	originalNewTicker = statscache.NewTicker
)

func setup(flush time.Duration) (statscache.StatsCache, chan statscache.DPStats, func()) {
	sc := statscache.New(flush)
	in := make(chan statscache.DPStats)
	cxt, cancel := context.WithCancel(context.Background())

	sc.Start(cxt, in)
	return sc, in, cancel
}

func TestDuration(t *testing.T) {
	RegisterTestingT(t)

	sc, in, cancel := setup(50 * time.Millisecond)

	// Send in stats and check we receive them in an aggregated stats update. We send the stats after a short
	// delay to ensure we are ready to receive the aggregated stats.
	go func() {
		time.Sleep(20 * time.Millisecond)
		in <- statscache.DPStats{
			Tuple: tuple1,
			Values: statscache.Values{
				HTTPRequestsAllowed: 1,
				HTTPRequestsDenied:  3,
			},
		}
	}()
	shouldReceiveAggregated(sc, 120*time.Millisecond, map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  3,
		},
	})

	// Cancel the context to stop the cache.
	cancel()
}

func TestMultipleFlushes(t *testing.T) {
	RegisterTestingT(t)

	// Mock out the flush ticker so that we can control when we flush.
	ticks := mockoutNewTicker(3 * time.Millisecond)
	defer reinstateNewTicker()
	sc, in, cancel := setup(3 * time.Millisecond)

	// Send in a couple of stats with the same tuple.
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 1,
		},
	}
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsDenied: 3,
		},
	}

	// Check that we receive aggregated stats once the ticker has ticked.
	go func() {
		// Pause before sending the tick to ensure we are reading from the Aggregated stats channel.
		time.Sleep(20 * time.Millisecond)
		ticks <- time.Now()
	}()
	shouldReceiveAggregated(sc, 100*time.Millisecond, map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  3,
		},
	})

	// Send in more stats with different tuples.
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 10,
			HTTPRequestsDenied:  33,
		},
	}
	in <- statscache.DPStats{
		Tuple: tuple2,
		Values: statscache.Values{
			HTTPRequestsAllowed: 15,
		},
	}

	// Check that we receive aggregated stats once the ticker has ticked.
	go func() {
		// Pause before sending the tick to ensure we are reading from the Aggregated stats channel.
		time.Sleep(20 * time.Millisecond)
		ticks <- time.Now()
	}()
	shouldReceiveAggregated(sc, 100*time.Millisecond, map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 10,
			HTTPRequestsDenied:  33,
		},
		tuple2: {
			HTTPRequestsAllowed: 15,
			HTTPRequestsDenied:  0,
		},
	})

	// Cancel the context to stop the cache.
	cancel()
}

func TestNoData(t *testing.T) {
	RegisterTestingT(t)

	// Mockout the flush ticker so that we can control when we flush.
	ticks := mockoutNewTicker(7 * time.Millisecond)
	defer reinstateNewTicker()
	sc, in, cancel := setup(7 * time.Millisecond)

	// Do multiple ticks without sending in any data. We should not receive any empty
	// aggregated stats.
	go func() {
		// Pause before sending the tick to ensure we are reading from the Aggregated stats channel.
		time.Sleep(20 * time.Millisecond)

		// Send multiple ticks (since the channel is blocking, we can guarantee that the previous ticks
		// have been processed.
		ticks <- time.Now()
		ticks <- time.Now()
		ticks <- time.Now()
	}()
	shouldNotReceiveAggregated(sc, 200*time.Millisecond)

	// Send in some stats now. (The call to shouldNotReceiveAggregated is long enough for all previous ticks to
	// have bee processed).
	in <- statscache.DPStats{
		Tuple: tuple2,
		Values: statscache.Values{
			HTTPRequestsDenied: 13,
		},
	}
	go func() {
		// Pause before sending the tick to ensure we are reading from the Aggregated stats channel.
		time.Sleep(20 * time.Millisecond)
		ticks <- time.Now()
	}()
	shouldReceiveAggregated(sc, 100*time.Millisecond, map[statscache.Tuple]statscache.Values{
		tuple2: {
			HTTPRequestsDenied: 13,
		},
	})

	// Send in no data.
	shouldNotReceiveAggregated(sc, 550*time.Millisecond)

	// Cancel the context to stop the cache.
	cancel()
}

func TestNoReceiver(t *testing.T) {
	RegisterTestingT(t)

	// Mockout the flush ticker so that we can control when we flush.
	ticks := mockoutNewTicker(7 * time.Millisecond)
	defer reinstateNewTicker()
	sc, in, cancel := setup(7 * time.Millisecond)

	// Send in some stats whilst no one is ready to receive.
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 12,
		},
	}

	// Send multiple ticks to ensure the agregated stats have been flushed.
	ticks <- time.Now()
	ticks <- time.Now()
	shouldNotReceiveAggregated(sc, 100*time.Millisecond)

	// Send in some stats now. (The call to shouldNotReceiveAggregated is long enough for all previous ticks to
	// have bee processed).
	in <- statscache.DPStats{
		Tuple: tuple2,
		Values: statscache.Values{
			HTTPRequestsDenied: 13,
		},
	}
	go func() {
		// Pause before sending in some stats and another flush tick. We should receive these stats.
		time.Sleep(20 * time.Millisecond)
		ticks <- time.Now()
	}()
	shouldReceiveAggregated(sc, 100*time.Millisecond, map[statscache.Tuple]statscache.Values{
		tuple2: {
			HTTPRequestsDenied: 13,
		},
	})

	// Cancel the context to stop the cache.
	cancel()
}

// mockoutNewTicker replaces the statscache.NewTicker helper method with one that allows us
// direct access to the tick channel. Timer ticks are then controlled by the test code.
func mockoutNewTicker(expectedDuration time.Duration) chan<- time.Time {
	mockTicker := time.NewTicker(time.Hour)
	ticks := make(chan time.Time)
	mockTicker.C = ticks
	statscache.NewTicker = func(d time.Duration) *time.Ticker {
		Expect(d).To(Equal(expectedDuration))
		return mockTicker
	}
	return ticks
}

// reinstateNewTicker reinstates the original statscache.NewTicker helper method.
func reinstateNewTicker() {
	statscache.NewTicker = originalNewTicker
}

// shouldNotReceiveAggregated checks that nothing is received over the Aggregated stats channel.
// It is not possible to use the ginkgo methods for testing since they perform polled checks on
// the channel and the statscache will drop aggregated stats if there is noone to receive them.
func shouldNotReceiveAggregated(sc statscache.StatsCache, timeout time.Duration) {
	select {
	case a := <-sc.Aggregated():
		Expect(a).To(BeNil())
	case <-time.After(timeout):
	}
}

// shouldReceiveAggregated checks that something is received over the Aggregated stats channel.
// It is not possible to use the ginkgo methods for testing since they perform polled checks on
// the channel and the statscache will drop aggregated stats if there is noone to receive them.
func shouldReceiveAggregated(sc statscache.StatsCache, timeout time.Duration, expected map[statscache.Tuple]statscache.Values) {
	select {
	case a := <-sc.Aggregated():
		Expect(a).To(Equal(expected))
	case <-time.After(timeout):
		Expect(nil).To(Equal(expected))
	}
}
