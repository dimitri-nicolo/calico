// Copyright (c) 2017 Tigera, Inc. All rights reserved.

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
	"errors"
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cerrors "github.com/projectcalico/libcalico-go/lib/errors"
	"k8s.io/apimachinery/pkg/fields"
)

// customK8sResourceClient implements the K8sResourceClient interface and provides a generic
// mechanism for a 1:1 mapping between a Calico Resource and an equivalent Kubernetes
// custom resource type.
type customK8sResourceClient struct {
	clientSet           *kubernetes.Clientset
	restClient          *rest.RESTClient
	name                string
	resource            string
	description         string
	k8sResourceType     reflect.Type
	k8sResourceTypeMeta metav1.TypeMeta
	k8sListType         reflect.Type
	namespaced          bool
	resourceKind        string
}

// Create creates a new Custom K8s Resource instance in the k8s API from the supplied KVPair.
func (c *customK8sResourceClient) Create(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	logContext := log.WithFields(log.Fields{
		"Key":      kvp.Key,
		"Value":    kvp.Value,
		"Resource": c.resource,
	})
	logContext.Debug("Create custom Kubernetes resource")

	// Convert the KVPair to the K8s resource.
	resIn, err := c.convertKVPairToResource(kvp)
	if err != nil {
		logContext.WithError(err).Info("Error creating resource")
		return nil, err
	}

	// Send the update request using the REST interface.
	resOut := reflect.New(c.k8sResourceType).Interface().(Resource)
	namespace := kvp.Key.(model.ResourceKey).Namespace
	err = c.restClient.Post().
		NamespaceIfScoped(namespace, c.namespaced).
		Context(ctx).
		Resource(c.resource).
		Body(resIn).
		Do().Into(resOut)
	if err != nil {
		logContext.WithError(err).Info("Error creating resource")
		return nil, K8sErrorToCalico(err, kvp.Key)
	}

	// Update the revision information from the response.
	kvp.Revision = resOut.GetObjectMeta().GetResourceVersion()
	return kvp, nil
}

// Update updates an existing Custom K8s Resource instance in the k8s API from the supplied KVPair.
func (c *customK8sResourceClient) Update(ctx context.Context, kvp *model.KVPair) (*model.KVPair, error) {
	logContext := log.WithFields(log.Fields{
		"Key":      kvp.Key,
		"Value":    kvp.Value,
		"Resource": c.resource,
	})
	logContext.Debug("Update custom Kubernetes resource")

	// Create storage for the updated resource.
	resOut := reflect.New(c.k8sResourceType).Interface().(Resource)

	var updateError error
	// Convert the KVPair to a K8s resource.
	resIn, err := c.convertKVPairToResource(kvp)
	if err != nil {
		logContext.WithError(err).Info("Error updating resource")
		return nil, err
	}

	// Send the update request using the name.
	name := resIn.GetObjectMeta().GetName()
	namespace := resIn.GetObjectMeta().GetNamespace()
	logContext = logContext.WithField("Name", name)
	logContext.Debug("Update resource by name")
	updateError = c.restClient.Put().
		Context(ctx).
		Resource(c.resource).
		NamespaceIfScoped(namespace, c.namespaced).
		Body(resIn).
		Name(name).
		Do().Into(resOut)
	if updateError != nil {
		// Failed to update the resource.
		logContext.WithError(updateError).Error("Error updating resource")
		return nil, K8sErrorToCalico(updateError, kvp.Key)
	}

	resOut.GetObjectKind().SetGroupVersionKind(c.k8sResourceTypeMeta.GetObjectKind().GroupVersionKind())

	// Success. Update the revision information from the response.
	kvp.Revision = resOut.GetObjectMeta().GetResourceVersion()
	return kvp, nil
}

