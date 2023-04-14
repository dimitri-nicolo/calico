// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7

import (
	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	FlowPath    = "/l7"
	LogPath     = "/l7/logs"
	AggsPath    = "/l7/logs/aggregation"
	LogPathBulk = "/l7/logs/bulk"
)

type l7 struct {
	flows handler.ReadOnlyHandler[v1.L7Flow, v1.L7FlowParams]
	logs  handler.GenericHandler[v1.L7Log, v1.L7LogParams, v1.L7Log, v1.L7AggregationParams]
}

func New(flows bapi.L7FlowBackend, logs bapi.L7LogBackend) handler.Handler {
	return &l7{
		flows: handler.NewReadOnlyHandler[v1.L7Flow, v1.L7FlowParams](flows.List),
		logs: handler.NewCompositeHandler[v1.L7Log, v1.L7LogParams, v1.L7Log, v1.L7AggregationParams](
			logs.Create, logs.List, logs.Aggregations),
	}
}

func (h l7) APIS() []handler.API {
	return []handler.API{
		{
			Method:  "POST",
			URL:     FlowPath,
			Handler: h.flows.List(),
		},
		{
			Method:  "POST",
			URL:     LogPathBulk,
			Handler: h.logs.Create(),
		},
		{
			Method:  "POST",
			URL:     LogPath,
			Handler: h.logs.List(),
		},
		{
			Method:  "POST",
			URL:     AggsPath,
			Handler: h.logs.Aggregate(),
		},
	}
}
