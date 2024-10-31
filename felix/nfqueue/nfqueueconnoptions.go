// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package nfqueue

import (
	"time"

	gonfqueue "github.com/florianl/go-nfqueue"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/projectcalico/calico/felix/netlinkshim"
	"github.com/projectcalico/calico/felix/timeshim"
)

type Option func(c *nfQueueConnector)

// OptMaxQueueLength sets the maximum length of the queue.
func OptMaxQueueLength(m uint32) Option {
	return func(c *nfQueueConnector) {
		c.maxQueueLen = m
	}
}

// OptMaxPacketLength sets the maximum length of the packet copied into user space.
func OptMaxPacketLength(m uint32) Option {
	return func(c *nfQueueConnector) {
		c.maxPacketLen = m
	}
}

// OptMaxHoldTime sets the maximum hold time of each packet. Packets are released with the specific verdict
// once the time has elapsed.
func OptMaxHoldTime(m time.Duration) Option {
	return func(c *nfQueueConnector) {
		c.maxHoldTime = m
	}
}

// OptHoldTimeCheckInterval sets the interval used for checking whether the max hold time has passed for each packet.
func OptHoldTimeCheckInterval(h time.Duration) Option {
	return func(c *nfQueueConnector) {
		c.holdTimeCheckInterval = h
	}
}

// OptFailOpen causes the kernel to automatically ACCEPT packets when the NF Queue is full.
func OptFailOpen() Option {
	return func(c *nfQueueConnector) {
		c.flags |= gonfqueue.NfQaCfgFlagFailOpen
	}
}

// OptVerdictAccept sets the verdict (for explicit release or via timeout) to ACCEPT.
func OptVerdictAccept() Option {
	return func(c *nfQueueConnector) {
		c.verdict = gonfqueue.NfAccept
		c.dnrMark = 0
	}
}

// OptVerdictAccept sets the verdict (for explicit release or via timeout) to DROP.
func OptVerdictDrop() Option {
	return func(c *nfQueueConnector) {
		c.verdict = gonfqueue.NfDrop
		c.dnrMark = 0
	}
}

// OptVerdictAccept sets the verdict (for explicit release or via timeout) to DROP. A do-not-repeat mark must be
// specified.
func OptVerdictRepeat(dnrMark uint32) Option {
	return func(c *nfQueueConnector) {
		c.verdict = gonfqueue.NfRepeat
		c.dnrMark = dnrMark
	}
}

// OptTimeShim sets the timeshim. Used for testing.
func OptTimeShim(t timeshim.Interface) Option {
	return func(c *nfQueueConnector) {
		c.time = t
	}
}

// OptNfQueueFactoryShim sets the NfQueue factory. Used for testing.
func OptNfQueueFactoryShim(n func(config *gonfqueue.Config) (netlinkshim.NfQueue, error)) Option {
	return func(c *nfQueueConnector) {
		c.nfqFactory = n
	}
}

// OptPacketsHeldGauge sets prometheus gauge for queue length. This provides a measure of the number of packets
// currently in user-space awaiting release.
func OptPacketsHeldGauge(p prometheus.Gauge) Option {
	return func(c *nfQueueConnector) {
		c.prometheusPacketsHeldGauge = p
	}
}

// OptShutdownCounter sets prometheus counter for queue shutdowns.
func OptShutdownCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusShutdownCounter = p
	}
}

// OptSetVerdictFailureCounter sets prometheus counter for failures setting the verdict.
func OptSetVerdictFailureCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusSetVerdictFailureCounter = p
	}
}

// OptPacketsSeenCounter sets prometheus counter for packets seen (valid or not).
func OptPacketsSeenCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusPacketsSeenCounter = p
	}
}

// OptDNRDroppedCounter sets prometheus counter for packets dropped because they have the Do-Not-Repeast mark set.
func OptDNRDroppedCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusDNRDroppedCounter = p
	}
}

// OptNoPayloadDroppedCounter sets prometheus counter for packets dropped because they have no payload.
func OptNoPayloadDroppedCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusNoPayloadDroppedCounter = p
	}
}

// OptConnectionClosedDroppedCounter sets prometheus counter for packets dropped because the connection was explicitly
// closed.
func OptConnectionClosedDroppedCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusConnectionClosedDroppedCounter = p
	}
}

// OptPacketReleasedAfterHoldTimeCounter sets prometheus counter for packets released after the hold time.
func OptPacketReleasedAfterHoldTimeCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusReleasedAfterHoldTimeCounter = p
	}
}

// OptPacketReleasedCounter sets prometheus counter for packets explicitly released by the client.
func OptPacketReleasedCounter(p prometheus.Counter) Option {
	return func(c *nfQueueConnector) {
		c.prometheusReleasedCounter = p
	}
}

// OptPacketInNfQueueSummary sets prometheus summary for the length of time a packet was in the NfQueue before
// being seen by userspace.
func OptPacketInNfQueueSummary(p prometheus.Summary) Option {
	return func(c *nfQueueConnector) {
		c.prometheusTimeInQueueSummary = p
	}
}

// OptPacketHoldTimeSummary sets prometheus summary for the length of time a packet was being held in userspace
// before being released.
func OptPacketHoldTimeSummary(p prometheus.Summary) Option {
	return func(c *nfQueueConnector) {
		c.prometheusHoldTimeSummary = p
	}
}

func OptMarkBitsToPreserve(bits uint32) Option {
	return func(c *nfQueueConnector) {
		c.markBitsToPreserve = bits
	}
}
