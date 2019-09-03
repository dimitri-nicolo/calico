// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package apiserver

import (
	calicorest "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/rest"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/install"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var (
	groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
	registry             = registered.NewOrDie("")
	Scheme               = runtime.NewScheme()
	Codecs               = serializer.NewCodecFactory(Scheme)
)

func init() {
	install.Install(groupFactoryRegistry, registry, Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

type ExtraConfig struct {
	// Place you custom config here.
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// ProjectCalicoServer contains state for a Kubernetes cluster master/api server.
type ProjectCalicoServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of WardleServer from the given config.
func (c completedConfig) New() (*ProjectCalicoServer, error) {
	genericServer, err := c.GenericConfig.New("calico-k8sapiserver", genericapiserver.EmptyDelegate)
	if err != nil {
		return nil, err
	}

	s := &ProjectCalicoServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(projectcalico.GroupName, registry, Scheme, metav1.ParameterCodec, Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = v3.SchemeGroupVersion
	// TODO: Make the storage type configurable
	calicostore := calicorest.RESTStorageProvider{StorageType: "calico"}
	apiGroupInfo.VersionedResourcesStorageMap["v3"], err = calicostore.NewV3Storage(Scheme, c.GenericConfig.RESTOptionsGetter, c.GenericConfig.Authorization.Authorizer)
	if err != nil {
		return nil, err
	}

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
