// Copyright (c) 2023 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"
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
		return nil, fmt.Errorf("error getting new http client %s", err)
	}

	return &client{
		restClient: rc,
	}, nil
}
