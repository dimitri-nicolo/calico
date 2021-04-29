// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.
package vfp

import (
	"time"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/collector"
)

// InfoReader implements collector.PacketInfoReader and collector.ConntrackInfoReader.
type InfoReader struct{}

func NewInfoReader(lookupsCache *calc.LookupsCache, period time.Duration) *InfoReader {
	return &InfoReader{}
}

func (r *InfoReader) Start() error {
	return nil
}

func (r *InfoReader) Stop() {
	return
}

// PacketInfoChan returns the channel with converted PacketInfo.
func (r *InfoReader) PacketInfoChan() <-chan collector.PacketInfo {
	return nil
}

// ConntrackInfoChan returns the channel with converted ConntrackInfo.
func (r *InfoReader) ConntrackInfoChan() <-chan []collector.ConntrackInfo {
	return nil
}

// Endpoint
func (r *InfoReader) EndpointEventHandler() *InfoReader {
	return r
}

// Cache endpoint updates.
func (r *InfoReader) HandleEndpointsUpdate(ids []string) {
	return
}

// Cache policy updates.
func (r *InfoReader) HandlePolicyUpdate(id string) {
	return
}
