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

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/types"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
)

func NewWorkloadEndpointClient(c kubernetes.Interface) K8sResourceClient {
	return &WorkloadEndpointClient{
		clientSet: c,
		converter: conversion.NewConverter(),
	}
}

// Implements the api.Client interface for WorkloadEndpoints.
type WorkloadEndpointClient struct {
	clientSet kubernetes.Interface
	converter conversion.Converter
}

func (c *WorkloadEndpointClient) Create(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	log.Debug("Received Create request on WorkloadEndpoint type")
	// As a special case for the CNI plugin, try to patch the Pod with the IP that we've calculated.
	// This works around a bug in kubelet that causes it to delay writing the Pod IP for a long time:
	// https://github.com/kubernetes/kubernetes/issues/39113.
	//
	// Note: it's a bit odd to do this in the Create, but the CNI plugin uses CreateOrUpdate().  Doing it
	// here makes sure that, if the update fails: we retry here, and, we don't report success without
	// making the patch.
	return c.patchPodIP(ctx, kvp)
}

func (c *WorkloadEndpointClient) Update(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	log.Debug("Received Update request on WorkloadEndpoint type")
	// As a special case for the CNI plugin, try to patch the Pod with the IP that we've calculated.
	// This works around a bug in kubelet that causes it to delay writing the Pod IP for a long time:
	// https://github.com/kubernetes/kubernetes/issues/39113.
	return c.patchPodIP(ctx, kvp)
}

func (c *WorkloadEndpointClient) DeleteKVP(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	return c.Delete(ctx, kvp.Key, kvp.Revision, kvp.UID)
}

func (c *WorkloadEndpointClient) Delete(ctx context.Context, key model.Key, revision string, uid *types.UID) (*model.KVPair, error) {
	log.Warn("Operation Delete is not supported on WorkloadEndpoint type")
	return nil, cerrors.ErrorOperationNotSupported{
		Identifier: key,
		Operation:  "Delete",
	}
}

// patchPodIP PATCHes the Kubernetes Pod associated with the given KVPair with the IP addresses it contains.
// This is a no-op if there is no IP address.
//
// We store the IP addresses in annotations because patching the PodStatus directly races with changes that
// kubelet makes so kubelet can undo our changes.
func (c *WorkloadEndpointClient) patchPodIP(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	ips := kvp.Value.(*apiv3.WorkloadEndpoint).Spec.IPNetworks
	if len(ips) == 0 {
		return kvp, nil
	}

	log.Debugf("PATCHing pod with IPs: %v", ips)
	wepID, err := c.converter.ParseWorkloadEndpointName(kvp.Key.(model.ResourceKey).Name)
	if err != nil {
		return nil, err
	}
	if wepID.Pod == "" {
		return nil, cerrors.ErrorInsufficientIdentifiers{Name: kvp.Key.(model.ResourceKey).Name}
	}
	// Write the IP addresses into annotations.  This generates an event more quickly than
	// waiting for kubelet to update the PodStatus PodIP and PodIPs fields.
	ns := kvp.Key.(model.ResourceKey).Namespace
	patch, err := calculateAnnotationPatch(
		conversion.AnnotationPodIP, ips[0],
		conversion.AnnotationPodIPs, strings.Join(ips, ","),
	)
	if err != nil {
		log.WithError(err).Error("failed to calculate Pod patch.")
		return nil, err
	}
	pod, err := c.clientSet.CoreV1().Pods(ns).Patch(wepID.Pod, types.StrategicMergePatchType, patch, "status")
	if err != nil {
		return nil, K8sErrorToCalico(err, kvp.Key)
	}
	log.Debugf("Successfully PATCHed pod to add podIP annotation: %+v", pod)

	kvps, err := c.converter.PodToWorkloadEndpoints(pod)
	if err != nil {
		return nil, err
	}

	return kvps[0], nil
}

const annotationNameValueTemplate = `%s: %s`
const annotationPatchTemplate = `{"metadata": {"annotations": {%s}}}`

func calculateAnnotationPatch(namesAndValues ...string) ([]byte, error) {
	settings := []string{}
	for i := 0; i < len(namesAndValues); i += 2 {
		// Marshal the key and value in order to make sure all the escaping is done correctly.
		nameJson, err := json.Marshal(namesAndValues[i])
		if err != nil {
			return nil, err
		}
		valueJson, err := json.Marshal(namesAndValues[i+1])
		if err != nil {
			return nil, err
		}
		settings = append(settings, fmt.Sprintf(annotationNameValueTemplate, nameJson, valueJson))
	}
	patch := []byte(fmt.Sprintf(annotationPatchTemplate, strings.Join(settings, ", ")))
	return patch, nil
}

func (c *WorkloadEndpointClient) Get(ctx context.Context, key model.Key, revision string) (*model.KVPair, error) {
	log.Debug("Received Get request on WorkloadEndpoint type")
	k := key.(model.ResourceKey)

	// Parse resource name so we can get get the podName
	wepID, err := c.converter.ParseWorkloadEndpointName(key.(model.ResourceKey).Name)
	if err != nil {
		return nil, err
	}
	if wepID.Pod == "" {
		return nil, cerrors.ErrorResourceDoesNotExist{
			Identifier: key,
			Err:        errors.New("malformed WorkloadEndpoint name - unable to determine Pod name"),
		}
	}

	pod, err := c.clientSet.CoreV1().Pods(k.Namespace).Get(wepID.Pod, metav1.GetOptions{ResourceVersion: revision})
	if err != nil {
		return nil, K8sErrorToCalico(err, k)
	}

	// Decide if this pod should be displayed.
	if !c.converter.IsValidCalicoWorkloadEndpoint(pod) {
		return nil, cerrors.ErrorResourceDoesNotExist{Identifier: k}
	}

	kvps, err := c.converter.PodToWorkloadEndpoints(pod)
	if err != nil {
		return nil, err
	}

	// Find the WorkloadEndpoint that has a name matching the name in the given key
	for _, kvp := range kvps {
		wep := kvp.Value.(*apiv3.WorkloadEndpoint)
		if wep.Name == key.(model.ResourceKey).Name {
			return kvp, nil
		}
	}

	return nil, kerrors.NewNotFound(apiv3.Resource("WorkloadEndpoint"), key.String())
}

