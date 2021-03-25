// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	etcd "k8s.io/apiserver/pkg/storage/etcd3"

	"github.com/projectcalico/libcalico-go/lib/backend/k8s/conversion"
)

// NetworkPolicyAPIObjectVersioner implements versioning and extracting etcd node information
// for Calico NetworkPolicy resources.
type NetworkPolicyAPIObjectVersioner struct {
	*etcd.APIObjectVersioner
}

// ObjectResourceVersion implements Versioner
func (a NetworkPolicyAPIObjectVersioner) ObjectResourceVersion(obj runtime.Object) (uint64, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	version := accessor.GetResourceVersion()
	if len(version) == 0 {
		return 0, nil
	}
	if strings.ContainsRune(version, '/') == true {
		conv := conversion.NewConverter()
		crdNPRev, k8sNPRev, _ := conv.SplitNetworkPolicyRevision(version)
		if crdNPRev == "" && k8sNPRev != "" {
			reason := "kubernetes network policies must be managed through the kubernetes API"
			return 0, errors.NewBadRequest(reason)
		}
		version = crdNPRev
	}
	return strconv.ParseUint(version, 10, 64)
}

// ProfileAPIObjectVersioner implements versioning and extracting etcd node information
// for Calico Profile resources.
type ProfileAPIObjectVersioner struct {
	*etcd.APIObjectVersioner
}

// ObjectResourceVersion implements Versioner
func (a ProfileAPIObjectVersioner) ObjectResourceVersion(obj runtime.Object) (uint64, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	version := accessor.GetResourceVersion()
	if len(version) == 0 {
		return 0, nil
	}
	if strings.ContainsRune(version, '/') == true {
		// k8s api machinery expects the resource version to be a number, so we have to covert our two-part version
		// into a single number.  Just use the first segment which refers to the namespace.  Note that libcalico-go
		// accepts a single section and attributes it to the namespace, so this will work.  This should be ok given
		// profiles are not manageable through the API server when backed by kdd - this may result in some aggressive
		// list/watch behavior since we always end up listing/watching with no service account revision.
		conv := conversion.NewConverter()
		nsRev, saRev, _ := conv.SplitProfileRevision(version)
		if nsRev == "" && saRev != "" {
			reason := "profiles cannot be managed directly"
			return 0, errors.NewBadRequest(reason)
		}
		version = nsRev
	}
	return strconv.ParseUint(version, 10, 64)
}
