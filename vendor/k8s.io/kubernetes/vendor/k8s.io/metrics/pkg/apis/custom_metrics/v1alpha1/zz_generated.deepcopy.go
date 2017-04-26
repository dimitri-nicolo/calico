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

package v1alpha1

import (
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
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1alpha1_MetricValue, InType: reflect.TypeOf(&MetricValue{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1alpha1_MetricValueList, InType: reflect.TypeOf(&MetricValueList{})},
	)
}

func DeepCopy_v1alpha1_MetricValue(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*MetricValue)
		out := out.(*MetricValue)
		*out = *in
		out.Timestamp = in.Timestamp.DeepCopy()
		if in.WindowSeconds != nil {
			in, out := &in.WindowSeconds, &out.WindowSeconds
			*out = new(int64)
			**out = **in
		}
		out.Value = in.Value.DeepCopy()
		return nil
	}
}

func DeepCopy_v1alpha1_MetricValueList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*MetricValueList)
		out := out.(*MetricValueList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]MetricValue, len(*in))
			for i := range *in {
				if err := DeepCopy_v1alpha1_MetricValue(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
