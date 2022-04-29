// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package events

import (
	"github.com/projectcalico/calico/lma/pkg/api"
)

type HoneypodEventInterface interface {
	EventData() api.EventsData
}

type HoneypodEvent struct {
	api.EventsData
}

func (he HoneypodEvent) EventData() api.EventsData {
	return he.EventsData
}

type HoneypodAlertRecord struct {
	Count       *int64  `json:"count,omitempty"`
	HostKeyword *string `json:"host.keyword,omitempty"`
}

type HoneypodSnortEventRecord struct {
	Snort *Snort `json:"snort,omitempty"`
}

type Snort struct {
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Flags       string `json:"flags,omitempty"`
	Occurrence  string `json:"occurrence,omitempty"`
	Other       string `json:"other,omitempty"`
}