// Delete deletes an existing Custom K8s Resource instance in the k8s API using the supplied KVPair.
func (c *customK8sResourceClient) Delete(ctx context.Context, k model.Key, revision string) (*model.KVPair, error) {
	logContext := log.WithFields(log.Fields{
		"Key":      k,
		"Resource": c.resource,
	})
	logContext.Debug("Delete custom Kubernetes resource")

	// Convert the Key to a resource name.
	name, err := c.keyToName(k)
	if err != nil {
		logContext.WithError(err).Info("Error deleting resource")
		return nil, err
	}

	existing, err := c.Get(ctx, k, revision)
	if err != nil {
		return nil, err
	}

	namespace := k.(model.ResourceKey).Namespace

	// Delete the resource using the name.
	logContext = logContext.WithField("Name", name)
	logContext.Debug("Send delete request by name")
	err = c.restClient.Delete().
		Context(ctx).
		NamespaceIfScoped(namespace, c.namespaced).
		Resource(c.resource).
		Name(name).
		Do().
		Error()
	if err != nil {
		logContext.WithError(err).Info("Error deleting resource")
		return nil, K8sErrorToCalico(err, k)
	}
	return existing, nil
}

// Get gets an existing Custom K8s Resource instance in the k8s API using the supplied Key.
func (c *customK8sResourceClient) Get(ctx context.Context, key model.Key, revision string) (*model.KVPair, error) {
	logContext := log.WithFields(log.Fields{
		"Key":      key,
		"Resource": c.resource,
		"Revision": revision,
	})
	logContext.Debug("Get custom Kubernetes resource")
	name, err := c.keyToName(key)
	if err != nil {
		logContext.WithError(err).Info("Error getting resource")
		return nil, err
	}
	namespace := key.(model.ResourceKey).Namespace

	// Add the name to the log context now that we know it, and query
	// Kubernetes.
	logContext = logContext.WithField("Name", name)
	logContext.Debug("Get custom Kubernetes resource by name")
	resOut := reflect.New(c.k8sResourceType).Interface().(Resource)
	err = c.restClient.Get().
		Context(ctx).
		NamespaceIfScoped(namespace, c.namespaced).
		Resource(c.resource).
		Name(name).
		Do().Into(resOut)
	if err != nil {
		logContext.WithError(err).Info("Error getting resource")
		return nil, K8sErrorToCalico(err, key)
	}
	resOut.GetObjectKind().SetGroupVersionKind(c.k8sResourceTypeMeta.GetObjectKind().GroupVersionKind())

	return c.convertResourceToKVPair(resOut)
}

// List lists configured Custom K8s Resource instances in the k8s API matching the
// supplied ListInterface.
func (c *customK8sResourceClient) List(ctx context.Context, list model.ListInterface, revision string) (*model.KVPairList, error) {
	logContext := log.WithFields(log.Fields{
		"ListInterface": list,
		"Resource":      c.resource,
	})
	logContext.Debug("List Custom K8s Resource")
	kvps := []*model.KVPair{}

	if revision != "" {
		return nil, errors.New("Cannot List this resource type specifying a ResourceVersion")
	}

	// Attempt to convert the ListInterface to a Key.  If possible, the parameters
	// indicate a fully qualified resource, and we'll need to use Get instead of
	// List.
	if key := c.listInterfaceToKey(list); key != nil {
		logContext.Debug("Performing List using Get")
		if kvp, err := c.Get(ctx, key, revision); err != nil {
			// The error will already be a Calico error type.  Ignore
			// error that it doesn't exist - we'll return an empty
			// list.
			if _, ok := err.(cerrors.ErrorResourceDoesNotExist); !ok {
				log.WithField("Resource", c.resource).WithError(err).Info("Error listing resource")
				return nil, err
			}
			return &model.KVPairList{
				KVPairs:  kvps,
				Revision: revision,
			}, nil
		} else {
			kvps = append(kvps, kvp)
			return &model.KVPairList{
				KVPairs:  kvps,
				Revision: revision,
			}, nil
		}
	}

	// Since we are not performing an exact Get, Kubernetes will return a
	// list of resources.
	reslOut := reflect.New(c.k8sListType).Interface().(ResourceList)

	// If it is a namespaced resource, then we'll need the namespace.
	namespace := list.(model.ResourceListOptions).Namespace

	// Perform the request.
	err := c.restClient.Get().
		Context(ctx).
		NamespaceIfScoped(namespace, c.namespaced).
		Resource(c.resource).
		Do().Into(reslOut)
	if err != nil {
		// Don't return errors for "not found".  This just
		// means there are no matching Custom K8s Resources, and we should return
		// an empty list.
		if !kerrors.IsNotFound(err) {
			log.WithError(err).Info("Error listing resources")
			return nil, K8sErrorToCalico(err, list)
		}
		return &model.KVPairList{
			KVPairs:  kvps,
			Revision: revision,
		}, nil
	}

	// We expect the list type to have an "Items" field that we can
	// iterate over.
	elem := reflect.ValueOf(reslOut).Elem()
	items := reflect.ValueOf(elem.FieldByName("Items").Interface())
	for idx := 0; idx < items.Len(); idx++ {
		res := items.Index(idx).Addr().Interface().(Resource)
		res.GetObjectKind().SetGroupVersionKind(c.k8sResourceTypeMeta.GetObjectKind().GroupVersionKind())

		if kvp, err := c.convertResourceToKVPair(res); err == nil {
			kvps = append(kvps, kvp)
		} else {
			logContext.WithError(err).WithField("Item", res).Warning("unable to process resource, skipping")
		}
	}
	return &model.KVPairList{
		KVPairs:  kvps,
		Revision: reslOut.GetListMeta().GetResourceVersion(),
	}, nil
}

