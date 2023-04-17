// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package bgp

import (
	"github.com/projectcalico/calico/linseed/pkg/handler"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

const (
	LogPath     = "/bgp/logs"
	LogPathBulk = "/bgp/logs/bulk"
)

type bgp struct {
	logs handler.RWHandler[v1.BGPLog, v1.BGPLogParams, v1.BGPLog]
}

func New(b bapi.BGPBackend) *bgp {
	return &bgp{
		logs: handler.NewRWHandler[v1.BGPLog, v1.BGPLogParams, v1.BGPLog](b.Create, b.List),
	}
}

func (h bgp) APIS() []handler.API {
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
	}
}
