/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package calico

import (
	"strconv"
	"strings"

	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/etcd"
)

// APIObjectVersioner implements versioning and extracting etcd node information
// for objects that have an embedded ObjectMeta or ListMeta field.
type APIObjectVersioner struct {
	*etcd.APIObjectVersioner
}

// ObjectResourceVersion implements Versioner
func (a APIObjectVersioner) ObjectResourceVersion(obj runtime.Object) (uint64, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	version := accessor.GetResourceVersion()
	if len(version) == 0 {
		return 0, nil
	}
	if strings.ContainsRune(version, '/') == true {
		conv := conversion.Converter{}
		crdNPRev, k8sNPRev, _ := conv.SplitNetworkPolicyRevision(version)
		if crdNPRev == "" && k8sNPRev != "" {
			reason := "kubernetes network policies must be managed through the kubernetes API"
			return 0, errors.NewBadRequest(reason)
		}
		version = crdNPRev
	}
	return strconv.ParseUint(version, 10, 64)
}
