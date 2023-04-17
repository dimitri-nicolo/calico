// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf

import (
	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	LogPath     = "/waf/logs"
	AggsPath    = "/waf/logs/aggregation"
	LogPathBulk = "/waf/logs/bulk"
)

type waf struct {
	logs handler.GenericHandler[v1.WAFLog, v1.WAFLogParams, v1.WAFLog, v1.WAFLogAggregationParams]
}

func New(b bapi.WAFBackend) *waf {
	return &waf{
		logs: handler.NewCompositeHandler[v1.WAFLog, v1.WAFLogParams, v1.WAFLog, v1.WAFLogAggregationParams](
			b.Create, b.List, b.Aggregations),
	}
}

func (h waf) APIS() []handler.API {
	return []handler.API{
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
