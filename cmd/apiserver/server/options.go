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
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	k8sopenapi "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/klog"

	"github.com/tigera/apiserver/pkg/apiserver"
	"github.com/tigera/apiserver/pkg/openapi"
)

// CalicoServerOptions contains the aggregation of configuration structs for
// the calico server. It contains everything needed to configure a basic API server.
// It is public so that integration tests can access it.
type CalicoServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions

	// DisableAuth disables delegating authentication and authorization for testing scenarios
	DisableAuth bool

	// Enable Admission Controller support.
	EnableAdmissionController bool

	// Path to CA cert and key required for managed cluster creation
	// The parameters below can only be used in conjunction with
	// EnableManagedClustersCreateAPI flag
	ManagedClustersCACertPath      string
	ManagedClustersCAKeyPath       string
	EnableManagedClustersCreateAPI bool

	// Use this to populate the managementClusterAddr inside the managementClusterConnection CR.
	ManagementClusterAddr string

	StopCh <-chan struct{}
}

func (s *CalicoServerOptions) addFlags(flags *pflag.FlagSet) {
	s.RecommendedOptions.AddFlags(flags)
	flags.BoolVar(&s.EnableAdmissionController, "enable-admission-controller-support", s.EnableAdmissionController,
		"If true, admission controller hooks will be enabled.")
	flags.BoolVar(&s.EnableManagedClustersCreateAPI, "enable-managed-clusters-create-api", false,
		"If true, --set-managed-clusters-ca-cert and --set-managed-clusters-ca-key will be evaluated.")
	flags.StringVar(&s.ManagedClustersCACertPath, "set-managed-clusters-ca-cert",
		"/code/apiserver.local.config/multicluster/certificates/cert",
		"If set, the path to the CA cert will be used to generate managed clusters")
	flags.StringVar(&s.ManagedClustersCAKeyPath, "set-managed-clusters-ca-key",
		"/code/apiserver.local.config/multicluster/certificates/key",
		"If set, the path to the CA key will be used to generate managed clusters")
	flags.StringVar(&s.ManagementClusterAddr, "managementClusterAddr",
		"<your-management-cluster-address>",
		"If set, manifests created for new managed clusters will use this value.")
}

func (o CalicoServerOptions) Validate(args []string) error {
	errors := []error{}
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

func (o *CalicoServerOptions) Complete() error {
	return nil
}

func (o *CalicoServerOptions) Config() (*apiserver.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(apiserver.Codecs)
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(openapi.GetOpenAPIDefinitions, k8sopenapi.NewDefinitionNamer(apiserver.Scheme))
	if serverConfig.OpenAPIConfig.Info == nil {
		serverConfig.OpenAPIConfig.Info = &spec.Info{}
	}
	if serverConfig.OpenAPIConfig.Info.Version == "" {
		if serverConfig.Version != nil {
			serverConfig.OpenAPIConfig.Info.Version = strings.Split(serverConfig.Version.String(), "-")[0]
		} else {
			serverConfig.OpenAPIConfig.Info.Version = "unversioned"
		}
	}

	if err := o.RecommendedOptions.Etcd.ApplyTo(&serverConfig.Config); err != nil {
		return nil, err
	}
	o.RecommendedOptions.Etcd.StorageConfig.Paging = utilfeature.DefaultFeatureGate.Enabled(features.APIListChunking)
	if err := o.RecommendedOptions.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	//Explicitly setting cipher suites in order to remove deprecated ones:
	//- TLS_RSA --- lack perfect forward secrecy
	//- 3DES --- widely considered to be too weak
	//Order matters as indicates preference for the cipher in the selection algorithm. Also, some suites (TLS_ECDHE_ECDSA_WITH_RC4_128_SHA
	//for instance, for full list refer to golang.org/x/net/http2) are blacklisted by HTTP/2 spec and MUST be placed after the HTTP/2-approved
	//cipher suites. Not doing that might cause client to be given an unapproved suite and reject the connection.
	cipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305, tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA, tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA}
	serverConfig.SecureServing.CipherSuites = cipherSuites

	if !o.DisableAuth {
		if err := o.RecommendedOptions.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, serverConfig.OpenAPIConfig); err != nil {
			return nil, err
		}
		if err := o.RecommendedOptions.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
			return nil, err
		}
	} else {
		// always warn when auth is disabled, since this should only be used for testing
		klog.Infof("Authentication and authorization disabled for testing purposes")
	}

	if err := o.RecommendedOptions.Audit.ApplyTo(&serverConfig.Config, nil, nil, o.RecommendedOptions.ProcessInfo, nil); err != nil {
		return nil, err
	}
	if err := o.RecommendedOptions.Features.ApplyTo(&serverConfig.Config); err != nil {
		return nil, err
	}

	if err := o.RecommendedOptions.CoreAPI.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	if initializers, err := o.RecommendedOptions.ExtraAdmissionInitializers(serverConfig); err != nil {
		return nil, err
	} else if err := o.RecommendedOptions.Admission.ApplyTo(&serverConfig.Config, serverConfig.SharedInformerFactory, serverConfig.ClientConfig, o.RecommendedOptions.FeatureGate, initializers...); err != nil {
		return nil, err
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig: apiserver.ExtraConfig{
			ManagedClustersCACert:          o.ManagedClustersCACertPath,
			ManagedClustersCAKey:           o.ManagedClustersCAKeyPath,
			EnableManagedClustersCreateAPI: o.EnableManagedClustersCreateAPI,
			ManagementClusterAddr:          o.ManagementClusterAddr,
			KubernetesAPIServerConfig:      serverConfig.ClientConfig,
		},
	}

	return config, nil
}
