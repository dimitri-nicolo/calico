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

package apiserver

import (
	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/apis/v2"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	calicov2 "github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v2"
	calicorest "github.com/tigera/calico-k8sapiserver/pkg/registry/calico/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/storage"
)

// calicoConfig contains a generic API server Config along with config specific to
// the calico API server.
type calicoConfig struct {
	genericConfig *genericapiserver.Config
}

// NewCalicoConfig returns a new server config to describe an etcd-backed API server
func NewCalicoConfig(
	genCfg *genericapiserver.Config,
) Config {
	return &calicoConfig{
		genericConfig: genCfg,
	}
}

// Complete fills in any fields not set that are required to have valid data
// and can be derived from other fields.
func (c *calicoConfig) Complete() CompletedConfig {
	completeGenericConfig(c.genericConfig)
	return completedCalicoConfig{
		calicoConfig: c,
		// Not every API group compiled in is necessarily enabled by the operator
		// at runtime.
		//
		// Install the API resource config source, which describes versions of
		// which API groups are enabled.
		apiResourceConfigSource: DefaultAPIResourceConfigSource(),
	}
}

// CompletedCalicoConfig is an internal type to take advantage of typechecking in
// the type system.
type completedCalicoConfig struct {
	*calicoConfig
	apiResourceConfigSource storage.APIResourceConfigSource
}

// NewServer creates a new server that can be run. Returns a non-nil error if the server couldn't
// be created
func (c completedCalicoConfig) NewServer() (*CalicoAPIServer, error) {
	s, err := createSkeletonServer(c.genericConfig)
	if err != nil {
		return nil, err
	}
	glog.V(4).Infoln("Created skeleton API server")

	glog.V(4).Infoln("Installing API group")
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(calico.GroupName, registry, Scheme, metav1.ParameterCodec, Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = v2.SchemeGroupVersion
	// TODO: Make the storage type configurable
	calicostore := calicorest.RESTStorageProvider{StorageType: "calico"}
	storage, err := calicostore.NewV2Storage(c.apiResourceConfigSource, c.genericConfig.RESTOptionsGetter, c.genericConfig.Authorizer)
	if err != nil {
		return nil, err
	}
	apiGroupInfo.VersionedResourcesStorageMap = map[string]map[string]rest.Storage{
		calicov2.SchemeGroupVersion.Version: storage,
	}

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		glog.Fatalf("Error installing API group %v: %v", calicostore.GroupName(), err)
	}

	glog.Infoln("Finished installing API groups")

	return s, nil
}
