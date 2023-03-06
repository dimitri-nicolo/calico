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
