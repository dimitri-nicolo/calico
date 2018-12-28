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
)

func setup(flush time.Duration) (statscache.StatsCache, chan statscache.DPStats, func()) {
	sc := statscache.New(flush)
	in := make(chan statscache.DPStats, 5)
	cxt, cancel := context.WithCancel(context.Background())

	sc.Start(cxt, in)
	return sc, in, cancel
}

func TestDuration(t *testing.T) {
	RegisterTestingT(t)
	sc, in, cancel := setup(3 * time.Second)

	// Send in a couple of stats for the same tuple.
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  3,
		},
	}
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  2,
		},
	}

	// Check that we get data after 3s (10% jitter), but not before.
	shouldNotReceiveAggregated(sc, 2*time.Second)
	shouldReceiveAggregated(sc, 1400*time.Millisecond, map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 2,
			HTTPRequestsDenied:  5,
		},
	})

	// Cancel the context to stop the cache.
	cancel()
}

func TestMultipleFlushes(t *testing.T) {
	RegisterTestingT(t)
	sc, in, cancel := setup(500 * time.Millisecond)

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

	shouldReceiveAggregated(sc, time.Second, map[statscache.Tuple]statscache.Values{
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
	shouldReceiveAggregated(sc, time.Second, map[statscache.Tuple]statscache.Values{
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
	sc, in, cancel := setup(500 * time.Millisecond)

	// Send in no data.
	shouldNotReceiveAggregated(sc, 550*time.Millisecond)

	// Send in some stats.
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 12,
		},
	}
	shouldReceiveAggregated(sc, time.Second, map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 12,
		},
	})

	// Send in no data.
	shouldNotReceiveAggregated(sc, 550*time.Millisecond)

	// Send in some stats.
	in <- statscache.DPStats{
		Tuple: tuple2,
		Values: statscache.Values{
			HTTPRequestsDenied: 13,
		},
	}
	shouldReceiveAggregated(sc, time.Second, map[statscache.Tuple]statscache.Values{
		tuple2: {
			HTTPRequestsDenied: 13,
		},
	})

	// Send in no data.
	shouldNotReceiveAggregated(sc, 550*time.Millisecond)

	// Cancel the context to stop the cache.
	cancel()
}

func shouldNotReceiveAggregated(sc statscache.StatsCache, timeout time.Duration) {
	select {
	case a := <-sc.Aggregated():
		Expect(a).To(BeNil())
	case <-time.After(timeout):
	}
}

func shouldReceiveAggregated(sc statscache.StatsCache, timeout time.Duration, expected map[statscache.Tuple]statscache.Values) {
	select {
	case a := <-sc.Aggregated():
		Expect(a).To(Equal(expected))
	case <-time.After(timeout):
		Expect(nil).To(Equal(expected))
	}
}
