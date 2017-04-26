/*
Copyright 2016 The Kubernetes Authors.

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

package v1alpha1

import (
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
)

// GroupName is the group name use in this package
const GroupName = "kubeadm.k8s.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, addDefaultingFuncs)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&MasterConfiguration{},
		&NodeConfiguration{},
		&ClusterInfo{},
		&v1.ListOptions{},
		&v1.DeleteOptions{},
		&metav1.ExportOptions{},
	)
	return nil
}

func (obj *MasterConfiguration) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }
func (obj *NodeConfiguration) GetObjectKind() schema.ObjectKind   { return &obj.TypeMeta }
func (obj *ClusterInfo) GetObjectKind() schema.ObjectKind         { return &obj.TypeMeta }
