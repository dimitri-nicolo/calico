// Copyright (c) 2017 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KindRemoteClusterConfiguration     = "RemoteClusterConfiguration"
	KindRemoteClusterConfigurationList = "RemoteClusterConfigurationList"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RemoteClusterConfiguration contains the configuration for remote clusters.
type RemoteClusterConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the RemoteClusterConfiguration.
	Spec RemoteClusterConfigurationSpec `json:"spec,omitempty"`
}

// It's desirable to keep the list of things configurable here in sync with the other mechanism in apiconfig.go

// RemoteClusterConfigurationSpec contains the values of describing the cluster.
type RemoteClusterConfigurationSpec struct {
	// Indicates the datastore to use. If unspecified, defaults to etcdv3
	DatastoreType string `json:"datastoreType" validate:"datastoreType"`
	// A comma separated list of etcd endpoints. Valid if DatastoreType is etcdv3.
	EtcdEndpoints            string `json:"etcdEndpoints,omitempty" validate:"omitempty,etcdEndpoints"`
	// User name for RBAC. Valid if DatastoreType is etcdv3.
	EtcdUsername             string `json:"etcdUsername,omitempty" validate:"omitempty"`
	// Password for the given user name. Valid if DatastoreType is etcdv3.
	EtcdPassword             string `json:"etcdPassword,omitempty" validate:"omitempty"`
	// Path to the etcd key file. Valid if DatastoreType is etcdv3.
	EtcdKeyFile              string `json:"etcdKeyFile,omitempty" validate:"omitempty,file"`
	// Path to the etcd client certificate. Valid if DatastoreType is etcdv3.
	EtcdCertFile             string `json:"etcdCertFile,omitempty" validate:"omitempty,file"`
	// Path to the etcd Certificate Authority file. Valid if DatastoreType is etcdv3.
	EtcdCACertFile           string `json:"etcdCACertFile,omitempty" validate:"omitempty,file"`
	// When using the Kubernetes datastore, the location of a kubeconfig file. Valid if DatastoreType is kubernetes.
	Kubeconfig               string `json:"kubeconfig,omitempty" validate:"omitempty,file"`
	// Location of the Kubernetes API. Not required if using kubeconfig. Valid if DatastoreType is kubernetes.
	K8sAPIEndpoint           string `json:"k8sAPIEndpoint,omitempty" validate:"omitempty,k8sEndpoint"`
	// Location of a client key for accessing the Kubernetes API. Valid if DatastoreType is kubernetes.
	K8sKeyFile               string `json:"k8sKeyFile,omitempty" validate:"omitempty,file"`
	// Location of a client certificate for accessing the Kubernetes API. Valid if DatastoreType is kubernetes.
	K8sCertFile              string `json:"k8sCertFile,omitempty" validate:"omitempty,file"`
	// Location of a CA for accessing the Kubernetes API. Valid if DatastoreType is kubernetes.
	K8sCAFile                string `json:"k8sCAFile,omitempty" validate:"omitempty,file"`
	// Token to be used for accessing the Kubernetes API. Valid if DatastoreType is kubernetes.
	K8sAPIToken              string `json:"k8sAPIToken,omitempty" validate:"omitempty"`
	K8sInsecureSkipTLSVerify bool   `json:"k8sInsecureSkipTLSVerify,omitempty" validate:"omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RemoteClusterConfigurationList contains a list of RemoteClusterConfiguration resources
type RemoteClusterConfigurationList struct {
	metav1.TypeMeta                    `json:",inline"`
	metav1.ListMeta                    `json:"metadata"`
	Items []RemoteClusterConfiguration `json:"items"`
}

// New RemoteClusterConfiguration creates a new (zeroed) RemoteClusterConfiguration struct with the TypeMetadata
// initialized to the current version.
func NewRemoteClusterConfiguration() *RemoteClusterConfiguration {
	return &RemoteClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindRemoteClusterConfiguration,
			APIVersion: GroupVersionCurrent,
		},
	}
}

// NewRemoteClusterConfigurationList creates a new (zeroed) RemoteClusterConfigurationList struct with the TypeMetadata
// initialized to the current version.
func NewRemoteClusterConfigurationList() *RemoteClusterConfigurationList {
	return &RemoteClusterConfigurationList{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindRemoteClusterConfigurationList,
			APIVersion: GroupVersionCurrent,
		},
	}
}
