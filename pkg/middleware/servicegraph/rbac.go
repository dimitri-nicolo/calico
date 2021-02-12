// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	log "github.com/sirupsen/logrus"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	v1 "github.com/tigera/es-proxy/pkg/apis/v1"
)

// This file implements an RBAC flow filter. It parses the AuthorizedResourceVerbs returned by a authorization
// review to determine which endpoint types are listable. At least one endpoint in a flow should be listable for the
// flow to be included.

type RBACFilter interface {
	IncludeFlow(f FlowEdge) bool
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
	if f.includeEndpoint(e.Source) {
		return true
	}
	if f.includeEndpoint(e.Dest) {
		return true
	}
	return false
}

func (f *rbacFilter) includeEndpoint(e FlowEndpoint) bool {
	// Flow data should only consists of the endpoint types contained in the flow logs, and not any of the generated
	// types for the graph.
	switch e.Type {
	case v1.GraphNodeTypeWorkload, v1.GraphNodeTypeReplicaSet:
		return f.listAllPods || f.listPodNamespaces[e.Namespace]
	case v1.GraphNodeTypeNetwork:
		return false
	case v1.GraphNodeTypeNetworkSet:
		if e.Namespace == "" {
			return f.listAllGlobalNetworkSets
		}
		return f.listAllNetworkSets || f.listNetworkSetNamespaces[e.Namespace]
	case v1.GraphNodeTypeHostEndpoint:
		return f.listAllHostEndpoints
	default:
		log.Panicf("Unexpected endpoint type in parsed flows: %s", e.Type)
	}
	return false
}
