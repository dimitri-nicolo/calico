// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package processor

import (
	"context"
	"time"

	api "github.com/tigera/lma/pkg/api"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	Index         = "tigera_secure_ee_events.cluster"
	PacketCapture = "capture-honey"
	PcapPath      = "/pcap"
	SnortPath     = "/snort"
)

type HoneypodLogProcessor struct {
	Ctx                context.Context
	LogHandler         api.AlertLogReportHandler
	Client             elastic.Client
	LastProcessingTime time.Time
}

func NewHoneypodLogProcessor(c elastic.Client, ctx context.Context) (HoneypodLogProcessor, error) {
	return HoneypodLogProcessor{LogHandler: c, Ctx: ctx, Client: c, LastProcessingTime: time.Now().Add(-10 * time.Minute)}, nil
}

func (p *HoneypodLogProcessor) UpdateLastProcessTime(newtime time.Time) {
	p.LastProcessingTime = newtime
}
