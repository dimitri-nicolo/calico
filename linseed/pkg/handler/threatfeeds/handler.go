// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package threatfeeds

import (
	"fmt"

	authzv1 "k8s.io/api/authorization/v1"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/handler"
)

const (
	FeedsPatternPath = "/threatfeeds/%s"
	BulkPatternPath  = "/threatfeeds/%s/bulk"
	IPSet            = "ipset"
	DomainNameSet    = "domainnameset"
)

var (
	ipSet         = fmt.Sprintf("threatfeeds_%s", IPSet)
	domainNameSet = fmt.Sprintf("threatfeeds_%s", DomainNameSet)
)

type threatFeeds struct {
	ipSet     handler.RWDHandler[v1.IPSetThreatFeed, v1.IPSetThreatFeedParams, v1.IPSetThreatFeed]
	domainSet handler.RWDHandler[v1.DomainNameSetThreatFeed, v1.DomainNameSetThreatFeedParams, v1.DomainNameSetThreatFeed]
}

func New(ib bapi.IPSetBackend, db bapi.DomainNameSetBackend) *threatFeeds {
	return &threatFeeds{
		ipSet:     handler.NewRWDHandler[v1.IPSetThreatFeed, v1.IPSetThreatFeedParams, v1.IPSetThreatFeed](ib.Create, ib.List, ib.Delete),
		domainSet: handler.NewRWDHandler[v1.DomainNameSetThreatFeed, v1.DomainNameSetThreatFeedParams, v1.DomainNameSetThreatFeed](db.Create, db.List, db.Delete),
	}
}

func (h threatFeeds) APIS() []handler.API {

	return []handler.API{
		// IP set threat feeds
		{
			Method:          "POST",
			URL:             fmt.Sprintf(FeedsPatternPath, IPSet),
			Handler:         h.ipSet.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: ipSet},
		},
		{
			Method:          "POST",
			URL:             fmt.Sprintf(BulkPatternPath, IPSet),
			Handler:         h.ipSet.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: ipSet},
		},
		{
			Method:          "DELETE",
			URL:             fmt.Sprintf(BulkPatternPath, IPSet),
			Handler:         h.ipSet.Delete(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Delete, Group: handler.APIGroup, Resource: ipSet},
		},
		// Domain name set threat feeds
		{
			Method:          "POST",
			URL:             fmt.Sprintf(FeedsPatternPath, DomainNameSet),
			Handler:         h.domainSet.List(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Get, Group: handler.APIGroup, Resource: domainNameSet},
		},
		{
			Method:          "POST",
			URL:             fmt.Sprintf(BulkPatternPath, DomainNameSet),
			Handler:         h.domainSet.Create(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Create, Group: handler.APIGroup, Resource: domainNameSet},
		},
		{
			Method:          "DELETE",
			URL:             fmt.Sprintf(BulkPatternPath, DomainNameSet),
			Handler:         h.domainSet.Delete(),
			AuthzAttributes: &authzv1.ResourceAttributes{Verb: handler.Delete, Group: handler.APIGroup, Resource: domainNameSet},
		},
	}
}
