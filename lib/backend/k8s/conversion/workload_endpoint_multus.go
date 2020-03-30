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

package conversion

import (
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/names"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	netutils "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/utils"
	kapiv1 "k8s.io/api/core/v1"

	cnet "github.com/projectcalico/libcalico-go/lib/net"
)

type AdditionalPodInterfaceLimitExceededError struct{}

func (e *AdditionalPodInterfaceLimitExceededError) Error() string {
	return "a maximum of 9 additional interfaces is allowed on a pod"
}

func newMultusWorkloadEndpointConverter() *multusWorkloadEndpointConverter {
	return &multusWorkloadEndpointConverter{
		defaultConverter: defaultWorkloadEndpointConverter{},
	}
}

type multusWorkloadEndpointConverter struct {
	defaultConverter defaultWorkloadEndpointConverter
}

// VethNameForWorkload passes through to defaultWorkloadEndpointConverter.VethNameForWorkloadEndpoint, and exists to be
// compatible with the WorkloadEndpointConverter interface
func (m multusWorkloadEndpointConverter) VethNameForWorkload(namespace, podName string) string {
	return m.defaultConverter.VethNameForWorkload(namespace, podName)
}

// podCNCFNetStatusAnnotation gets the annotation name used for the network status information, if it's available.
func podCNCFNetStatusAnnotation(pod *kapiv1.Pod) string {
	annotations := pod.GetAnnotations()
	if _, exists := annotations[nettypes.OldNetworkStatusAnnot]; exists {
		return nettypes.OldNetworkStatusAnnot
	} else if _, exists := annotations[nettypes.NetworkStatusAnnot]; exists {
		return nettypes.NetworkStatusAnnot
	}

	return ""
}

// InterfacesForPod calculates the PodInterfaces for the interfaces on the given pod.
// If the MULTI_INTERFACE_MODE ENV variable is set to "multus" and the k8s.v1.cni.cncf.io network annotations are set,
// this function will return a PodInterface for each network interface configured by those annotations.
//
// The first entry in the returned PodInterface will always be the default interface. The IPNets for this
// entry will never be retrieved from the k8s.v1.cni.cncf.io annotations, but instead from the Pod "Status" or the
// cni.projectcalico.org IP annotations.
//
// If the k8s.v1.cni.cncf.io/network-status annotation is set then that is used to populate the PodInterfaces
// returned.
//
// If the k8s.v1.cni.cncf.io/network-status annotation is not available then the k8s.v1.cni.cncf.io/networks annotation
// used to calculate the PodInterfaces. In this case the IPNets will not be populated for the additional
// PodInterfaces (additional being entries other than the first one).
//
// Note: this assumes every network in the k8s.v1.cni.cncf.io/networks or k8s.v1.cni.cncf.io/network-status is a calico
// network
func (m multusWorkloadEndpointConverter) InterfacesForPod(pod *kapiv1.Pod) ([]*PodInterface, error) {
	var podIfaces []*PodInterface
	var err error

	annotations := pod.GetAnnotations()
	// Use the network-status annotation if available because it includes the IP addresses.
	if cncfNetStatusAnnot := podCNCFNetStatusAnnotation(pod); cncfNetStatusAnnot != "" {
		var networkStatuses []*nettypes.NetworkStatus
		if err := json.Unmarshal([]byte(annotations[cncfNetStatusAnnot]), &networkStatuses); err != nil {
			return nil, err
		}

		// default interface is included in this list, so 9 + 1 interfaces
		if len(networkStatuses) > 10 {
			return nil, new(AdditionalPodInterfaceLimitExceededError)
		}

		for i, networkStatus := range networkStatuses {
			var ipNets []*cnet.IPNet

			// first networkStatus is always for the default interface
			isDefault := i == 0
			if isDefault {
				// default entry doesn't use the network status annotation to populate it's IPs
				ipNets, err = getPodIPs(pod)
				if err != nil {
					return nil, err
				}
			} else {
				ipNets, err = stringsToIPNets(networkStatus.IPs)
				if err != nil {
					return nil, err
				}
			}

			podIfaces = append(podIfaces, &PodInterface{
				IsDefault:          isDefault, //first one is always the default interface
				NetworkName:        networkStatus.Name,
				InsidePodIfaceName: networkStatus.Interface,
				HostSideIfaceName:  calculateHostSideVethName(pod.Namespace, pod.Name, i),
				InsidePodGW:        net.IPv4(169, 254, 1, 1+byte(i)),
				IPNets:             ipNets,
			})
		}
	} else {
		// Add the default interface as the first element since it's not in the "k8s.v1.cni.cncf.io/networks" annotation.
		// Note: when the network-status annotation becomes available, the inside pod interface and network may change for
		// this default pod interface, as we don't know what they are at this point.
		defaultPodInterface, err := m.defaultInterfaceForPod(pod)
		if err != nil {
			return nil, err
		}

		podIfaces = append(podIfaces, defaultPodInterface)
		// If network-status isn't available yet use the networks instead, but the IP addresses aren't known yet.
		if annotations[nettypes.NetworkAttachmentAnnot] != "" {
			netSelectionElements, err := netutils.ParsePodNetworkAnnotation(pod)
			if err != nil {
				return nil, err
			}

			if len(netSelectionElements) > 9 {
				return nil, new(AdditionalPodInterfaceLimitExceededError)
			}

			for i, netSelectionElement := range netSelectionElements {
				if netSelectionElement.InterfaceRequest == "" {
					// this is what multus defaults the interface name to, if libcalico-go starts supporting more CNI
					// delegating plugins this will need to be revisited, as other plugins may not use the same default
					// interface naming scheme
					netSelectionElement.InterfaceRequest = fmt.Sprintf("net%d", i+1)
				}

				podIfaces = append(podIfaces, &PodInterface{
					NetworkName:        netSelectionElement.Name,
					InsidePodIfaceName: netSelectionElement.InterfaceRequest,
					HostSideIfaceName:  calculateHostSideVethName(pod.Namespace, pod.Name, i+1),
					InsidePodGW:        net.IPv4(169, 254, 1, 2+byte(i)),
				})
			}
		}
	}

	return podIfaces, nil
}

