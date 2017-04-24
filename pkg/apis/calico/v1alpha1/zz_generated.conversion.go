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

package v1alpha1

import (
	numorstring "github.com/projectcalico/libcalico-go/lib/numorstring"
	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	net "net"
	unsafe "unsafe"
)

func init() {
	SchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedConversionFuncs(
		Convert_v1alpha1_EntityRule_To_calico_EntityRule,
		Convert_calico_EntityRule_To_v1alpha1_EntityRule,
		Convert_v1alpha1_ICMPFields_To_calico_ICMPFields,
		Convert_calico_ICMPFields_To_v1alpha1_ICMPFields,
		Convert_v1alpha1_Policy_To_calico_Policy,
		Convert_calico_Policy_To_v1alpha1_Policy,
		Convert_v1alpha1_PolicyList_To_calico_PolicyList,
		Convert_calico_PolicyList_To_v1alpha1_PolicyList,
		Convert_v1alpha1_PolicySpec_To_calico_PolicySpec,
		Convert_calico_PolicySpec_To_v1alpha1_PolicySpec,
		Convert_v1alpha1_PolicyStatus_To_calico_PolicyStatus,
		Convert_calico_PolicyStatus_To_v1alpha1_PolicyStatus,
		Convert_v1alpha1_Rule_To_calico_Rule,
		Convert_calico_Rule_To_v1alpha1_Rule,
	)
}

func autoConvert_v1alpha1_EntityRule_To_calico_EntityRule(in *EntityRule, out *calico.EntityRule, s conversion.Scope) error {
	out.Tag = in.Tag
	out.Net = (*net.IPNet)(unsafe.Pointer(in.Net))
	out.Selector = in.Selector
	out.Ports = *(*[]numorstring.Port)(unsafe.Pointer(&in.Ports))
	out.NotTag = in.NotTag
	out.NotNet = (*net.IPNet)(unsafe.Pointer(in.NotNet))
	out.NotSelector = in.NotSelector
	out.NotPorts = *(*[]numorstring.Port)(unsafe.Pointer(&in.NotPorts))
	return nil
}

// Convert_v1alpha1_EntityRule_To_calico_EntityRule is an autogenerated conversion function.
func Convert_v1alpha1_EntityRule_To_calico_EntityRule(in *EntityRule, out *calico.EntityRule, s conversion.Scope) error {
	return autoConvert_v1alpha1_EntityRule_To_calico_EntityRule(in, out, s)
}

func autoConvert_calico_EntityRule_To_v1alpha1_EntityRule(in *calico.EntityRule, out *EntityRule, s conversion.Scope) error {
	out.Tag = in.Tag
	out.Net = (*net.IPNet)(unsafe.Pointer(in.Net))
	out.Selector = in.Selector
	out.Ports = *(*[]numorstring.Port)(unsafe.Pointer(&in.Ports))
	out.NotTag = in.NotTag
	out.NotNet = (*net.IPNet)(unsafe.Pointer(in.NotNet))
	out.NotSelector = in.NotSelector
	out.NotPorts = *(*[]numorstring.Port)(unsafe.Pointer(&in.NotPorts))
	return nil
}

// Convert_calico_EntityRule_To_v1alpha1_EntityRule is an autogenerated conversion function.
func Convert_calico_EntityRule_To_v1alpha1_EntityRule(in *calico.EntityRule, out *EntityRule, s conversion.Scope) error {
	return autoConvert_calico_EntityRule_To_v1alpha1_EntityRule(in, out, s)
}

func autoConvert_v1alpha1_ICMPFields_To_calico_ICMPFields(in *ICMPFields, out *calico.ICMPFields, s conversion.Scope) error {
	out.Type = (*int)(unsafe.Pointer(in.Type))
	out.Code = (*int)(unsafe.Pointer(in.Code))
	return nil
}

// Convert_v1alpha1_ICMPFields_To_calico_ICMPFields is an autogenerated conversion function.
func Convert_v1alpha1_ICMPFields_To_calico_ICMPFields(in *ICMPFields, out *calico.ICMPFields, s conversion.Scope) error {
	return autoConvert_v1alpha1_ICMPFields_To_calico_ICMPFields(in, out, s)
}

