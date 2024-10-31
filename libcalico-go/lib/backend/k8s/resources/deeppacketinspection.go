// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package resources

import (
	"reflect"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DeepPacketInspectionResourceName = "DeepPacketInspections"
	DeepPacketInspectionCRDName      = "deeppacketinspections.crd.projectcalico.org"
)

func NewDeepPacketInspectionClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            DeepPacketInspectionCRDName,
		resource:        DeepPacketInspectionResourceName,
		description:     "Tigera Deep Packet Inspection",
		k8sResourceType: reflect.TypeOf(apiv3.DeepPacketInspection{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindDeepPacketInspection,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.DeepPacketInspectionList{}),
		resourceKind: apiv3.KindDeepPacketInspection,
		namespaced:   true,
	}
}
