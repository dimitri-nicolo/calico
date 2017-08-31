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

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/pkg/api"

	"github.com/tigera/calico-k8sapiserver/pkg/apiserver"
)

// CalicoServerOptions contains the aggregation of configuration structs for
// the calico server. It contains everything needed to configure a basic API server.
// It is public so that integration tests can access it.
type CalicoServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	// DisableAuth disables delegating authentication and authorization for testing scenarios
	DisableAuth bool
	StopCh      <-chan struct{}
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
	if err := o.RecommendedOptions.Etcd.ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	if err := o.RecommendedOptions.SecureServing.ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	if !o.DisableAuth {
		if err := o.RecommendedOptions.Authentication.ApplyTo(genericConfig); err != nil {
			return nil, err
		}
		if err := o.RecommendedOptions.Authorization.ApplyTo(genericConfig); err != nil {
			return nil, err
		}
	} else {
		// always warn when auth is disabled, since this should only be used for testing
		glog.Infof("Authentication and authorization disabled for testing purposes")
	}
	if err := o.RecommendedOptions.Audit.ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	if err := o.RecommendedOptions.Features.ApplyTo(genericConfig); err != nil {
		return nil, err
	}
	config := apiserver.NewCalicoConfig(genericConfig)

	return config, nil
}
