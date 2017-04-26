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

package metrics

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	api "k8s.io/client-go/pkg/api"
	reflect "reflect"
)

func init() {
	SchemeBuilder.Register(RegisterDeepCopies)
}

// RegisterDeepCopies adds deep-copy functions to the given scheme. Public
// to allow building arbitrary schemes.
func RegisterDeepCopies(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedDeepCopyFuncs(
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_metrics_ContainerMetrics, InType: reflect.TypeOf(&ContainerMetrics{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_metrics_NodeMetrics, InType: reflect.TypeOf(&NodeMetrics{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_metrics_NodeMetricsList, InType: reflect.TypeOf(&NodeMetricsList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_metrics_PodMetrics, InType: reflect.TypeOf(&PodMetrics{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_metrics_PodMetricsList, InType: reflect.TypeOf(&PodMetricsList{})},
	)
}

func DeepCopy_metrics_ContainerMetrics(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ContainerMetrics)
		out := out.(*ContainerMetrics)
		*out = *in
		if in.Usage != nil {
			in, out := &in.Usage, &out.Usage
			*out = make(api.ResourceList)
			for key, val := range *in {
				(*out)[key] = val.DeepCopy()
			}
		}
		return nil
	}
}

func DeepCopy_metrics_NodeMetrics(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*NodeMetrics)
		out := out.(*NodeMetrics)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		out.Timestamp = in.Timestamp.DeepCopy()
		if in.Usage != nil {
			in, out := &in.Usage, &out.Usage
			*out = make(api.ResourceList)
			for key, val := range *in {
				(*out)[key] = val.DeepCopy()
			}
		}
		return nil
	}
}

func DeepCopy_metrics_NodeMetricsList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*NodeMetricsList)
		out := out.(*NodeMetricsList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]NodeMetrics, len(*in))
			for i := range *in {
				if err := DeepCopy_metrics_NodeMetrics(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func DeepCopy_metrics_PodMetrics(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*PodMetrics)
		out := out.(*PodMetrics)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		out.Timestamp = in.Timestamp.DeepCopy()
		if in.Containers != nil {
			in, out := &in.Containers, &out.Containers
			*out = make([]ContainerMetrics, len(*in))
			for i := range *in {
				if err := DeepCopy_metrics_ContainerMetrics(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func DeepCopy_metrics_PodMetricsList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*PodMetricsList)
		out := out.(*PodMetricsList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]PodMetrics, len(*in))
			for i := range *in {
				if err := DeepCopy_metrics_PodMetrics(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
