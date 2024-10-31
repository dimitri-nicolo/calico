// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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
	"strconv"
	"strings"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Resource interface {
	runtime.Object
	metav1.ObjectMetaAccessor
}

type ResourceList interface {
	runtime.Object
	metav1.ListMetaAccessor
}

func GetResourceID(r Resource) apiv3.ResourceID {
	//TODO(rlb): We are not yet including UID.
	return apiv3.ResourceID{
		TypeMeta:  GetTypeMeta(r),
		Name:      r.GetObjectMeta().GetName(),
		Namespace: r.GetObjectMeta().GetNamespace(),
	}
}

// GetResourceVersion extracts the resource version from a resource and returns it as an int.
// Split on / for the possible calico resource that provides two resource versions.
// We can safely access element 0 since strings.Split will always return an array of length 1.
func GetResourceVersion(r Resource) (int, error) {
	return strconv.Atoi(strings.Split(r.GetObjectMeta().GetResourceVersion(), "/")[0])
}
