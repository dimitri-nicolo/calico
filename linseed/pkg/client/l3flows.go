// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// L3FlowsInterface has methods related to flows.
type L3FlowsInterface interface {
	List(ctx context.Context, params v1.L3FlowParams) (v1.List[v1.L3Flow], error)
}

// L3Flows implements L3FlowsInterface.
type l3Flows struct {
	restClient *rest.RESTClient
	clusterID  string
}

// newFlows returns a new FlowsInterface bound to the supplied client.
func newL3Flows(c *client, cluster string) L3FlowsInterface {
	return &l3Flows{restClient: c.restClient, clusterID: cluster}
}

// List gets the l3 flow list for the given flow input params.
func (f *l3Flows) List(ctx context.Context, flowParams v1.L3FlowParams) (v1.List[v1.L3Flow], error) {
	flows := v1.List[v1.L3Flow]{}
	err := f.restClient.Post().
		Path("/api/v1/flows/network").
		Params(&flowParams).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&flows)
	if err != nil {
		return flows, err
	}
	return flows, nil
}
