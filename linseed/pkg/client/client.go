// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

type Client interface {
	L3Flows() L3FlowsInterface
	L7Flows() L7FlowsInterface
	DNSFlows() DNSFlowsInterface
}

type client struct {
	restClient *rest.RESTClient
}

// L3Flows returns an interface for managing v1.L3Flow resources.
func (c *client) L3Flows() L3FlowsInterface {
	return newL3Flows(c)
}

// L7Flows returns an interface for managing v1.L7Flow resources.
func (c *client) L7Flows() L7FlowsInterface {
	return newL7Flows(c)
}

// DNSFlows returns an interface for managing v1.DNSFlow resources.
func (c *client) DNSFlows() DNSFlowsInterface {
	return newDNSFlows(c)
}

func NewClient(clusterID, tenantID string, cfg rest.Config) (Client, error) {
	rc, err := rest.NewClient(clusterID, tenantID, cfg)
	if err != nil {
		return nil, err
	}
	return &client{
		restClient: rc,
	}, nil
}
