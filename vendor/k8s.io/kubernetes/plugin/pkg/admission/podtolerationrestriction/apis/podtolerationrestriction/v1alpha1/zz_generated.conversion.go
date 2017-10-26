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
	v1 "k8s.io/api/core/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	api "k8s.io/kubernetes/pkg/api"
	podtolerationrestriction "k8s.io/kubernetes/plugin/pkg/admission/podtolerationrestriction/apis/podtolerationrestriction"
	unsafe "unsafe"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedConversionFuncs(
		Convert_v1alpha1_Configuration_To_podtolerationrestriction_Configuration,
		Convert_podtolerationrestriction_Configuration_To_v1alpha1_Configuration,
	)
}

func autoConvert_v1alpha1_Configuration_To_podtolerationrestriction_Configuration(in *Configuration, out *podtolerationrestriction.Configuration, s conversion.Scope) error {
	out.Default = *(*[]api.Toleration)(unsafe.Pointer(&in.Default))
	out.Whitelist = *(*[]api.Toleration)(unsafe.Pointer(&in.Whitelist))
	return nil
}

// Convert_v1alpha1_Configuration_To_podtolerationrestriction_Configuration is an autogenerated conversion function.
func Convert_v1alpha1_Configuration_To_podtolerationrestriction_Configuration(in *Configuration, out *podtolerationrestriction.Configuration, s conversion.Scope) error {
	return autoConvert_v1alpha1_Configuration_To_podtolerationrestriction_Configuration(in, out, s)
}

func autoConvert_podtolerationrestriction_Configuration_To_v1alpha1_Configuration(in *podtolerationrestriction.Configuration, out *Configuration, s conversion.Scope) error {
	out.Default = *(*[]v1.Toleration)(unsafe.Pointer(&in.Default))
	out.Whitelist = *(*[]v1.Toleration)(unsafe.Pointer(&in.Whitelist))
	return nil
}

// Convert_podtolerationrestriction_Configuration_To_v1alpha1_Configuration is an autogenerated conversion function.
func Convert_podtolerationrestriction_Configuration_To_v1alpha1_Configuration(in *podtolerationrestriction.Configuration, out *Configuration, s conversion.Scope) error {
	return autoConvert_podtolerationrestriction_Configuration_To_v1alpha1_Configuration(in, out, s)
}
