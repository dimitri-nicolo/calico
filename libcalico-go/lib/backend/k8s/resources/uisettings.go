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
	UISettingsResourceName = "UISettings"
	UISettingsCRDName      = "UISettings.crd.projectcalico.org"
)

func NewUISettingsClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            UISettingsCRDName,
		resource:        UISettingsResourceName,
		description:     "Calico UI Settings",
		k8sResourceType: reflect.TypeOf(apiv3.UISettings{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindUISettings,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.UISettingsList{}),
		resourceKind: apiv3.KindUISettings,
	}
}
