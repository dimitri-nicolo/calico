// Copyright (c) 2018-2024 Tigera, Inc. All rights reserved.
package client

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
)

// TODO (rlb):  These data types are basically focussed on the requirements of the web server
// and calicoq.  However this means we have already selected what data we want to return to
// the client application.  This feels wrong.  We should probably just return a full copy of
// the data associated with each resource and let the app display it however it wants. Not
// worrying about this for now, but may prove important for AAPIS integration.

// QueryInterface is the (very generic) interface used to perform simple synchronous queries
// against the cached data.  It takes one of the Query*Req structures as the query request
// and returns the corresponding Query*Resp structure, or an error.
type QueryInterface interface {
	RunQuery(ctx context.Context, req interface{}) (interface{}, error)
}

type QueryClusterReq struct {
	// timestamp for the historical summary data point
	Timestamp *time.Time
	// prometheus endpoint to retrieve historical data
	PrometheusEndpoint string
	Token              string
}

type QueryClusterResp struct {
	NumGlobalNetworkPolicies          int                                    `json:"numGlobalNetworkPolicies"`
	NumNetworkPolicies                int                                    `json:"numNetworkPolicies"`
	NumHostEndpoints                  int                                    `json:"numHostEndpoints"`
	NumWorkloadEndpoints              int                                    `json:"numWorkloadEndpoints"`
	NumUnmatchedGlobalNetworkPolicies int                                    `json:"numUnmatchedGlobalNetworkPolicies"`
	NumUnmatchedNetworkPolicies       int                                    `json:"numUnmatchedNetworkPolicies"`
	NumUnlabelledHostEndpoints        int                                    `json:"numUnlabelledHostEndpoints"`
	NumUnlabelledWorkloadEndpoints    int                                    `json:"numUnlabelledWorkloadEndpoints"`
	NumUnprotectedHostEndpoints       int                                    `json:"numUnprotectedHostEndpoints"`
	NumUnprotectedWorkloadEndpoints   int                                    `json:"numUnprotectedWorkloadEndpoints"`
	NumFailedWorkloadEndpoints        int                                    `json:"numFailedWorkloadEndpoints"`
	NumNodes                          int                                    `json:"numNodes"`
	NumNodesWithNoEndpoints           int                                    `json:"numNodesWithNoEndpoints"`
	NumNodesWithNoHostEndpoints       int                                    `json:"numNodesWithNoHostEndpoints"`
	NumNodesWithNoWorkloadEndpoints   int                                    `json:"numNodesWithNoWorkloadEndpoints"`
	NamespaceCounts                   map[string]QueryClusterNamespaceCounts `json:"namespaceCounts"`
}

type QueryClusterNamespaceCounts struct {
	NumNetworkPolicies              int `json:"numNetworkPolicies"`
	NumWorkloadEndpoints            int `json:"numWorkloadEndpoints"`
	NumUnmatchedNetworkPolicies     int `json:"numUnmatchedNetworkPolicies"`
	NumUnlabelledWorkloadEndpoints  int `json:"numUnlabelledWorkloadEndpoints"`
	NumUnprotectedWorkloadEndpoints int `json:"numUnprotectedWorkloadEndpoints"`
	NumFailedWorkloadEndpoints      int `json:"numFailedWorkloadEndpoints"`
}

type QueryNodesReq struct {
	// Queries
	Node model.Key

	// Filters
	Page *Page
	Sort *Sort
}

type QueryNodesResp struct {
	Count int    `json:"count"`
	Items []Node `json:"items"`
}

type Node struct {
	Name                 string   `json:"name"`
	BGPIPAddresses       []string `json:"bgpIPAddresses"`
	Addresses            []string `json:"addressses"`
	NumHostEndpoints     int      `json:"numHostEndpoints"`
	NumWorkloadEndpoints int      `json:"numWorkloadEndpoints"`
}

type QueryPoliciesReq struct {
	// Queries (select one)
	Endpoint   model.Key
	Labels     map[string]string
	Policy     model.Key
	NetworkSet model.Key

	// Filters
	Unmatched bool
	Tier      []string
	Page      *Page
	Sort      *Sort
}

type QueryPoliciesResp struct {
	Count int      `json:"count"`
	Items []Policy `json:"items"`
}

