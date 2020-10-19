// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package processor

import (
	"context"
	"time"

	"github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	Index         = "tigera_secure_ee_events.cluster"
	PacketCapture = "capture-honey"
	PcapPath      = "/pcap"
	SnortPath     = "/snort"
)

type HoneyPodLogProcessor struct {
	Ctx                context.Context
	LogHandler         api.AlertLogReportHandler
	Client             elastic.Client
	LastProcessingTime time.Time
}

func NewHoneyPodLogProcessor(c elastic.Client, ctx context.Context) *HoneyPodLogProcessor {
	return &HoneyPodLogProcessor{
		LogHandler: c, Ctx: ctx, Client: c, LastProcessingTime: time.Now().Add(-10 * time.Minute),
	}
}