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

package v2

import (
	apiv2 "github.com/projectcalico/libcalico-go/lib/apiv2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v2_NetworkPolicy, InType: reflect.TypeOf(&NetworkPolicy{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v2_NetworkPolicyList, InType: reflect.TypeOf(&NetworkPolicyList{})},
	)
}

// DeepCopy_v2_NetworkPolicy is an autogenerated deepcopy function.
func DeepCopy_v2_NetworkPolicy(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*NetworkPolicy)
		out := out.(*NetworkPolicy)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*apiv2.PolicySpec)
		}
		return nil
	}
}

// DeepCopy_v2_NetworkPolicyList is an autogenerated deepcopy function.
func DeepCopy_v2_NetworkPolicyList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*NetworkPolicyList)
		out := out.(*NetworkPolicyList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]NetworkPolicy, len(*in))
			for i := range *in {
				if err := DeepCopy_v2_NetworkPolicy(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
