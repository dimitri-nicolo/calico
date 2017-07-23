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

package v1

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
		Convert_v1_Endpoint_To_calico_Endpoint,
		Convert_calico_Endpoint_To_v1_Endpoint,
		Convert_v1_EndpointList_To_calico_EndpointList,
		Convert_calico_EndpointList_To_v1_EndpointList,
		Convert_v1_EndpointMeta_To_calico_EndpointMeta,
		Convert_calico_EndpointMeta_To_v1_EndpointMeta,
		Convert_v1_Policy_To_calico_Policy,
		Convert_calico_Policy_To_v1_Policy,
		Convert_v1_PolicyList_To_calico_PolicyList,
		Convert_calico_PolicyList_To_v1_PolicyList,
		Convert_v1_PolicyStatus_To_calico_PolicyStatus,
		Convert_calico_PolicyStatus_To_v1_PolicyStatus,
		Convert_v1_Tier_To_calico_Tier,
		Convert_calico_Tier_To_v1_Tier,
		Convert_v1_TierList_To_calico_TierList,
		Convert_calico_TierList_To_v1_TierList,
	)
}

func autoConvert_v1_Endpoint_To_calico_Endpoint(in *Endpoint, out *calico.Endpoint, s conversion.Scope) error {
	if err := Convert_v1_EndpointMeta_To_calico_EndpointMeta(&in.EndpointMeta, &out.EndpointMeta, s); err != nil {
		return err
	}
	out.Spec = in.Spec
	return nil
}

// Convert_v1_Endpoint_To_calico_Endpoint is an autogenerated conversion function.
func Convert_v1_Endpoint_To_calico_Endpoint(in *Endpoint, out *calico.Endpoint, s conversion.Scope) error {
	return autoConvert_v1_Endpoint_To_calico_Endpoint(in, out, s)
}

func autoConvert_calico_Endpoint_To_v1_Endpoint(in *calico.Endpoint, out *Endpoint, s conversion.Scope) error {
	if err := Convert_calico_EndpointMeta_To_v1_EndpointMeta(&in.EndpointMeta, &out.EndpointMeta, s); err != nil {
		return err
	}
	out.Spec = in.Spec
	return nil
}

// Convert_calico_Endpoint_To_v1_Endpoint is an autogenerated conversion function.
func Convert_calico_Endpoint_To_v1_Endpoint(in *calico.Endpoint, out *Endpoint, s conversion.Scope) error {
	return autoConvert_calico_Endpoint_To_v1_Endpoint(in, out, s)
}

func autoConvert_v1_EndpointList_To_calico_EndpointList(in *EndpointList, out *calico.EndpointList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]calico.Endpoint)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1_EndpointList_To_calico_EndpointList is an autogenerated conversion function.
func Convert_v1_EndpointList_To_calico_EndpointList(in *EndpointList, out *calico.EndpointList, s conversion.Scope) error {
	return autoConvert_v1_EndpointList_To_calico_EndpointList(in, out, s)
}

func autoConvert_calico_EndpointList_To_v1_EndpointList(in *calico.EndpointList, out *EndpointList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items == nil {
		out.Items = make([]Endpoint, 0)
	} else {
		out.Items = *(*[]Endpoint)(unsafe.Pointer(&in.Items))
	}
	return nil
}

// Convert_calico_EndpointList_To_v1_EndpointList is an autogenerated conversion function.
func Convert_calico_EndpointList_To_v1_EndpointList(in *calico.EndpointList, out *EndpointList, s conversion.Scope) error {
	return autoConvert_calico_EndpointList_To_v1_EndpointList(in, out, s)
}

func autoConvert_v1_EndpointMeta_To_calico_EndpointMeta(in *EndpointMeta, out *calico.EndpointMeta, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.WorkloadEndpointMetadata = in.WorkloadEndpointMetadata
	return nil
}

// Convert_v1_EndpointMeta_To_calico_EndpointMeta is an autogenerated conversion function.
func Convert_v1_EndpointMeta_To_calico_EndpointMeta(in *EndpointMeta, out *calico.EndpointMeta, s conversion.Scope) error {
	return autoConvert_v1_EndpointMeta_To_calico_EndpointMeta(in, out, s)
}

func autoConvert_calico_EndpointMeta_To_v1_EndpointMeta(in *calico.EndpointMeta, out *EndpointMeta, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.WorkloadEndpointMetadata = in.WorkloadEndpointMetadata
	return nil
}

// Convert_calico_EndpointMeta_To_v1_EndpointMeta is an autogenerated conversion function.
func Convert_calico_EndpointMeta_To_v1_EndpointMeta(in *calico.EndpointMeta, out *EndpointMeta, s conversion.Scope) error {
	return autoConvert_calico_EndpointMeta_To_v1_EndpointMeta(in, out, s)
}

