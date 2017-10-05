// +build !ignore_autogenerated

/*
Copyright 2017 The Kubernetes Authors.

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

// This file was autogenerated by conversion-gen. Do not edit it manually!

package v2

import (
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	unsafe "unsafe"
)

func init() {
	SchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedConversionFuncs(
		Convert_v2_NetworkPolicy_To_calico_NetworkPolicy,
		Convert_calico_NetworkPolicy_To_v2_NetworkPolicy,
		Convert_v2_NetworkPolicyList_To_calico_NetworkPolicyList,
		Convert_calico_NetworkPolicyList_To_v2_NetworkPolicyList,
	)
}

func autoConvert_v2_NetworkPolicy_To_calico_NetworkPolicy(in *NetworkPolicy, out *calico.NetworkPolicy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v2_NetworkPolicy_To_calico_NetworkPolicy is an autogenerated conversion function.
func Convert_v2_NetworkPolicy_To_calico_NetworkPolicy(in *NetworkPolicy, out *calico.NetworkPolicy, s conversion.Scope) error {
	return autoConvert_v2_NetworkPolicy_To_calico_NetworkPolicy(in, out, s)
}

func autoConvert_calico_NetworkPolicy_To_v2_NetworkPolicy(in *calico.NetworkPolicy, out *NetworkPolicy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_calico_NetworkPolicy_To_v2_NetworkPolicy is an autogenerated conversion function.
func Convert_calico_NetworkPolicy_To_v2_NetworkPolicy(in *calico.NetworkPolicy, out *NetworkPolicy, s conversion.Scope) error {
	return autoConvert_calico_NetworkPolicy_To_v2_NetworkPolicy(in, out, s)
}

func autoConvert_v2_NetworkPolicyList_To_calico_NetworkPolicyList(in *NetworkPolicyList, out *calico.NetworkPolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]calico.NetworkPolicy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v2_NetworkPolicyList_To_calico_NetworkPolicyList is an autogenerated conversion function.
func Convert_v2_NetworkPolicyList_To_calico_NetworkPolicyList(in *NetworkPolicyList, out *calico.NetworkPolicyList, s conversion.Scope) error {
	return autoConvert_v2_NetworkPolicyList_To_calico_NetworkPolicyList(in, out, s)
}

func autoConvert_calico_NetworkPolicyList_To_v2_NetworkPolicyList(in *calico.NetworkPolicyList, out *NetworkPolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items == nil {
		out.Items = make([]NetworkPolicy, 0)
	} else {
		out.Items = *(*[]NetworkPolicy)(unsafe.Pointer(&in.Items))
	}
	return nil
}

// Convert_calico_NetworkPolicyList_To_v2_NetworkPolicyList is an autogenerated conversion function.
func Convert_calico_NetworkPolicyList_To_v2_NetworkPolicyList(in *calico.NetworkPolicyList, out *NetworkPolicyList, s conversion.Scope) error {
	return autoConvert_calico_NetworkPolicyList_To_v2_NetworkPolicyList(in, out, s)
}