func (c *customK8sResourceClient) Watch(ctx context.Context, list model.ListInterface, revision string) (api.WatchInterface, error) {
	resl, ok := list.(model.ResourceListOptions)
	if !ok {
		return nil, errors.New("internal error: custom resource watch invoked for non v2 resource type")
	}
	if len(resl.Name) != 0 {
		return nil, fmt.Errorf("cannot watch specific resource instance: %s", resl.Name)
	}

	k8sWatchClient := cache.NewListWatchFromClient(
		c.restClient,
		c.resource,
		resl.Namespace,
		fields.Everything())
	k8sWatch, err := k8sWatchClient.WatchFunc(metav1.ListOptions{ResourceVersion: revision})
	if err != nil {
		return nil, K8sErrorToCalico(err, list)
	}
	toKVPair := func(r Resource) (*model.KVPair, error) {
		r.GetObjectKind().SetGroupVersionKind(c.k8sResourceTypeMeta.GetObjectKind().GroupVersionKind())
		return c.convertResourceToKVPair(r)
	}
	return newK8sWatcherConverter(ctx, resl.Kind+" (custom)", toKVPair, k8sWatch), nil
}

// EnsureInitialized is a no-op since the CRD should be
// initialized in advance.
func (c *customK8sResourceClient) EnsureInitialized() error {
	return nil
}

func (c *customK8sResourceClient) listInterfaceToKey(l model.ListInterface) model.Key {
	pl := l.(model.ResourceListOptions)
	if pl.Name != "" {
		return model.ResourceKey{Name: pl.Name, Kind: pl.Kind}
	}
	return nil
}

func (c *customK8sResourceClient) keyToName(k model.Key) (string, error) {
	return k.(model.ResourceKey).Name, nil
}

func (c *customK8sResourceClient) nameToKey(name string) (model.Key, error) {
	return model.ResourceKey{
		Name: name,
		Kind: c.resourceKind,
	}, nil
}

func (c *customK8sResourceClient) convertResourceToKVPair(r Resource) (*model.KVPair, error) {
	kvp := &model.KVPair{
		Key: model.ResourceKey{
			Name:      r.GetObjectMeta().GetName(),
			Namespace: r.GetObjectMeta().GetNamespace(),
			Kind:      c.resourceKind,
		},
		Value:    r,
		Revision: r.GetObjectMeta().GetResourceVersion(),
	}

	return kvp, nil
}

func (c *customK8sResourceClient) convertKVPairToResource(kvp *model.KVPair) (Resource, error) {
	resource := kvp.Value.(Resource)
	resource.GetObjectMeta().SetResourceVersion(kvp.Revision)

	return resource, nil
}
