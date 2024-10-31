// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package dns

import (
	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
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
		flows: handler.NewReadOnlyHandler(flows.List),
		logs:  handler.NewCompositeHandler(logs.Create, logs.List, logs.Aggregations),
	}
}

func (h dns) APIS() []handler.API {
	return []handler.API{
		{
			Method:          "POST",
			URL:             FlowPath,
			Handler:         h.flows.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "dnsflows"},
		},
		{
			Method:          "POST",
			URL:             LogPathBulk,
			Handler:         h.logs.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: "dnslogs"},
		},
		{
			Method:          "POST",
			URL:             LogPath,
			Handler:         h.logs.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "dnslogs"},
		},
		{
			Method:          "POST",
			URL:             AggsPath,
			Handler:         h.logs.Aggregate(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: "dnslogs"},
		},
	}
}
