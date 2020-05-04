// Copyright (c) 2016-2020 Tigera, Inc. All rights reserved.

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

// TODO move the WorkloadEndpoint converters to is own package. Some refactoring of the annotation and label constants
// is necessary to avoid circular imports, which is why this has been deferred.
package conversion

import (
	"net"
	"os"

	kapiv1 "k8s.io/api/core/v1"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
)

// PodInterface contains the network configuration for a particular interface on a Pod.
type PodInterface struct {
	IsDefault          bool
	NetworkName        string
	NetworkNamespace   string // The k8s Namespace the network definition is in
	HostSideIfaceName  string
	InsidePodIfaceName string
	InsidePodGW        net.IP
	IPNets             []*cnet.IPNet
}

type WorkloadEndpointConverter interface {
	// Deprecated: Use InterfacesForPod. This exists for compatibility with OS
	VethNameForWorkload(namespace, podName string) string
	// InterfacesForPod must return the PodInterface for the default interface as it's first entry
	InterfacesForPod(pod *kapiv1.Pod) ([]*PodInterface, error)
	PodToWorkloadEndpoints(pod *kapiv1.Pod) ([]*model.KVPair, error)
}

// NewWorkloadEndpointConverter creates a WorkloadEndpointConverter. Which converter is created depends on the value returned
// by the MultiInterfaceMode function. This defaults to the defaultWorkloadEndpointConverter.
func NewWorkloadEndpointConverter() WorkloadEndpointConverter {
	switch MultiInterfaceMode() {
	case "multus":
		return newMultusWorkloadEndpointConverter()
	default:
		return &defaultWorkloadEndpointConverter{}
	}
}

// MultiInterfaceMode retrieves the multi interface mode set by the user (by reading the MULTI_INTERFACE_MODE env variable).
func MultiInterfaceMode() string {
	return os.Getenv("MULTI_INTERFACE_MODE")
}