// DefaultInterfaceForPod retrieves the PodInterface for the default interface
// and populates it with the default values.
func (m multusWorkloadEndpointConverter) defaultInterfaceForPod(pod *kapiv1.Pod) (*PodInterface, error) {
	ipNets, err := getPodIPs(pod)
	if err != nil {
		return nil, err
	}

	return &PodInterface{
		IsDefault:          true,
		NetworkName:        "k8s-pod-network",
		InsidePodIfaceName: "eth0",
		HostSideIfaceName:  calculateHostSideVethName(pod.Namespace, pod.Name, 0),
		InsidePodGW:        net.IPv4(169, 254, 1, 1),
		IPNets:             ipNets,
	}, nil
}

// PodToWorkloadEndpoints calculates the WorkloadEndpoints for the given Pod. The ordering of the WorkloadEndpoints and
// whether multiple WorkloadEndpoints or just the default WorkloadEndpoint are returned depends on what's returned from
// InterfacesForPod.
func (m multusWorkloadEndpointConverter) PodToWorkloadEndpoints(pod *kapiv1.Pod) ([]*model.KVPair, error) {
	var kvps []*model.KVPair

	podIfaces, err := m.InterfacesForPod(pod)
	if err != nil {
		return nil, err
	}

	for _, podIface := range podIfaces {
		kvp, err := m.workloadEndpointForPodInterface(pod, *podIface)
		if err != nil {
			return nil, err
		}

		kvps = append(kvps, kvp)
	}

	return kvps, nil
}

// workloadEndpointForPodInterface calculates the WorkloadEndpoint for a Pod with the given PodInterface. It assumes the calling code
// has verified that the provided Pod is valid to convert to a WorkloadEndpoint.
//
// workloadEndpointFromPod requires the Pod name and Node name to be populated. It will fail to calculate the WorkloadEndpoint
// otherwise.
//
// workloadEndpointFoPod uses default WorkloadEndpointConverter to calculate the base WorkloadEndpoint, and updates the
// name and labels based on the podInterface given. Additionally, if the podInterface provided is for a non default WorkloadEndpoint
// then the following additional changes are made to the WorkloadEndpoint:
// * recalculated using the IPNets in the podInterfaces
// * the .Spec.IPNATs field is set to nil
// * the "cni.projectcalico.org/floatingIPs" annotation is removed
// * the .Spec.EgressGateway field is set to nil
func (m multusWorkloadEndpointConverter) workloadEndpointForPodInterface(pod *kapiv1.Pod, podInterface PodInterface) (*model.KVPair, error) {
	log.WithField("pod", pod).Debug("Converting pod to WorkloadEndpoint")

	defaultKVP, err := m.defaultConverter.podToDefaultWorkloadEndpoint(pod)
	if err != nil {
		return nil, err
	}

	wep := defaultKVP.Value.(*apiv3.WorkloadEndpoint)

	wepids := names.WorkloadEndpointIdentifiers{
		Node:         pod.Spec.NodeName,
		Orchestrator: apiv3.OrchestratorKubernetes,
		Endpoint:     podInterface.InsidePodIfaceName,
		Pod:          pod.Name,
	}
	wep.Name, err = wepids.CalculateWorkloadEndpointName(false)
	if err != nil {
		return nil, err
	}

	wep.Labels[apiv3.LabelNetwork] = podInterface.NetworkName
	wep.Labels[apiv3.LabelNetworkInterface] = podInterface.InsidePodIfaceName

	wep.Spec.Endpoint = podInterface.InsidePodIfaceName
	wep.Spec.InterfaceName = podInterface.HostSideIfaceName

	// If this is the default interface the calculations done with the IPs are correct by default, otherwise, some things
	// need to be recalculated
	if !podInterface.IsDefault {
		podIPNets := podInterface.IPNets
		if IsFinished(pod) {
			// Pod is finished but not yet deleted.  In this state the IP will have been freed and returned to the pool
			// so we need to make sure we don't let the caller believe it still belongs to this endpoint.
			// Pods with no IPs will get filtered out before they get to Felix in the watcher syncer cache layer.
			// We can't pretend the workload endpoint is deleted _here_ because that would confuse users of the
			// native v3 Watch() API.
			podIPNets = nil
		}

		wep.Spec.IPNetworks = []string{}
		for _, ipNet := range podIPNets {
			wep.Spec.IPNetworks = append(wep.Spec.IPNetworks, ipNet.String())
		}

		wep.Spec.IPNATs = nil
		if _, exists := wep.Annotations["cni.projectcalico.org/floatingIPs"]; exists {
			delete(wep.Annotations, "")
		}

		wep.Spec.EgressGateway = nil
	}

	// Embed the workload endpoint into a KVPair.
	kvp := model.KVPair{
		Key: model.ResourceKey{
			Name:      wep.Name,
			Namespace: pod.Namespace,
			Kind:      apiv3.KindWorkloadEndpoint,
		},
		Value:    wep,
		Revision: pod.ResourceVersion,
	}
	return &kvp, nil
}

