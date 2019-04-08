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
	return r.GroupVersion().String() + "/" + r.Kind + "/" + r.NameNamespace.String()
}

func GetResourceID(r Resource) ResourceID {
	return ResourceID{
		GroupVersionKind: r.GetObjectKind().GroupVersionKind(),
		NameNamespace:    GetNameNamespace(r),
	}
}

func (nn NameNamespace) String() string {
	if nn.Namespace == "" {
		return nn.Name
	}
	return nn.Namespace + "/" + nn.Name
}

func GetNameNamespace(r Resource) NameNamespace {
	return NameNamespace{
		Name:      r.GetObjectMeta().GetName(),
		Namespace: r.GetObjectMeta().GetNamespace(),
	}
}
