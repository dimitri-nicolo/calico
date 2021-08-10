// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package nfqueue

import "time"

type Option func(packetProcessor *DNSPolicyPacketProcessor)

func WithPacketDropTimeout(duration time.Duration) Option {
	return func(packetProcessor *DNSPolicyPacketProcessor) {
		packetProcessor.packetDropTimeout = duration
	}
}

func WithPacketReleaseTimeout(duration time.Duration) Option {
	return func(packetProcessor *DNSPolicyPacketProcessor) {
		packetProcessor.packetReleaseTimeout = duration
	}
}

func WithReleaseTickerTimeout(duration time.Duration) Option {
	return func(packetProcessor *DNSPolicyPacketProcessor) {
		packetProcessor.releaseTickerDuration = duration
	}
}
