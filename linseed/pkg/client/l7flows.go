// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// L7FlowsInterface has methods related to flows.
type L7FlowsInterface interface {
	List(ctx context.Context, params v1.Params) (*v1.List[v1.L7Flow], error)
}

// L7Flows implements L7FlowsInterface.
type l7Flows struct {
	restClient *rest.RESTClient
	clusterID  string
}

// newFlows returns a new FlowsInterface bound to the supplied client.
func newL7Flows(c *client, cluster string) L7FlowsInterface {
	return &l7Flows{restClient: c.restClient, clusterID: cluster}
}

// List gets the l3 flow list for the given flow input params.
func (f *l7Flows) List(ctx context.Context, params v1.Params) (*v1.List[v1.L7Flow], error) {
	flows := v1.List[v1.L7Flow]{}
	err := f.restClient.Post().
		Path("/api/v1/flows/l7").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&flows)
	if err != nil {
		return &flows, err
	}
	return &flows, nil
}
