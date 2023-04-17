// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns

import (
	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	FlowPath    = "/dns"
	LogPath     = "/dns/logs"
	AggsPath    = "/dns/logs/aggregation"
	LogPathBulk = "/dns/logs/bulk"
)

type dns struct {
	logs  handler.GenericHandler[v1.DNSLog, v1.DNSLogParams, v1.DNSLog, v1.DNSAggregationParams]
	flows handler.ReadOnlyHandler[v1.DNSFlow, v1.DNSFlowParams]
}

func New(flows bapi.DNSFlowBackend, logs bapi.DNSLogBackend) *dns {
	return &dns{
		flows: handler.NewReadOnlyHandler[v1.DNSFlow, v1.DNSFlowParams](flows.List),
		logs: handler.NewCompositeHandler[v1.DNSLog, v1.DNSLogParams, v1.DNSLog, v1.DNSAggregationParams](
			logs.Create, logs.List, logs.Aggregations),
	}
}

func (h dns) APIS() []handler.API {
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