// calculateHostSideVethName calculates what the interface name should for the host side of a veth pair be given the
// namespace, podName, and interface index. The namespace, podName, and encoding type are used to attempt to generate a unique
// suffix for the interface name given the restriction that an interface name is allowed a maximum size of 15 characters.
//
// The interfaceIndex is used to generate a prefix for the Veth name, and is the deciding factor on whether to hex encode
// or base32 encode the Veth name suffix. If there are multiple interfaces for the pod, this index corresponds to the order
// that the interfaces are added in. If the interfaceIndex is 0, it is assumed this is the default interface and the name
// will be of the form of cali<11-chars-of-hex-encoded-container-id>. If the interfaceIndex is not 0, it is assumed this
// is an additional interface for the pod and the name will be of the form of
// calim<interfaceIndex><9-chars-of-base32-encoded-container-id>. Note that if the environment variable FELIX_INTERFACEPREFIX
// is specified, the prefix "cali" is replace with the value of this environment variable
//
// The interface name suffix is created by first creating an ID for the pod by calculating the SHA1 from <namespace>.<podName>,
// then either hex or base32 encoding that SHA1. If the SHA1 is hex encoded then the first 11 characters of this ID are
// used as the suffix, otherwise the first 15-len(prefix) are used.
//
// Note on base32 encoding:
// The base32 encoding option was added so that we could generate a shorter suffix without losing entropy. This is needed
// when adding additional interfaces to a pod, as the prefix needs to be slightly longer for the additional interfaces,
// i.e. calim1, calim2. 9 characters of the base32 encoded ID has the same amount of entropy as 11 characters of the hex
// encoded ID.
func calculateHostSideVethName(namespace, podName string, interfaceIndex int) string {
	isDefault := interfaceIndex == 0

	basePrefix := os.Getenv("FELIX_INTERFACEPREFIX")
	if basePrefix == "" {
		// Prefix is not set. Default to "cali"
		basePrefix = "cali"
	} else {
		// Prefix is set - use the first value in the list.
		splits := strings.Split(basePrefix, ",")
		basePrefix = splits[0]
	}

	var prefix string
	if isDefault {
		prefix = basePrefix
	} else {
		prefix = fmt.Sprintf("%sm%d", basePrefix, interfaceIndex)
	}

	// A SHA1 is always 20 bytes long, and so is sufficient for generating the
	// veth name and mac addr.
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s.%s", namespace, podName)))

	var containerID string
	var containerSuffixLen int

	// If this is for the default interface then use hex encoding
	if isDefault {
		// this remains fixed at 11 characters for backwards compatibility
		containerSuffixLen = 11
		containerID = hex.EncodeToString(h.Sum(nil))
	} else {
		// Otherwise use base32 encoding
		containerSuffixLen = 9
		containerID = base32.StdEncoding.EncodeToString(h.Sum(nil))
	}

	containerSuffix := containerID[:containerSuffixLen]

	log.WithField("prefix", prefix).Debugf("Using prefix to create a WorkloadEndpoint veth name")
	return fmt.Sprintf("%s%s", prefix, containerSuffix)
}
