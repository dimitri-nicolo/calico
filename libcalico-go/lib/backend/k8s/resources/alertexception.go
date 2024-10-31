// Copyright (c) 2022 Tigera, Inc. All rights reserved.

package resources

import (
	"reflect"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	AlertExceptionResourceName = "AlertExceptions"
	AlertExceptionCRDName      = "alertexceptions.crd.projectcalico.org"
)

func NewAlertExceptionClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            AlertExceptionCRDName,
		resource:        AlertExceptionResourceName,
		description:     "Tigera Alert Exceptions",
		k8sResourceType: reflect.TypeOf(apiv3.AlertException{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindAlertException,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.AlertExceptionList{}),
		resourceKind: apiv3.KindAlertException,
		namespaced:   false,
	}
}
