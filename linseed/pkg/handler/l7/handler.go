// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package l7

import (
	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
)

const (
	// URLs
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
		flows: handler.NewReadOnlyHandler(flows.List),
		logs:  handler.NewCompositeHandler(logs.Create, logs.List, logs.Aggregations),
	}
}

func (h l7) APIS() []handler.API {
	return []handler.API{
		{
			Method:          "POST",
			URL:             FlowPath,
			Handler:         h.flows.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "l7flows"},
		},
		{
			Method:          "POST",
			URL:             LogPathBulk,
			Handler:         h.logs.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "l7logs"},
		},
		{
			Method:          "POST",
			URL:             LogPath,
			Handler:         h.logs.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "l7logs"},
		},
		{
			Method:          "POST",
			URL:             AggsPath,
			Handler:         h.logs.Aggregate(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "l7logs"},
		},
	}
}
