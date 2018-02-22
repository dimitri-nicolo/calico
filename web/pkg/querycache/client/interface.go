// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package client

import (
	"context"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
)

//TODO (rlb):  These data types are basically focussed on the requirements of the web server
// and calicoq.  However this means we have already selected what data we want to return to
// the client application.  This feels wrong.  We should probably just return a full copy of
// the data associated with each resource and let the app display it however it wants. Not
// worrying about this for now, but may prove important for AAPIS integration.

// QueryInterface is the (very generic) interface used to perform simple synchronous queries
// against the cached data.  It takes one of the Query*Req structures as the query request
// and returns the corresponding Query*Resp structure, or an error.
type QueryInterface interface {
	RunQuery(context.Context, interface{}) (interface{}, error)
}

type QueryClusterReq struct {
}

type QueryClusterResp struct {
	NumGlobalNetworkPolicies        int `json:"numGlobalNetworkPolicies"`
	NumNetworkPolicies              int `json:"numNetworkPolicies"`
	NumHostEndpoints                int `json:"numHostEndpoints"`
	NumWorkloadEndpoints            int `json:"numWorkloadEndpoints"`
	NumUnlabelledWorkloadEndpoints  int `json:"numUnlabelledWorkloadEndpoints"`
	NumUnlabelledHostEndpoints      int `json:"numUnlabelledHostEndpoints"`
	NumNodes                        int `json:"numNodes"`
	NumNodesWithNoEndpoints         int `json:"numNodesWithNoEndpoints"`
	NumNodesWithNoWorkloadEndpoints int `json:"numNodesWithNoWorkloadEndpoints"`
	NumNodesWithNoHostEndpoints     int `json:"numNodesWithNoHostEndpoints"`
}

type QueryNodesReq struct {
	// Filters
	Page *Page
}

type QueryNodesResp struct {
	Count int    `json:"count"`
	Items []Node `json:"items"`
}

type Node struct {
	Name                 string   `json:"name"`
	BGPIPAddresses       []string `json:"bgpIPAddresses"`
	NumWorkloadEndpoints int      `json:"numWorkloadEndpoints"`
	NumHostEndpoints     int      `json:"numHostEndpoints"`
}

type QueryPoliciesReq struct {
	// Queries (select one)
	Endpoint  model.Key
	Unmatched bool
	Labels    map[string]string
	Policies  []model.Key //TODO

	// Filters
	Page      *Page
	Tier      string
}

type QueryPoliciesResp struct {
	Count int      `json:"count"`
	Items []Policy `json:"items"`
}

type Policy struct {
	Kind                 string          `json:"kind"`
	Name                 string          `json:"name"`
	Namespace            string          `json:"namespace,omitempty"`
	Tier                 string          `json:"tier"`
	Ingress              []RuleDirection `json:"ingress"`
	Egress               []RuleDirection `json:"egress"`
	NumWorkloadEndpoints int             `json:"numWorkloadEndpoints"`
	NumHostEndpoints     int             `json:"numHostEndpoints"`
}

type RuleDirection struct {
	Source      RuleEntity `json:"source"`
	Destination RuleEntity `json:"destination"`
}

type RuleEntity struct {
	Selector    RuleEntityEndpoints `json:"selector"`
	NotSelector RuleEntityEndpoints `json:"notSelector"`
}

type RuleEntityEndpoints struct {
	NumWorkloadEndpoints int `json:"numWorkloadEndpoints"`
	NumHostEndpoints     int `json:"numHostEndpoints"`
}

type QueryEndpointsReq struct {
	// Queries
	Policy              model.Key
	RuleDirection       string
	RuleIndex           int
	RuleEntity          string
	RuleNegatedSelector bool
	Selector            string
	Endpoints           []model.Key  // TODO

	// Filters
	Node                string   // TODO
	Page                *Page
}

const (
	RuleDirectionIngress = "ingress"
	RuleDirectionEgress = "egress"
	RuleEntitySource = "source"
	RuleEntityDestination = "destination"
)

type QueryEndpointsResp struct {
	Count int        `json:"count"`
	Items []Endpoint `json:"items"`
}

type EndpointCount struct {
	NumWorkloadEndpoints int `json:"numWorkloadEndpoints"`
	NumHostEndpoints     int `json:"numHostEndpoints"`
}

type PolicyCount struct {
	NumGlobalNetworkPolicies int `json:"numGlobalNetworkPolicies"`
	NumNetworkPolicies       int `json:"numNetworkPolicies"`
}

type Endpoint struct {
	Name                     string `json:"name"`
	Namespace                string `json:"namespace,omitempty"`
	Kind                     string `json:"kind"`
	NumGlobalNetworkPolicies int    `json:"numGlobalNetworkPolicies"`
	NumNetworkPolicies       int    `json:"numNetworkPolicies"`
}

type Page struct {
	PageNum    int
	NumPerPage int
}
