// Copyright (c) 2022 Tigera Inc. All rights reserved.

package syncer

import (
	v1 "k8s.io/api/core/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
)

const (
	StagedNetworkPolicyNameSuffix = "recommendation"
)

// The following are Query structs used for the RunQuery Interface from the corresponding
// controller indicated by the Kubernetes Resource in the name.  (ie, NamespaceQuery will
// be called from the  Namespace Controller to synchronize the caches)
type NamespaceQuery struct {
	MetaSelectors
}

type NamespaceQueryResult struct {
	StagedNetworkPolicies []*v3.StagedNetworkPolicy
}

type PolicyRecommendationScopeQuery struct {
	MetaSelectors
}

type PolicyReqScopeQueryResult struct {
	StagedNetworkPolicies []*v3.StagedNetworkPolicy
}

type MetaSelectors struct {
	Source *api.Update
	Labels map[string]string
}

// CacheSet contains all the caches that will be synchronized in the Query Interface.
type CacheSet struct {
	Namespaces            cache.ObjectCache[*v1.Namespace]
	NetworkSets           cache.ObjectCache[*v3.NetworkSet]
	StagedNetworkPolicies cache.ObjectCache[*v3.StagedNetworkPolicy]
}
