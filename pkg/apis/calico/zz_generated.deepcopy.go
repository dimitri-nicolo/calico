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

package calico

import (
	api "github.com/projectcalico/libcalico-go/lib/api"
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
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_Endpoint, InType: reflect.TypeOf(&Endpoint{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_EndpointList, InType: reflect.TypeOf(&EndpointList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_EndpointMeta, InType: reflect.TypeOf(&EndpointMeta{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_Policy, InType: reflect.TypeOf(&Policy{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_PolicyList, InType: reflect.TypeOf(&PolicyList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_PolicyStatus, InType: reflect.TypeOf(&PolicyStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_Tier, InType: reflect.TypeOf(&Tier{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_calico_TierList, InType: reflect.TypeOf(&TierList{})},
	)
}

func DeepCopy_calico_Endpoint(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Endpoint)
		out := out.(*Endpoint)
		*out = *in
		if err := DeepCopy_calico_EndpointMeta(&in.EndpointMeta, &out.EndpointMeta, c); err != nil {
			return err
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*api.WorkloadEndpointSpec)
		}
		return nil
	}
}

func DeepCopy_calico_EndpointList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*EndpointList)
		out := out.(*EndpointList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Endpoint, len(*in))
			for i := range *in {
				if err := DeepCopy_calico_Endpoint(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func DeepCopy_calico_EndpointMeta(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*EndpointMeta)
		out := out.(*EndpointMeta)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.WorkloadEndpointMetadata); err != nil {
			return err
		} else {
			out.WorkloadEndpointMetadata = *newVal.(*api.WorkloadEndpointMetadata)
		}
		return nil
	}
}

func DeepCopy_calico_Policy(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Policy)
		out := out.(*Policy)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*api.PolicySpec)
		}
		return nil
	}
}

func DeepCopy_calico_PolicyList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*PolicyList)
		out := out.(*PolicyList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Policy, len(*in))
			for i := range *in {
				if err := DeepCopy_calico_Policy(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func DeepCopy_calico_PolicyStatus(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*PolicyStatus)
		out := out.(*PolicyStatus)
		*out = *in
		return nil
	}
}

func DeepCopy_calico_Tier(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Tier)
		out := out.(*Tier)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		if newVal, err := c.DeepCopy(&in.Spec); err != nil {
			return err
		} else {
			out.Spec = *newVal.(*api.TierSpec)
		}
		return nil
	}
}

func DeepCopy_calico_TierList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*TierList)
		out := out.(*TierList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Tier, len(*in))
			for i := range *in {
				if err := DeepCopy_calico_Tier(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