func autoConvert_v1_Policy_To_calico_Policy(in *Policy, out *calico.Policy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	if err := Convert_v1_PolicyStatus_To_calico_PolicyStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1_Policy_To_calico_Policy is an autogenerated conversion function.
func Convert_v1_Policy_To_calico_Policy(in *Policy, out *calico.Policy, s conversion.Scope) error {
	return autoConvert_v1_Policy_To_calico_Policy(in, out, s)
}

func autoConvert_calico_Policy_To_v1_Policy(in *calico.Policy, out *Policy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	if err := Convert_calico_PolicyStatus_To_v1_PolicyStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_calico_Policy_To_v1_Policy is an autogenerated conversion function.
func Convert_calico_Policy_To_v1_Policy(in *calico.Policy, out *Policy, s conversion.Scope) error {
	return autoConvert_calico_Policy_To_v1_Policy(in, out, s)
}

func autoConvert_v1_PolicyList_To_calico_PolicyList(in *PolicyList, out *calico.PolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]calico.Policy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1_PolicyList_To_calico_PolicyList is an autogenerated conversion function.
func Convert_v1_PolicyList_To_calico_PolicyList(in *PolicyList, out *calico.PolicyList, s conversion.Scope) error {
	return autoConvert_v1_PolicyList_To_calico_PolicyList(in, out, s)
}

func autoConvert_calico_PolicyList_To_v1_PolicyList(in *calico.PolicyList, out *PolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items == nil {
		out.Items = make([]Policy, 0)
	} else {
		out.Items = *(*[]Policy)(unsafe.Pointer(&in.Items))
	}
	return nil
}

// Convert_calico_PolicyList_To_v1_PolicyList is an autogenerated conversion function.
func Convert_calico_PolicyList_To_v1_PolicyList(in *calico.PolicyList, out *PolicyList, s conversion.Scope) error {
	return autoConvert_calico_PolicyList_To_v1_PolicyList(in, out, s)
}

func autoConvert_v1_PolicyStatus_To_calico_PolicyStatus(in *PolicyStatus, out *calico.PolicyStatus, s conversion.Scope) error {
	return nil
}

// Convert_v1_PolicyStatus_To_calico_PolicyStatus is an autogenerated conversion function.
func Convert_v1_PolicyStatus_To_calico_PolicyStatus(in *PolicyStatus, out *calico.PolicyStatus, s conversion.Scope) error {
	return autoConvert_v1_PolicyStatus_To_calico_PolicyStatus(in, out, s)
}

func autoConvert_calico_PolicyStatus_To_v1_PolicyStatus(in *calico.PolicyStatus, out *PolicyStatus, s conversion.Scope) error {
	return nil
}

// Convert_calico_PolicyStatus_To_v1_PolicyStatus is an autogenerated conversion function.
func Convert_calico_PolicyStatus_To_v1_PolicyStatus(in *calico.PolicyStatus, out *PolicyStatus, s conversion.Scope) error {
	return autoConvert_calico_PolicyStatus_To_v1_PolicyStatus(in, out, s)
}

func autoConvert_v1_Tier_To_calico_Tier(in *Tier, out *calico.Tier, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v1_Tier_To_calico_Tier is an autogenerated conversion function.
func Convert_v1_Tier_To_calico_Tier(in *Tier, out *calico.Tier, s conversion.Scope) error {
	return autoConvert_v1_Tier_To_calico_Tier(in, out, s)
}

func autoConvert_calico_Tier_To_v1_Tier(in *calico.Tier, out *Tier, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_calico_Tier_To_v1_Tier is an autogenerated conversion function.
func Convert_calico_Tier_To_v1_Tier(in *calico.Tier, out *Tier, s conversion.Scope) error {
	return autoConvert_calico_Tier_To_v1_Tier(in, out, s)
}

func autoConvert_v1_TierList_To_calico_TierList(in *TierList, out *calico.TierList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]calico.Tier)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1_TierList_To_calico_TierList is an autogenerated conversion function.
func Convert_v1_TierList_To_calico_TierList(in *TierList, out *calico.TierList, s conversion.Scope) error {
	return autoConvert_v1_TierList_To_calico_TierList(in, out, s)
}

func autoConvert_calico_TierList_To_v1_TierList(in *calico.TierList, out *TierList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items == nil {
		out.Items = make([]Tier, 0)
	} else {
		out.Items = *(*[]Tier)(unsafe.Pointer(&in.Items))
	}
	return nil
}

// Convert_calico_TierList_To_v1_TierList is an autogenerated conversion function.
func Convert_calico_TierList_To_v1_TierList(in *calico.TierList, out *TierList, s conversion.Scope) error {
	return autoConvert_calico_TierList_To_v1_TierList(in, out, s)
}
