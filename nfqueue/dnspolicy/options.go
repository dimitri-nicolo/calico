// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dnspolicy

import "time"

type Option func(packetProcessor *PacketProcessor)

func WithPacketDropTimeout(duration time.Duration) Option {
	return func(packetProcessor *PacketProcessor) {
		packetProcessor.packetDropTimeout = duration
	}
}

func WithPacketReleaseTimeout(duration time.Duration) Option {
	return func(packetProcessor *PacketProcessor) {
		packetProcessor.packetReleaseTimeout = duration
	}
}

func WithReleaseTickerTimeout(duration time.Duration) Option {
	return func(packetProcessor *PacketProcessor) {
		packetProcessor.releaseTickerDuration = duration
	}
}
