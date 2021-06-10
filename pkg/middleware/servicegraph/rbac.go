// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"context"
	"net/http"

	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/k8s"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file implements an RBAC flow filter. It parses the AuthorizedResourceVerbs returned by a authorization
// review to determine which endpoint types are listable. At least one endpoint in a flow should be listable for the
// flow to be included.

type RBACFilter interface {
	IncludeFlow(f FlowEdge) bool
	IncludeEndpoint(f FlowEndpoint) bool
	IncludeHostEndpoints() bool
	IncludeGlobalNetworkSets() bool
	IncludeNetworkSets(namespace string) bool
	IncludePods(namespace string) bool
}

// NewRBACFilter performs an authorization review and uses the response to construct an RBAC filter.
func NewRBACFilter(
	ctx context.Context, csFactory k8s.ClientSetFactory, req *http.Request, cluster string,
) (RBACFilter, error) {
	verbs, err := auth.PerformUserAuthorizationReviewForElasticLogs(ctx, csFactory, req, cluster)
	if err != nil {
		return nil, err
	}
	return NewRBACFilterFromAuth(verbs), nil
}

// NewRBACFilterFromAuth creates a new RBAC filter from a set of AuthorizedResourceVerbs.
func NewRBACFilterFromAuth(verbs []v3.AuthorizedResourceVerbs) RBACFilter {
	f := &rbacFilter{
		listPodNamespaces:        make(map[string]bool),
		listNetworkSetNamespaces: make(map[string]bool),
	}

	for _, r := range verbs {
		for _, v := range r.Verbs {
			if v.Verb != "list" {
				// Only interested in the list verbs.
				continue
			}
			for _, rg := range v.ResourceGroups {
				switch r.Resource {
				case "hostendpoints":
					f.listAllHostEndpoints = true
				case "networksets":
					if rg.Namespace == "" {
						f.listAllNetworkSets = true
					} else {
						f.listNetworkSetNamespaces[rg.Namespace] = true
					}
				case "globalnetworksets":
					f.listAllGlobalNetworkSets = true
				case "pods":
					if rg.Namespace == "" {
						f.listAllPods = true
					} else {
						f.listPodNamespaces[rg.Namespace] = true
					}
				}
			}
		}
	}

	return f
}

// rbacFilter implements the RBACFilter interface.
type rbacFilter struct {
	listAllPods              bool
	listAllHostEndpoints     bool
	listAllGlobalNetworkSets bool
	listAllNetworkSets       bool
	listPodNamespaces        map[string]bool
	listNetworkSetNamespaces map[string]bool
}

func (f *rbacFilter) IncludeFlow(e FlowEdge) bool {
	if f.IncludeEndpoint(e.Source) {
		return true
	}
	if f.IncludeEndpoint(e.Dest) {
		return true
	}
	return false
}

func (f *rbacFilter) IncludeEndpoint(e FlowEndpoint) bool {
	// L3Flow data should only consists of the endpoint types contained in the flow logs, and not any of the generated
	// types for the graph.
	switch e.Type {
	case v1.GraphNodeTypeWorkload, v1.GraphNodeTypeReplicaSet:
		return f.IncludePods(e.Namespace)
	case v1.GraphNodeTypeNetwork:
		return false
	case v1.GraphNodeTypeNetworkSet:
		if e.Namespace == "" {
			return f.IncludeGlobalNetworkSets()
		}
		return f.IncludeNetworkSets(e.Namespace)
	case v1.GraphNodeTypeHost:
		return f.IncludeHostEndpoints()
	default:
		log.Panicf("Unexpected endpoint type in parsed flows: %s", e.Type)
	}
	return false
}

func (f *rbacFilter) IncludeHostEndpoints() bool {
	return f.listAllHostEndpoints
}

func (f *rbacFilter) IncludeGlobalNetworkSets() bool {
	return f.listAllGlobalNetworkSets
}

func (f *rbacFilter) IncludeNetworkSets(namespace string) bool {
	return f.listAllNetworkSets || f.listNetworkSetNamespaces[namespace]
}

func (f *rbacFilter) IncludePods(namespace string) bool {
	return f.listAllPods || f.listPodNamespaces[namespace]
}

// ---- Mock filters for testing ----
type RBACFilterIncludeAll struct{}

func (m RBACFilterIncludeAll) IncludeFlow(f FlowEdge) bool              { return true }
func (m RBACFilterIncludeAll) IncludeEndpoint(f FlowEndpoint) bool      { return true }
func (m RBACFilterIncludeAll) IncludeHostEndpoints() bool               { return true }
func (m RBACFilterIncludeAll) IncludeGlobalNetworkSets() bool           { return true }
func (m RBACFilterIncludeAll) IncludeNetworkSets(namespace string) bool { return true }
func (m RBACFilterIncludeAll) IncludePods(namespace string) bool        { return true }

type RBACFilterIncludeNone struct{}

func (m RBACFilterIncludeNone) IncludeFlow(f FlowEdge) bool              { return false }
func (m RBACFilterIncludeNone) IncludeEndpoint(f FlowEndpoint) bool      { return false }
func (m RBACFilterIncludeNone) IncludeHostEndpoints() bool               { return false }
func (m RBACFilterIncludeNone) IncludeGlobalNetworkSets() bool           { return false }
func (m RBACFilterIncludeNone) IncludeNetworkSets(namespace string) bool { return false }
func (m RBACFilterIncludeNone) IncludePods(namespace string) bool        { return false }
