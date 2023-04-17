// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l3

import (
	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	FlowPath    = "/flows"
	LogPath     = "/flows/logs"
	AggsPath    = "/flows/logs/aggregation"
	LogPathBulk = "/flows/logs/bulk"
)

type Flows struct {
	logs  handler.GenericHandler[v1.FlowLog, v1.FlowLogParams, v1.FlowLog, v1.FlowLogAggregationParams]
	flows handler.ReadOnlyHandler[v1.L3Flow, v1.L3FlowParams]
}

func New(flows bapi.FlowBackend, logs bapi.FlowLogBackend) *Flows {
	return &Flows{
		flows: handler.NewReadOnlyHandler[v1.L3Flow, v1.L3FlowParams](flows.List),
		logs: handler.NewCompositeHandler[v1.FlowLog, v1.FlowLogParams, v1.FlowLog, v1.FlowLogAggregationParams](
			logs.Create, logs.List, logs.Aggregations),
	}
}

func (h Flows) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     FlowPath,
			Handler: h.flows.List(),
		},
		{
			Method:  "POST",
			URL:     LogPath,
			Handler: h.logs.List(),
		},
		{
			Method:  "POST",
			URL:     LogPathBulk,
			Handler: h.logs.Create(),
		},
		{
			Method:  "POST",
			URL:     AggsPath,
			Handler: h.logs.Aggregate(),
		},
	}
}
