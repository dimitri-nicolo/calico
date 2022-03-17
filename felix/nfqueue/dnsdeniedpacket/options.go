// Copyright (c) 2021-2022 Tigera, Inc. All rights reserved.

package dnsdeniedpacket

import (
	"time"

	"github.com/projectcalico/calico/felix/timeshim"
)

type Option func(packetProcessor *packetProcessor)

func WithPacketReleaseTimeout(duration time.Duration) Option {
	return func(packetProcessor *packetProcessor) {
		packetProcessor.packetReleaseTimeout = duration
	}
}

func WithReleaseTickerDuration(duration time.Duration) Option {
	return func(packetProcessor *packetProcessor) {
		packetProcessor.releaseTickerDuration = duration
	}
}

func WithIPCacheDuration(duration time.Duration) Option {
	return func(packetProcessor *packetProcessor) {
		packetProcessor.ipCacheDuration = duration
	}
}

func WithTimeInterface(timeInf timeshim.Interface) Option {
	return func(packetProcessor *packetProcessor) {
		packetProcessor.time = timeInf
	}
}
