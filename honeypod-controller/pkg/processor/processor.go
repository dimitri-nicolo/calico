// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package processor

import (
	"context"
	"time"

	"github.com/projectcalico/calico/lma/pkg/elastic"
)

const (
	PacketCapture = "capture-honey"
	PcapPath      = "/pcap"
	SnortPath     = "/snort"
)

type HoneyPodLogProcessor struct {
	Ctx                context.Context
	Client             elastic.Client
	LastProcessingTime time.Time
}

func NewHoneyPodLogProcessor(c elastic.Client, ctx context.Context) *HoneyPodLogProcessor {
	return &HoneyPodLogProcessor{
		Ctx:                ctx,
		Client:             c,
		LastProcessingTime: time.Now().Add(-10 * time.Minute),
	}
}
