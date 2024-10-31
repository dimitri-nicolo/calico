// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package resources

import (
	"reflect"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	EgressGatewayPolicyResourceName = "EgressGatewayPolicies"
	EgressGatewayPolicyCRDName      = "egressgatewaypolicies.crd.projectcalico.org"
)

func NewEgressPolicyClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            EgressGatewayPolicyCRDName,
		resource:        EgressGatewayPolicyResourceName,
		description:     "EgressGatewayPolicy",
		k8sResourceType: reflect.TypeOf(apiv3.EgressGatewayPolicy{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindEgressGatewayPolicy,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.EgressGatewayPolicyList{}),
		resourceKind: apiv3.KindEgressGatewayPolicy,
	}
}