type Policy struct {
	Index                int               `json:"index"`
	Kind                 string            `json:"kind"`
	Name                 string            `json:"name"`
	Namespace            string            `json:"namespace,omitempty"`
	Tier                 string            `json:"tier"`
	Annotations          map[string]string `json:"annotations"`
	NumHostEndpoints     int               `json:"numHostEndpoints"`
	NumWorkloadEndpoints int               `json:"numWorkloadEndpoints"`
	Ingress              []RuleDirection   `json:"ingressRules"`
	Egress               []RuleDirection   `json:"egressRules"`
	Order                *float64          `json:"order"`
	CreationTime         *v1.Time          `json:"creation-time"`
}

type RuleDirection struct {
	Source      RuleEntity `json:"source"`
	Destination RuleEntity `json:"destination"`
}

type RuleEntity struct {
	NumHostEndpoints     int `json:"numHostEndpoints"`
	NumWorkloadEndpoints int `json:"numWorkloadEndpoints"`
}

// QueryEndpointsReqBody is used to UnMarshal endpoints request body.
type QueryEndpointsReqBody struct {
	// Queries
	Policy              []string `json:"policy,omitempty" validate:"omitempty"`
	RuleDirection       string   `json:"ruleDirection,omitempty" validate:"omitempty"`
	RuleIndex           int      `json:"ruleIndex,omitempty" validate:"omitempty"`
	RuleEntity          string   `json:"ruleEntity,omitempty" validate:"omitempty"`
	RuleNegatedSelector bool     `json:"ruleNegatedSelector,omitempty" validate:"omitempty"`
	Selector            string   `json:"selector,omitempty" validate:"omitempty"`
	Endpoint            string   `json:"endpoint,omitempty" validate:"omitempty"`
	Unprotected         bool     `json:"unprotected,omitempty" validate:"omitempty"`

	// Filters
	EndpointsList []string `json:"endpointsList"` // we need to identify when this field is passed as empty list or is not passed
	Node          string   `json:"node,omitempty" validate:"omitempty"`
	Namespace     string   `json:"namespace,omitempty" validate:"omitempty"`
	Unlabelled    bool     `json:"unlabelled,omitempty"  validate:"omitempty"`
	Page          *Page    `json:"page,omitempty" validate:"omitempty"`
	Sort          *Sort    `json:"sort,omitempty" validate:"omitempty"`
}

// QueryEndpointsReq is the internal struct. Endpoints request.body --> QueryEndpointsReqBody --> QueryEndpointReq
// if any member is added / removed / changed in this struct, also update:
// 1. QueryEndpointsRequestBody struct defined above
// 2. getQueryServerRequestParams in es-proxy/pkg/middleware/endpoints_aggregation.go as needed.
type QueryEndpointsReq struct {
	// Queries
	Policy              model.Key
	RuleDirection       string
	RuleIndex           int
	RuleEntity          string
	RuleNegatedSelector bool
	Selector            string
	Endpoint            model.Key
	Unprotected         bool

	// Filters
	EndpointsList []string
	Node          string
	Namespace     string
	Unlabelled    bool
	Page          *Page
	Sort          *Sort
}

const (
	RuleDirectionIngress  = "ingress"
	RuleDirectionEgress   = "egress"
	RuleEntitySource      = "source"
	RuleEntityDestination = "destination"
)

type QueryEndpointsResp struct {
	Count int        `json:"count"`
	Items []Endpoint `json:"items"`
}

type EndpointCount struct {
	NumHostEndpoints     int `json:"numHostEndpoints"`
	NumWorkloadEndpoints int `json:"numWorkloadEndpoints"`
}

type PolicyCount struct {
	NumGlobalNetworkPolicies int `json:"numGlobalNetworkPolicies"`
	NumNetworkPolicies       int `json:"numNetworkPolicies"`
}

type Endpoint struct {
	Kind                     string            `json:"kind"`
	Name                     string            `json:"name"`
	Namespace                string            `json:"namespace,omitempty"`
	Node                     string            `json:"node"`
	Workload                 string            `json:"workload"`
	Orchestrator             string            `json:"orchestrator"`
	Pod                      string            `json:"pod"`
	InterfaceName            string            `json:"interfaceName"`
	IPNetworks               []string          `json:"ipNetworks"`
	Labels                   map[string]string `json:"labels"`
	NumGlobalNetworkPolicies int               `json:"numGlobalNetworkPolicies"`
	NumNetworkPolicies       int               `json:"numNetworkPolicies"`
}

type Page struct {
	PageNum    int `json:"pageNum,omitempty" validate:"gte=0,omitempty"`
	NumPerPage int `json:"numPerPage,omitempty" validate:"gt=0,omitempty"`
}

type Sort struct {
	SortBy  []string `json:"sortBy,omitempty" validate:"omitempty"`
	Reverse bool     `json:"reverse,omitempty" validate:"omitempty"`
}
