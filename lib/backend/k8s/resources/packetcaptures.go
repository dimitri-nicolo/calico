// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package resources

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

const (
	PacketCaptureResourceName = "PacketCaptures"
	PacketCaptureCRDName      = "packetcaptures.crd.projectcalico.org"
)

func NewPacketCaptureClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            PacketCaptureCRDName,
		resource:        PacketCaptureResourceName,
		description:     "Tigera Packet Captures",
		k8sResourceType: reflect.TypeOf(apiv3.PacketCapture{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindPacketCapture,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.PacketCaptureList{}),
		resourceKind: apiv3.KindPacketCapture,
		namespaced:   true,
	}
}
