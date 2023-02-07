// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"context"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// DNSFlowsInterface has methods related to flows.
type DNSFlowsInterface interface {
	List(ctx context.Context, params v1.DNSFlowParams) (v1.List[v1.DNSFlow], error)
}

// DNSFlows implements DNSFlowsInterface.
type dnsFlows struct {
	restClient *rest.RESTClient
}

// newFlows returns a new FlowsInterface bound to the supplied client.
func newDNSFlows(c *client) DNSFlowsInterface {
	return &dnsFlows{restClient: c.restClient}
}

// List gets the flow list for the given flow input params.
func (f *dnsFlows) List(ctx context.Context, params v1.DNSFlowParams) (v1.List[v1.DNSFlow], error) {
	flows := v1.List[v1.DNSFlow]{}
	err := f.restClient.Post().
		Path("/api/v1/flows/dns").
		Params(&params).
		Do(ctx).
		Into(&flows)
	if err != nil {
		return flows, err
	}
	return flows, nil
}
