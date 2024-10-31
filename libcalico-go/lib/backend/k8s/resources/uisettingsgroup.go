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
	UISettingsGroupResourceName = "UISettingsGroups"
	UISettingsGroupCRDName      = "uisettingsgroups.crd.projectcalico.org"
)

func NewUISettingsGroupClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            UISettingsGroupCRDName,
		resource:        UISettingsGroupResourceName,
		description:     "Calico UI Settings Group",
		k8sResourceType: reflect.TypeOf(apiv3.UISettingsGroup{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindUISettingsGroup,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.UISettingsGroupList{}),
		resourceKind: apiv3.KindUISettingsGroup,
	}
}
