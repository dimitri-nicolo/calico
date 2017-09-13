/*
Copyright 2016 The Kubernetes Authors.

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

package rest

import (
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
	"github.com/tigera/calico-k8sapiserver/pkg/storage/etcd"
	// calicoendpoint "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/endpoint"
	// caliconode "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/node"
	calicopolicy "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/policy"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/calico/server"
	calicotier "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/tier"

	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/pkg/api"

	calicov1 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1"
)

// RESTStorageProvider provides a factory method to create a new APIGroupInfo for
// the calico API group. It implements (./pkg/apiserver).RESTStorageProvider
type RESTStorageProvider struct {
	StorageType server.StorageType
}

// NewRESTStorage is a factory method to make a new APIGroupInfo for the
// calico API group.
func (p RESTStorageProvider) NewRESTStorage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter,
	authorizer authorizer.Authorizer,
) (*genericapiserver.APIGroupInfo, error) {
	storage, err := p.v1Storage(apiResourceConfigSource, restOptionsGetter, authorizer)
	if err != nil {
		return nil, err
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(calico.GroupName, api.Registry, api.Scheme, api.ParameterCodec, api.Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = v1.SchemeGroupVersion

	apiGroupInfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		calicov1.SchemeGroupVersion.Version: storage,
	}

	return &apiGroupInfo, nil
}

func (p RESTStorageProvider) v1Storage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter,
	authorizer authorizer.Authorizer,
) (map[string]rest.Storage, error) {
	policyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("policies"))
	if err != nil {
		return nil, err
	}
	policyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   policyRESTOptions,
			Capacity:      1000,
			ObjectType:    policy.EmptyObject(),
			ScopeStrategy: policy.NewScopeStrategy(),
			NewListFunc:   policy.NewList,
			GetAttrsFunc:  policy.GetAttrs,
			Trigger:       storage.NoTriggerPublisher,
		},
		calico.Options{},
		p.StorageType,
		authorizer,
	)

	storage := map[string]rest.Storage{}
	storage["policies"] = calicopolicy.NewREST(*policyOpts)
	storage["tiers"] = calicotier.NewREST(restOptionsGetter, storage["policies"])
	// storage["endpoints"] = calicoendpoint.NewREST(restOptionsGetter, authorizer)
	// storage["nodes"] = caliconode.NewREST(restOptionsGetter, authorizer)

	return storage, nil
}

// GroupName returns the API group name.
func (p RESTStorageProvider) GroupName() string {
	return calico.GroupName
}
