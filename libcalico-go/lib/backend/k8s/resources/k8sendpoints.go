// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package resources

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
)

// This client is a direct mapping through to the kubernetes endpoints API. It is used by the k8s-resource wrapper
// client (instantiated using k8s.NewK8sResourceWrapperClient) and provides read-only access to the kubernetes
// resources. This allows a syncer to return the kubernetes resources that are included in this client,
// These resource types are only accessible through the backend client API.

func NewEndpointsClient(c *kubernetes.Clientset) K8sResourceClient {
	return &endpointsClient{
		clientSet: c,
	}
}

// Implements the api.Client interface for Kubernetes Endpoints.
type endpointsClient struct {
	clientSet *kubernetes.Clientset
}

func (c *endpointsClient) Create(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	log.Warn("Operation Create is not supported on Kubernetes Endpoints type")
	return nil, cerrors.ErrorOperationNotSupported{
		Identifier: kvp.Key,
		Operation:  "Create",
	}
}

func (c *endpointsClient) Update(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	log.Warn("Operation Update is not supported on Kubernetes Endpoints type")
	return nil, cerrors.ErrorOperationNotSupported{
		Identifier: kvp.Key,
		Operation:  "Update",
	}
}

func (c *endpointsClient) DeleteKVP(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	return c.Delete(ctx, kvp.Key, kvp.Revision, kvp.UID)
}

func (c *endpointsClient) Delete(ctx context.Context, key model.Key, revision string, uid *types.UID) (*model.KVPair, error) {
	log.Warn("Operation Delete is not supported on Kubernetes Endpoints type")
	return nil, cerrors.ErrorOperationNotSupported{
		Identifier: key,
		Operation:  "Delete",
	}
}

func (c *endpointsClient) Get(ctx context.Context, key model.Key, revision string) (*model.KVPair, error) {
	rk := key.(model.ResourceKey)
	endpoints, err := c.clientSet.CoreV1().Endpoints(rk.Namespace).Get(ctx, rk.Name, metav1.GetOptions{ResourceVersion: revision})
	if err != nil {
		return nil, K8sErrorToCalico(err, key)
	}
	return c.convertToKVPair(endpoints), nil
}

func (c *endpointsClient) List(ctx context.Context, list model.ListInterface, revision string) (*model.KVPairList, error) {
	log.Debug("Received List request on Kubernetes Endpoints type")
	rl := list.(model.ResourceListOptions)
	kvps := []*model.KVPair{}

	if rl.Name != "" {
		// The endpoints is already fully qualified, so perform a Get instead.
		// If the entry does not exist then we just return an empty list.
		kvp, err := c.Get(ctx, model.ResourceKey{Name: rl.Name, Namespace: rl.Namespace, Kind: apiv3.KindK8sEndpoints}, revision)
		if err != nil {
			if _, ok := err.(cerrors.ErrorResourceDoesNotExist); !ok {
				return nil, err
			}
			return &model.KVPairList{
				KVPairs:  kvps,
				Revision: revision,
			}, nil
		}

		kvps = append(kvps, kvp)
		return &model.KVPairList{
			KVPairs:  kvps,
			Revision: revision,
		}, nil
	}

	// Listing all endpointss.
	endpointsList, err := c.clientSet.CoreV1().Endpoints(rl.Namespace).List(ctx, metav1.ListOptions{ResourceVersion: revision})
	if err != nil {
		return nil, K8sErrorToCalico(err, list)
	}

	for i := range endpointsList.Items {
		kvps = append(kvps, c.convertToKVPair(&endpointsList.Items[i]))
	}

	return &model.KVPairList{
		KVPairs:  kvps,
		Revision: endpointsList.ResourceVersion,
	}, nil
}

func (c *endpointsClient) EnsureInitialized() error {
	return nil
}

func (c *endpointsClient) Watch(ctx context.Context, list model.ListInterface, opts api.WatchOptions) (api.WatchInterface, error) {
	// Build watch options to pass to k8s.
	k8sOpts := metav1.ListOptions{ResourceVersion: opts.Revision, Watch: true}
	rl, ok := list.(model.ResourceListOptions)
	if !ok {
		return nil, fmt.Errorf("ListInterface is not a ResourceListOptions: %s", list)
	}
	if len(rl.Name) != 0 {
		// We've been asked to watch a specific endpoints resource.
		log.WithField("name", rl.Name).Debug("Watching a single endpoints")
		k8sOpts.FieldSelector = fields.OneTermEqualSelector("metadata.name", rl.Name).String()
	}

	k8sWatch, err := c.clientSet.CoreV1().Endpoints(rl.Namespace).Watch(ctx, k8sOpts)
	if err != nil {
		return nil, K8sErrorToCalico(err, list)
	}
	converter := func(r Resource) (*model.KVPair, error) {
		return c.convertToKVPair(r.(*kapiv1.Endpoints)), nil
	}
	return newK8sWatcherConverter(ctx, "Kubernetes Endpoints", converter, k8sWatch), nil
}

func (c *endpointsClient) convertToKVPair(endpoints *kapiv1.Endpoints) *model.KVPair {
	return &model.KVPair{
		Key: model.ResourceKey{
			Name:      endpoints.Name,
			Namespace: endpoints.Namespace,
			Kind:      apiv3.KindK8sEndpoints,
		},
		Value:    endpoints,
		Revision: endpoints.ResourceVersion,
	}
}
