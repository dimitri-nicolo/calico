// Copyright (c) 2022 Tigera, Inc. All rights reserved.

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

package resources

import (
	"reflect"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	BGPFilterResourceName = "BGPFilters"
	BGPFilterCRDName      = "BGPFilters.crd.projectcalico.org"
)

func NewBGPFilterClient(c *kubernetes.Clientset, r *rest.RESTClient) K8sResourceClient {
	return &customK8sResourceClient{
		clientSet:       c,
		restClient:      r,
		name:            BGPFilterCRDName,
		resource:        BGPFilterResourceName,
		description:     "BGPFilter",
		k8sResourceType: reflect.TypeOf(apiv3.BGPFilter{}),
		k8sResourceTypeMeta: metav1.TypeMeta{
			Kind:       apiv3.KindBGPFilter,
			APIVersion: apiv3.GroupVersionCurrent,
		},
		k8sListType:  reflect.TypeOf(apiv3.BGPFilterList{}),
		resourceKind: apiv3.KindBGPFilter,
	}
}
