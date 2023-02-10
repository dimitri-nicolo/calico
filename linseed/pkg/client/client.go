// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

type Client interface {
	L3Flows(string) L3FlowsInterface
	L7Flows(string) L7FlowsInterface
	DNSFlows(string) DNSFlowsInterface
}

type client struct {
	restClient *rest.RESTClient
}

// L3Flows returns an interface for managing v1.L3Flow resources.
func (c *client) L3Flows(cluster string) L3FlowsInterface {
	return newL3Flows(c, cluster)
}

// L7Flows returns an interface for managing v1.L7Flow resources.
func (c *client) L7Flows(cluster string) L7FlowsInterface {
	return newL7Flows(c, cluster)
}

// DNSFlows returns an interface for managing v1.DNSFlow resources.
func (c *client) DNSFlows(cluster string) DNSFlowsInterface {
	return newDNSFlows(c, cluster)
}

func NewClient(tenantID string, cfg rest.Config) (Client, error) {
	rc, err := rest.NewClient(tenantID, cfg)
	if err != nil {
		return nil, err
	}
	return &client{
		restClient: rc,
	}, nil
}
