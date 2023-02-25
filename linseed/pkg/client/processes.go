// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// ProcessesInterface has methods related to flows.
type ProcessesInterface interface {
	List(ctx context.Context, params v1.Params) (*v1.List[v1.ProcessInfo], error)
}

// processes implements ProcessesInterface.
type processes struct {
	restClient rest.RESTClient
	clusterID  string
}

// newFlows returns a new FlowsInterface bound to the supplied client.
func newProcesses(c Client, cluster string) ProcessesInterface {
	return &processes{restClient: c.RESTClient(), clusterID: cluster}
}

// List gets the l3 flow list for the given flow input params.
func (f *processes) List(ctx context.Context, params v1.Params) (*v1.List[v1.ProcessInfo], error) {
	flows := v1.List[v1.ProcessInfo]{}
	err := f.restClient.Post().
		Path("/processes").
		Params(params).
		Cluster(f.clusterID).
		Do(ctx).
		Into(&flows)
	if err != nil {
		return nil, err
	}
	return &flows, nil
}
