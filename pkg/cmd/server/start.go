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

package server

import (
	"fmt"
	"io"
	"net"

	"github.com/spf13/cobra"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico/v1alpha1"
	"github.com/tigera/calico-k8sapiserver/pkg/apiserver"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
)

const defaultEtcdPathPrefix = "/registry/calico.kubernetes.io"

type CalicoServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions

	StdOut io.Writer
	StdErr io.Writer
}

func NewCalicoServerOptions(out, errOut io.Writer) *CalicoServerOptions {
	o := &CalicoServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(defaultEtcdPathPrefix, apiserver.Scheme, apiserver.Codecs.LegacyCodec(v1alpha1.SchemeGroupVersion)),

		StdOut: out,
		StdErr: errOut,
	}

	return o
}

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartCalicoServer(out, errOut io.Writer, stopCh <-chan struct{}) *cobra.Command {
	o := NewCalicoServerOptions(out, errOut)

	cmd := &cobra.Command{
		Short: "Launch a calico API server",
		Long:  "Launch a calico API server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunCalicoServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.RecommendedOptions.AddFlags(flags)

	return cmd
}

func (o CalicoServerOptions) Validate(args []string) error {
	return nil
}

func (o *CalicoServerOptions) Complete() error {
	return nil
}

func (o CalicoServerOptions) Config() (*apiserver.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewConfig(apiserver.Codecs)
	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
	}
	return config, nil
}

func (o CalicoServerOptions) RunCalicoServer(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}