func (c *WorkloadEndpointClient) List(ctx context.Context, list model.ListInterface, revision string) (*model.KVPairList, error) {
	log.Debug("Received List request on WorkloadEndpoint type")
	l := list.(model.ResourceListOptions)

	// If a "Name" is provided, we may be able to get the exact WorkloadEndpoint or narrow the WorkloadEndpoints to a
	// single Pod.
	if l.Name != "" {
		return c.listUsingName(ctx, l, revision)
	}

	return c.list(l, revision)
}

// listUsingName uses the name in the listOptions to retrieve the WorkloadEndpoints. The name, at the very least, must identify
// a single Pod, otherwise an error will occur.
func (c *WorkloadEndpointClient) listUsingName(ctx context.Context, listOptions model.ResourceListOptions, revision string) (*model.KVPairList, error) {
	wepID, err := c.converter.ParseWorkloadEndpointName(listOptions.Name)
	if err != nil {
		return nil, err
	}

	if wepID.Pod == "" {
		return nil, cerrors.ErrorResourceDoesNotExist{
			Identifier: listOptions,
			Err:        errors.New("malformed WorkloadEndpoint name - unable to determine Pod name"),
		}
	}

	pod, err := c.clientSet.CoreV1().Pods(listOptions.Namespace).Get(wepID.Pod, metav1.GetOptions{ResourceVersion: revision})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return &model.KVPairList{
				KVPairs:  []*model.KVPair{},
				Revision: revision,
			}, nil
		} else {
			return nil, err
		}
	}

	kvps, err := c.converter.PodToWorkloadEndpoints(pod)
	if err != nil {
		return nil, err
	}

	// If Endpoint is available get the single WorkloadEndpoint
	if wepID.Endpoint != "" {
		// Set to an empty list in case a match isn't found
		var tmpKVPs []*model.KVPair

		wepName, err := wepID.CalculateWorkloadEndpointName(false)
		if err != nil {
			return nil, err
		}
		// Find the WorkloadEndpoint that has a name matching the name in the given key
		for _, kvp := range kvps {
			wep := kvp.Value.(*apiv3.WorkloadEndpoint)
			if wep.Name == wepName {
				tmpKVPs = []*model.KVPair{kvp}
				break
			}
		}

		kvps = tmpKVPs
	}

	return &model.KVPairList{
		KVPairs:  kvps,
		Revision: revision,
	}, nil
}

// list lists all the Workload endpoints for the namespace given in listOptions.
func (c *WorkloadEndpointClient) list(listOptions model.ResourceListOptions, revision string) (*model.KVPairList, error) {
	podList, err := c.clientSet.CoreV1().Pods(listOptions.Namespace).List(metav1.ListOptions{ResourceVersion: revision})
	if err != nil {
		return nil, K8sErrorToCalico(err, listOptions)
	}

	// For each Pod, return a workload endpoint.
	var ret []*model.KVPair
	for _, pod := range podList.Items {
		// Decide if this pod should be included.
		if !c.converter.IsValidCalicoWorkloadEndpoint(&pod) {
			continue
		}

		kvps, err := c.converter.PodToWorkloadEndpoints(&pod)
		if err != nil {
			return nil, err
		}
		ret = append(ret, kvps...)
	}

	return &model.KVPairList{
		KVPairs:  ret,
		Revision: revision,
	}, nil
}

func (c *WorkloadEndpointClient) EnsureInitialized() error {
	return nil
}

func (c *WorkloadEndpointClient) Watch(ctx context.Context, list model.ListInterface, revision string) (api.WatchInterface, error) {
	// Build watch options to pass to k8s.
	opts := metav1.ListOptions{ResourceVersion: revision, Watch: true}
	rlo, ok := list.(model.ResourceListOptions)
	if !ok {
		return nil, fmt.Errorf("ListInterface is not a ResourceListOptions: %s", list)
	}
	if len(rlo.Name) != 0 {
		if len(rlo.Namespace) == 0 {
			return nil, errors.New("cannot watch a specific WorkloadEndpoint without a namespace")
		}
		// We've been asked to watch a specific workloadendpoint
		wepids, err := c.converter.ParseWorkloadEndpointName(rlo.Name)
		if err != nil {
			return nil, err
		}
		log.WithField("name", wepids.Pod).Debug("Watching a single workloadendpoint")
		opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", wepids.Pod).String()
	}

	ns := rlo.Namespace
	k8sWatch, err := c.clientSet.CoreV1().Pods(ns).Watch(opts)
	if err != nil {
		return nil, K8sErrorToCalico(err, list)
	}
	converter := func(r Resource) ([]*model.KVPair, error) {
		k8sPod, ok := r.(*kapiv1.Pod)
		if !ok {
			return nil, errors.New("Pod conversion with incorrect k8s resource type")
		}
		if !c.converter.IsValidCalicoWorkloadEndpoint(k8sPod) {
			// If this is not a valid Calico workload endpoint then don't return in the watch.
			// Returning a nil KVP and a nil error swallows the event.
			return nil, nil
		}
		return c.converter.PodToWorkloadEndpoints(k8sPod)
	}
	return newK8sWatcherConverterOneToMany(ctx, "Pod", converter, k8sWatch), nil
}
