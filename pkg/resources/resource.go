// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources

import (
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apiv3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
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
