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

func setup(flush time.Duration) (chan statscache.DPStats, chan map[statscache.Tuple]statscache.Values, func()) {
	sc := statscache.New(flush)
	in := make(chan statscache.DPStats, 5)
	out := make(chan map[statscache.Tuple]statscache.Values, 1)
	cxt, cancel := context.WithCancel(context.Background())

	sc.Start(cxt, in, out)
	return in, out, cancel
}

func TestDuration(t *testing.T) {
	RegisterTestingT(t)
	in, out, cancel := setup(3 * time.Second)

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
	Consistently(out, "1s", "100ms").ShouldNot(Receive())
	Consistently(out, "1s", "100ms").ShouldNot(Receive())
	Eventually(out, "1.4s", "100ms").Should(Receive(Equal(map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 2,
			HTTPRequestsDenied:  5,
		},
	})))

	// Cancel the context to stop the cache.
	cancel()
}

func TestMultipleFlushes(t *testing.T) {
	RegisterTestingT(t)
	in, out, cancel := setup(500 * time.Millisecond)

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
	Eventually(out).Should(Receive(Equal(map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 1,
			HTTPRequestsDenied:  3,
		},
	})))

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
	Eventually(out).Should(Receive(Equal(map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 10,
			HTTPRequestsDenied:  33,
		},
		tuple2: {
			HTTPRequestsAllowed: 15,
			HTTPRequestsDenied:  0,
		},
	})))

	// Cancel the context to stop the cache.
	cancel()
}

func TestNoData(t *testing.T) {
	RegisterTestingT(t)
	in, out, cancel := setup(500 * time.Millisecond)

	// Send in no data.
	Consistently(out, "550ms", "100ms").ShouldNot(Receive())

	// Send in some stats.
	in <- statscache.DPStats{
		Tuple: tuple1,
		Values: statscache.Values{
			HTTPRequestsAllowed: 12,
		},
	}
	Eventually(out).Should(Receive(Equal(map[statscache.Tuple]statscache.Values{
		tuple1: {
			HTTPRequestsAllowed: 12,
		},
	})))

	// Send in no data.
	Consistently(out, "550ms", "100ms").ShouldNot(Receive())

	// Send in some stats.
	in <- statscache.DPStats{
		Tuple: tuple2,
		Values: statscache.Values{
			HTTPRequestsDenied: 13,
		},
	}
	Eventually(out).Should(Receive(Equal(map[statscache.Tuple]statscache.Values{
		tuple2: {
			HTTPRequestsDenied: 13,
		},
	})))

	// Send in no data.
	Consistently(out, "550ms", "100ms").ShouldNot(Receive())

	// Cancel the context to stop the cache.
	cancel()
}
