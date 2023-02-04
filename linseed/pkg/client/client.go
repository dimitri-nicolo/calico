// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package client

import (
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

type Client interface {
	L3Flows() L3FlowsInterface
}

type client struct {
	restClient *rest.RESTClient
}

// Flows returns an interface for managing l3flow resources.
func (c *client) L3Flows() L3FlowsInterface {
	return newL3Flows(c)
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
