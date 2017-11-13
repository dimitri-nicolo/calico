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
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	calicogpolicy "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/globalpolicy"
	calicopolicy "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/networkpolicy"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/server"
	calicotier "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/tier"
	"github.com/tigera/calico-k8sapiserver/pkg/storage/etcd"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"

	calicostorage "github.com/tigera/calico-k8sapiserver/pkg/storage/calico"
)

// RESTStorageProvider provides a factory method to create a new APIGroupInfo for
// the calico API group. It implements (./pkg/apiserver).RESTStorageProvider
type RESTStorageProvider struct {
	StorageType server.StorageType
}

// NewV3Storage constructs v3 api storage.
func (p RESTStorageProvider) NewV3Storage(
	scheme *runtime.Scheme,
	restOptionsGetter generic.RESTOptionsGetter,
	authorizer authorizer.Authorizer,
) (map[string]rest.Storage, error) {
	policyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("networkpolicies"))
	if err != nil {
		return nil, err
	}
	policyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   policyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicopolicy.EmptyObject(),
			ScopeStrategy: calicopolicy.NewStrategy(scheme),
			NewListFunc:   calicopolicy.NewList,
			GetAttrsFunc:  calicopolicy.GetAttrs,
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: policyRESTOptions,
		},
		p.StorageType,
		authorizer,
	)

	tierRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("tiers"))
	if err != nil {
		return nil, err
	}
	tierOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   tierRESTOptions,
			Capacity:      1000,
			ObjectType:    calicotier.EmptyObject(),
			ScopeStrategy: calicotier.NewStrategy(scheme),
			NewListFunc:   calicotier.NewList,
			GetAttrsFunc:  calicotier.GetAttrs,
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: tierRESTOptions,
		},
		p.StorageType,
		authorizer,
	)

	gpolicyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalnetworkpolicies"))
	if err != nil {
		return nil, err
	}
	gpolicyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   gpolicyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicogpolicy.EmptyObject(),
			ScopeStrategy: calicogpolicy.NewStrategy(scheme),
			NewListFunc:   calicogpolicy.NewList,
			GetAttrsFunc:  calicogpolicy.GetAttrs,
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: gpolicyRESTOptions,
		},
		p.StorageType,
		authorizer,
	)

	storage := map[string]rest.Storage{}
	storage["networkpolicies"] = calicopolicy.NewREST(scheme, *policyOpts)
	storage["tiers"] = calicotier.NewREST(scheme, *tierOpts)
	storage["globalnetworkpolicies"] = calicogpolicy.NewREST(scheme, *gpolicyOpts)

	return storage, nil
}

// GroupName returns the API group name.
func (p RESTStorageProvider) GroupName() string {
	return calico.GroupName
}