func autoConvert_calico_ICMPFields_To_v1alpha1_ICMPFields(in *calico.ICMPFields, out *ICMPFields, s conversion.Scope) error {
	out.Type = (*int)(unsafe.Pointer(in.Type))
	out.Code = (*int)(unsafe.Pointer(in.Code))
	return nil
}

// Convert_calico_ICMPFields_To_v1alpha1_ICMPFields is an autogenerated conversion function.
func Convert_calico_ICMPFields_To_v1alpha1_ICMPFields(in *calico.ICMPFields, out *ICMPFields, s conversion.Scope) error {
	return autoConvert_calico_ICMPFields_To_v1alpha1_ICMPFields(in, out, s)
}

func autoConvert_v1alpha1_Policy_To_calico_Policy(in *Policy, out *calico.Policy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha1_PolicySpec_To_calico_PolicySpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_PolicyStatus_To_calico_PolicyStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_Policy_To_calico_Policy is an autogenerated conversion function.
func Convert_v1alpha1_Policy_To_calico_Policy(in *Policy, out *calico.Policy, s conversion.Scope) error {
	return autoConvert_v1alpha1_Policy_To_calico_Policy(in, out, s)
}

func autoConvert_calico_Policy_To_v1alpha1_Policy(in *calico.Policy, out *Policy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_calico_PolicySpec_To_v1alpha1_PolicySpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_calico_PolicyStatus_To_v1alpha1_PolicyStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_calico_Policy_To_v1alpha1_Policy is an autogenerated conversion function.
func Convert_calico_Policy_To_v1alpha1_Policy(in *calico.Policy, out *Policy, s conversion.Scope) error {
	return autoConvert_calico_Policy_To_v1alpha1_Policy(in, out, s)
}

func autoConvert_v1alpha1_PolicyList_To_calico_PolicyList(in *PolicyList, out *calico.PolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]calico.Policy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v1alpha1_PolicyList_To_calico_PolicyList is an autogenerated conversion function.
func Convert_v1alpha1_PolicyList_To_calico_PolicyList(in *PolicyList, out *calico.PolicyList, s conversion.Scope) error {
	return autoConvert_v1alpha1_PolicyList_To_calico_PolicyList(in, out, s)
}

func autoConvert_calico_PolicyList_To_v1alpha1_PolicyList(in *calico.PolicyList, out *PolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items == nil {
		out.Items = make([]Policy, 0)
	} else {
		out.Items = *(*[]Policy)(unsafe.Pointer(&in.Items))
	}
	return nil
}

// Convert_calico_PolicyList_To_v1alpha1_PolicyList is an autogenerated conversion function.
func Convert_calico_PolicyList_To_v1alpha1_PolicyList(in *calico.PolicyList, out *PolicyList, s conversion.Scope) error {
	return autoConvert_calico_PolicyList_To_v1alpha1_PolicyList(in, out, s)
}

func autoConvert_v1alpha1_PolicySpec_To_calico_PolicySpec(in *PolicySpec, out *calico.PolicySpec, s conversion.Scope) error {
	out.Order = (*float64)(unsafe.Pointer(in.Order))
	out.IngressRules = *(*[]calico.Rule)(unsafe.Pointer(&in.IngressRules))
	out.EgressRules = *(*[]calico.Rule)(unsafe.Pointer(&in.EgressRules))
	out.Selector = in.Selector
	out.DoNotTrack = in.DoNotTrack
	return nil
}

// Convert_v1alpha1_PolicySpec_To_calico_PolicySpec is an autogenerated conversion function.
func Convert_v1alpha1_PolicySpec_To_calico_PolicySpec(in *PolicySpec, out *calico.PolicySpec, s conversion.Scope) error {
	return autoConvert_v1alpha1_PolicySpec_To_calico_PolicySpec(in, out, s)
}

func autoConvert_calico_PolicySpec_To_v1alpha1_PolicySpec(in *calico.PolicySpec, out *PolicySpec, s conversion.Scope) error {
	out.Order = (*float64)(unsafe.Pointer(in.Order))
	out.IngressRules = *(*[]Rule)(unsafe.Pointer(&in.IngressRules))
	out.EgressRules = *(*[]Rule)(unsafe.Pointer(&in.EgressRules))
	out.Selector = in.Selector
	out.DoNotTrack = in.DoNotTrack
	return nil
}

// Convert_calico_PolicySpec_To_v1alpha1_PolicySpec is an autogenerated conversion function.
func Convert_calico_PolicySpec_To_v1alpha1_PolicySpec(in *calico.PolicySpec, out *PolicySpec, s conversion.Scope) error {
	return autoConvert_calico_PolicySpec_To_v1alpha1_PolicySpec(in, out, s)
}

func autoConvert_v1alpha1_PolicyStatus_To_calico_PolicyStatus(in *PolicyStatus, out *calico.PolicyStatus, s conversion.Scope) error {
	return nil
}

// Convert_v1alpha1_PolicyStatus_To_calico_PolicyStatus is an autogenerated conversion function.
func Convert_v1alpha1_PolicyStatus_To_calico_PolicyStatus(in *PolicyStatus, out *calico.PolicyStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_PolicyStatus_To_calico_PolicyStatus(in, out, s)
}

func autoConvert_calico_PolicyStatus_To_v1alpha1_PolicyStatus(in *calico.PolicyStatus, out *PolicyStatus, s conversion.Scope) error {
	return nil
}

// Convert_calico_PolicyStatus_To_v1alpha1_PolicyStatus is an autogenerated conversion function.
func Convert_calico_PolicyStatus_To_v1alpha1_PolicyStatus(in *calico.PolicyStatus, out *PolicyStatus, s conversion.Scope) error {
	return autoConvert_calico_PolicyStatus_To_v1alpha1_PolicyStatus(in, out, s)
}

func autoConvert_v1alpha1_Rule_To_calico_Rule(in *Rule, out *calico.Rule, s conversion.Scope) error {
	out.Action = in.Action
	out.IPVersion = (*int)(unsafe.Pointer(in.IPVersion))
	out.Protocol = (*numorstring.Protocol)(unsafe.Pointer(in.Protocol))
	out.ICMP = (*calico.ICMPFields)(unsafe.Pointer(in.ICMP))
	out.NotProtocol = (*numorstring.Protocol)(unsafe.Pointer(in.NotProtocol))
	out.NotICMP = (*calico.ICMPFields)(unsafe.Pointer(in.NotICMP))
	if err := Convert_v1alpha1_EntityRule_To_calico_EntityRule(&in.Source, &out.Source, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_EntityRule_To_calico_EntityRule(&in.Destination, &out.Destination, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_Rule_To_calico_Rule is an autogenerated conversion function.
func Convert_v1alpha1_Rule_To_calico_Rule(in *Rule, out *calico.Rule, s conversion.Scope) error {
	return autoConvert_v1alpha1_Rule_To_calico_Rule(in, out, s)
}

func autoConvert_calico_Rule_To_v1alpha1_Rule(in *calico.Rule, out *Rule, s conversion.Scope) error {
	out.Action = in.Action
	out.IPVersion = (*int)(unsafe.Pointer(in.IPVersion))
	out.Protocol = (*numorstring.Protocol)(unsafe.Pointer(in.Protocol))
	out.ICMP = (*ICMPFields)(unsafe.Pointer(in.ICMP))
	out.NotProtocol = (*numorstring.Protocol)(unsafe.Pointer(in.NotProtocol))
	out.NotICMP = (*ICMPFields)(unsafe.Pointer(in.NotICMP))
	if err := Convert_calico_EntityRule_To_v1alpha1_EntityRule(&in.Source, &out.Source, s); err != nil {
		return err
	}
	if err := Convert_calico_EntityRule_To_v1alpha1_EntityRule(&in.Destination, &out.Destination, s); err != nil {
		return err
	}
	return nil
}

// Convert_calico_Rule_To_v1alpha1_Rule is an autogenerated conversion function.
func Convert_calico_Rule_To_v1alpha1_Rule(in *calico.Rule, out *Rule, s conversion.Scope) error {
	return autoConvert_calico_Rule_To_v1alpha1_Rule(in, out, s)
}
