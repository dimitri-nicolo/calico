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
	"sync"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1alpha1"
	calicopolicy "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/policy"
	policystore "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/policy/storage"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	registry = registered.NewOrDie("")
	Scheme   = runtime.NewScheme()
	Codecs   = serializer.NewCodecFactory(Scheme)
)

// RESTStorageProvider provides a factory method to create a new APIGroupInfo for
// the calico API group. It implements (./pkg/apiserver).RESTStorageProvider
type RESTStorageProvider struct{}

// NewRESTStorage is a factory method to make a new APIGroupInfo for the
// calico API group.
func (p RESTStorageProvider) NewRESTStorage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter,
) (*genericapiserver.APIGroupInfo, error) {

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(calico.GroupName, registry, Scheme, metav1.ParameterCodec, Codecs)
	storage := p.v1alpha1Storage(apiResourceConfigSource, restOptionsGetter)
	apiGroupInfo.GroupMeta.GroupVersion = v1alpha1.SchemeGroupVersion

	apiGroupInfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		"v1alpha1": storage,
	}

	return &apiGroupInfo, nil
}

func (p RESTStorageProvider) v1alpha1Storage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter,
) map[string]rest.Storage {
	once := new(sync.Once)
	var (
		policyStorage rest.StandardStorage
	)

	initializeStorage := func() {
		once.Do(func() {
			policyStorage = policystore.NewREST(restOptionsGetter)
		})
	}

	storage := map[string]rest.Storage{}
	initializeStorage()
	storage["policies"] = calicopolicy.NewStorage(policyStorage)
	return storage
}
