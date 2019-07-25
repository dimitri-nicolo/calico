// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package resources

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const (
	ManagedClusterResourceName = "ManagedClusters"
	ManagedClusterCRDName      = "managedclusters.crd.projectcalico.org"
)

func NewManagedClusterClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            ManagedClusterCRDName,
		resource:        ManagedClusterResourceName,
		description:     "Tigera Managed Clusters",
		k8sResourceType: reflect.TypeOf(apiv3.ManagedCluster{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindManagedCluster,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.ManagedClusterList{}),
		resourceKind: apiv3.KindManagedCluster,
		namespaced:   false,
	}
}
