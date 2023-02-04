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
	"context"
	"fmt"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// L3FlowsInterface has methods related to flows.
type L3FlowsInterface interface {
	List(params v1.L3FlowParams) ([]v1.L3Flow, error)
}

// L3Flows implements L3FlowsInterface.
type l3Flows struct {
	restClient *rest.RESTClient
}

// newFlows returns a new FlowsInterface bound to the supplied client.
func newL3Flows(c *client) L3FlowsInterface {
	return &l3Flows{restClient: c.restClient}
}

// List get the l3 flow list for the given flow input params.
func (f *l3Flows) List(flowParams v1.L3FlowParams) ([]v1.L3Flow, error) {

	flows := v1.List[v1.L3Flow]{}
	rc := f.restClient
	err := rc.Post().
		Path("/api/v1/flows/network").
		Params(&flowParams).
		Do(context.TODO()).
		Into(&flows)
	if err != nil {
		return nil, fmt.Errorf("error fetching data from linseed rest API: %s", err)
	}
	return flows.Items, nil

}
