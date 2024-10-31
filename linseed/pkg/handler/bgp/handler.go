// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package bgp

import (
	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
)

const (
	LogPath     = "/bgp/logs"
	LogPathBulk = "/bgp/logs/bulk"
)

type bgp struct {
	logs handler.RWHandler[v1.BGPLog, v1.BGPLogParams, v1.BGPLog]
}

func New(b bapi.BGPBackend) *bgp {
	return &bgp{logs: handler.NewRWHandler(b.Create, b.List)}
}

func (h bgp) APIS() []handler.API {
	return []handler.API{
		{
			Method:          "POST",
			URL:             LogPathBulk,
			Handler:         h.logs.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "bgplogs"},
		},
		{
			Method:          "POST",
			URL:             LogPath,
			Handler:         h.logs.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "bgplogs"},
		},
	}
}
