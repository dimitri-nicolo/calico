/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpoint

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

const (
	endpointDelim = "."
)

// legacyREST is storage for tiered policies based on libalico-go.
type legacyREST struct {
	store  *genericregistry.Store
	client *client.Client
}

func NewLegacyREST(s *genericregistry.Store) *legacyREST {
	var err error

	cfg, err := client.LoadClientConfig("")
	if err != nil {
		glog.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}

	c, err := client.New(*cfg)
	if err != nil {
		glog.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	glog.Infof("Client: %v", c)

	return &legacyREST{s, c}
}

type endpointLabels struct {
	nodeName     string
	orchestrator string
	namespace    string
	name         string
	iface        string
}

func getEndpointLabels(epName string) (*endpointLabels, error) {
	epSlice := strings.Split(epName, endpointDelim)
	if len(epSlice) < 5 {
		return nil, fmt.Errorf("Endpoint name not formatted")
	}
	return &endpointLabels{
		epSlice[0],
		epSlice[1],
		epSlice[2],
		epSlice[3],
		epSlice[4],
	}, nil
}

func (l *legacyREST) get(name string) (*api.HostEndpointList, error) {
	epLabels, err := getEndpointLabels(name)
	if err != nil {
		return nil, err
	}
	endpointMD := api.HostEndpointMetadata{}
	endpointMD.Name = epLabels.name
	endpointMD.Node = epLabels.nodeName

	eHandler := l.client.HostEndpoints()
	endpoints, err := eHandler.List(endpointMD)
	if err != nil {
		return nil, err
	}
	if len(endpoints.Items) < 1 {
		return nil, fmt.Errorf("endpoints %s not found", name)
	}
	return endpoints, nil
}
