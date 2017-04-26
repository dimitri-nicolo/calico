// +build !ignore_autogenerated

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

// This file was autogenerated by defaulter-gen. Do not edit it manually!

package v1alpha1

import (
	runtime "k8s.io/client-go/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&ClusterRoleBinding{}, func(obj interface{}) { SetObjectDefaults_ClusterRoleBinding(obj.(*ClusterRoleBinding)) })
	scheme.AddTypeDefaultingFunc(&ClusterRoleBindingList{}, func(obj interface{}) { SetObjectDefaults_ClusterRoleBindingList(obj.(*ClusterRoleBindingList)) })
	scheme.AddTypeDefaultingFunc(&RoleBinding{}, func(obj interface{}) { SetObjectDefaults_RoleBinding(obj.(*RoleBinding)) })
	scheme.AddTypeDefaultingFunc(&RoleBindingList{}, func(obj interface{}) { SetObjectDefaults_RoleBindingList(obj.(*RoleBindingList)) })
	return nil
}

func SetObjectDefaults_ClusterRoleBinding(in *ClusterRoleBinding) {
	SetDefaults_ClusterRoleBinding(in)
}

func SetObjectDefaults_ClusterRoleBindingList(in *ClusterRoleBindingList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_ClusterRoleBinding(a)
	}
}

func SetObjectDefaults_RoleBinding(in *RoleBinding) {
	SetDefaults_RoleBinding(in)
}

func SetObjectDefaults_RoleBindingList(in *RoleBindingList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_RoleBinding(a)
	}
}
