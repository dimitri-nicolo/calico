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

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1

import (
	api "github.com/projectcalico/libcalico-go/lib/api"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	reflect "reflect"
)

func init() {
	SchemeBuilder.Register(RegisterDeepCopies)
}

// RegisterDeepCopies adds deep-copy functions to the given scheme. Public
// to allow building arbitrary schemes.
func RegisterDeepCopies(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedDeepCopyFuncs(
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_Node, InType: reflect.TypeOf(&Node{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_NodeList, InType: reflect.TypeOf(&NodeList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_Policy, InType: reflect.TypeOf(&Policy{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_PolicyList, InType: reflect.TypeOf(&PolicyList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_PolicyStatus, InType: reflect.TypeOf(&PolicyStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_Tier, InType: reflect.TypeOf(&Tier{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1_TierList, InType: reflect.TypeOf(&TierList{})},
	)
}

// DeepCopy_v1_Node is an autogenerated deepcopy function.
func DeepCopy_v1_Node(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Node)
		out := out.(*Node)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*meta_v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*api.NodeSpec)
		}
		return nil
	}
}

// DeepCopy_v1_NodeList is an autogenerated deepcopy function.
func DeepCopy_v1_NodeList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*NodeList)
		out := out.(*NodeList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Node, len(*in))
			for i := range *in {
				if err := DeepCopy_v1_Node(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// DeepCopy_v1_Policy is an autogenerated deepcopy function.
func DeepCopy_v1_Policy(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Policy)
		out := out.(*Policy)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*meta_v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*api.PolicySpec)
		}
		return nil
	}
}

// DeepCopy_v1_PolicyList is an autogenerated deepcopy function.
func DeepCopy_v1_PolicyList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*PolicyList)
		out := out.(*PolicyList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Policy, len(*in))
			for i := range *in {
				if err := DeepCopy_v1_Policy(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// DeepCopy_v1_PolicyStatus is an autogenerated deepcopy function.
func DeepCopy_v1_PolicyStatus(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*PolicyStatus)
		out := out.(*PolicyStatus)
		*out = *in
		return nil
	}
}

// DeepCopy_v1_Tier is an autogenerated deepcopy function.
func DeepCopy_v1_Tier(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Tier)
		out := out.(*Tier)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*meta_v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*api.TierSpec)
		}
		return nil
	}
}

// DeepCopy_v1_TierList is an autogenerated deepcopy function.
func DeepCopy_v1_TierList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*TierList)
		out := out.(*TierList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Tier, len(*in))
			for i := range *in {
				if err := DeepCopy_v1_Tier(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
