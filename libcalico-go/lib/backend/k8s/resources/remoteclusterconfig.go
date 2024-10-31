// Copyright (c) 2018 Tigera, Inc. All rights reserved.

package resources

import (
	"reflect"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	RemoteClusterConfigurationResourceName = "RemoteClusterConfigurations"
	RemoteClusterConfigurationCRDName      = "remoteclusterconfigurations.crd.projectcalico.org"
)

func NewRemoteClusterConfigurationClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            RemoteClusterConfigurationCRDName,
		resource:        RemoteClusterConfigurationResourceName,
		description:     "Calico Remote Cluster Configuration",
		k8sResourceType: reflect.TypeOf(apiv3.RemoteClusterConfiguration{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindRemoteClusterConfiguration,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.RemoteClusterConfigurationList{}),
		resourceKind: apiv3.KindRemoteClusterConfiguration,
	}
}
