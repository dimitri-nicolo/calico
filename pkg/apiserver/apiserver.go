// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package apiserver

import (
	"github.com/tigera/apiserver/pkg/helpers"
	calicorest "github.com/tigera/apiserver/pkg/registry/projectcalico/rest"
	"github.com/tigera/apiserver/pkg/storage/calico"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/tigera/apiserver/pkg/apis/projectcalico"
	"github.com/tigera/apiserver/pkg/apis/projectcalico/install"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	install.Install(Scheme)

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
	ManagedClustersCACert          string
	ManagedClustersCAKey           string
	EnableManagedClustersCreateAPI bool
	ManagementClusterAddr          string
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

// New returns a new instance of ProjectCalicoServer from the given config.
func (c completedConfig) New() (*ProjectCalicoServer, error) {
	genericServer, err := c.GenericConfig.New("apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &ProjectCalicoServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(projectcalico.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	//apiGroupInfo.OptionsExternalVersion = &schema.GroupVersion{Version: "v3"}

	// TODO: Make the storage type configurable
	calicostore := calicorest.RESTStorageProvider{StorageType: "calico"}

	var res *calico.ManagedClusterResources
	if c.ExtraConfig.EnableManagedClustersCreateAPI {
		cert, key, err := helpers.ReadCredentials(c.ExtraConfig.ManagedClustersCACert, c.ExtraConfig.ManagedClustersCAKey)
		if err != nil {
			return nil, err
		}
		x509Cert, rsaKey, err := helpers.DecodeCertAndKey(cert, key)
		if err != nil {
			return nil, err
		}
		res = &calico.ManagedClusterResources{
			CACert:                x509Cert,
			CAKey:                 rsaKey,
			ManagementClusterAddr: c.ExtraConfig.ManagementClusterAddr,
		}
	}

	apiGroupInfo.VersionedResourcesStorageMap["v3"], err = calicostore.NewV3Storage(Scheme, c.GenericConfig.RESTOptionsGetter, c.GenericConfig.Authorization.Authorizer, res)
	if err != nil {
		return nil, err
	}

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
