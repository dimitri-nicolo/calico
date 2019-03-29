// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Resource interface {
	runtime.Object
	metav1.ObjectMetaAccessor
}

type ResourceList interface {
	runtime.Object
	metav1.ListMetaAccessor
}

type NameNamespace struct {
	Name      string
	Namespace string
}

type ResourceID struct {
	schema.GroupVersionKind
	NameNamespace
}

func (r ResourceID) String() string {
	if r.Namespace == "" {
		return r.GroupVersionKind.GroupVersion().String() + "/" + r.Kind + "/" + r.Name
	}
	return r.GroupVersionKind.GroupVersion().String() + "/" + r.Kind + "/" + r.Namespace + "/" + r.Name
}

func GetResourceID(r Resource) ResourceID {
	return ResourceID{
		GroupVersionKind: r.GetObjectKind().GroupVersionKind(),
		NameNamespace:    GetNameNamespace(r),
	}
}

func GetNameNamespace(r Resource) NameNamespace {
	return NameNamespace{
		Name:      r.GetObjectMeta().GetName(),
		Namespace: r.GetObjectMeta().GetNamespace(),
	}
}
