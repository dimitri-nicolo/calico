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

package server

import (
	"fmt"
	"net"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/pkg/api"

	"strings"

	"github.com/tigera/calico-k8sapiserver/pkg/apiserver"
)

// CalicoServerOptions contains the aggregation of configuration structs for
// the calico server. It contains everything needed to configure a basic API server.
// It is public so that integration tests can access it.
type CalicoServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	StopCh             <-chan struct{}
}

func (s *CalicoServerOptions) addFlags(flags *pflag.FlagSet) {
	s.RecommendedOptions.AddFlags(flags)
}

func (o CalicoServerOptions) Validate(args []string) error {
	return nil
}

func (o *CalicoServerOptions) Complete() error {
	return nil
}

func (o *CalicoServerOptions) Config() (apiserver.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	genericConfig := genericapiserver.NewConfig(api.Codecs)
	if err := o.RecommendedOptions.ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	// Overload RESTOptionsGetter to reflect ResourcePrefix
	genericConfig.RESTOptionsGetter = CalicoRESTOptionHelper{o.RecommendedOptions.Etcd}
	config := apiserver.NewCalicoConfig(genericConfig)

	return config, nil
}

type CalicoRESTOptionHelper struct {
	Options *genericoptions.EtcdOptions
}

func (f CalicoRESTOptionHelper) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	ret := generic.RESTOptions{
		StorageConfig:           &f.Options.StorageConfig,
		Decorator:               generic.UndecoratedStorage,
		EnableGarbageCollection: f.Options.EnableGarbageCollection,
		DeleteCollectionWorkers: f.Options.DeleteCollectionWorkers,
	}
	if f.Options.EnableWatchCache {
		ret.Decorator = genericregistry.StorageWithCacher
	}
	if strings.Compare(resource.Resource, "policies") == 0 {
		ret.ResourcePrefix = "/policy/tier"
	}
	return ret, nil
}
