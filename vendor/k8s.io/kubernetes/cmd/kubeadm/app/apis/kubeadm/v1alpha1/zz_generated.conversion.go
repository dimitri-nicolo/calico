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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	unsafe "unsafe"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedConversionFuncs(
		Convert_v1alpha1_API_To_kubeadm_API,
		Convert_kubeadm_API_To_v1alpha1_API,
		Convert_v1alpha1_Etcd_To_kubeadm_Etcd,
		Convert_kubeadm_Etcd_To_v1alpha1_Etcd,
		Convert_v1alpha1_MasterConfiguration_To_kubeadm_MasterConfiguration,
		Convert_kubeadm_MasterConfiguration_To_v1alpha1_MasterConfiguration,
		Convert_v1alpha1_Networking_To_kubeadm_Networking,
		Convert_kubeadm_Networking_To_v1alpha1_Networking,
		Convert_v1alpha1_NodeConfiguration_To_kubeadm_NodeConfiguration,
		Convert_kubeadm_NodeConfiguration_To_v1alpha1_NodeConfiguration,
		Convert_v1alpha1_TokenDiscovery_To_kubeadm_TokenDiscovery,
		Convert_kubeadm_TokenDiscovery_To_v1alpha1_TokenDiscovery,
	)
}

func autoConvert_v1alpha1_API_To_kubeadm_API(in *API, out *kubeadm.API, s conversion.Scope) error {
	out.AdvertiseAddress = in.AdvertiseAddress
	out.BindPort = in.BindPort
	return nil
}

// Convert_v1alpha1_API_To_kubeadm_API is an autogenerated conversion function.
func Convert_v1alpha1_API_To_kubeadm_API(in *API, out *kubeadm.API, s conversion.Scope) error {
	return autoConvert_v1alpha1_API_To_kubeadm_API(in, out, s)
}

func autoConvert_kubeadm_API_To_v1alpha1_API(in *kubeadm.API, out *API, s conversion.Scope) error {
	out.AdvertiseAddress = in.AdvertiseAddress
	out.BindPort = in.BindPort
	return nil
}

// Convert_kubeadm_API_To_v1alpha1_API is an autogenerated conversion function.
func Convert_kubeadm_API_To_v1alpha1_API(in *kubeadm.API, out *API, s conversion.Scope) error {
	return autoConvert_kubeadm_API_To_v1alpha1_API(in, out, s)
}

func autoConvert_v1alpha1_Etcd_To_kubeadm_Etcd(in *Etcd, out *kubeadm.Etcd, s conversion.Scope) error {
	out.Endpoints = *(*[]string)(unsafe.Pointer(&in.Endpoints))
	out.CAFile = in.CAFile
	out.CertFile = in.CertFile
	out.KeyFile = in.KeyFile
	out.DataDir = in.DataDir
	out.ExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.ExtraArgs))
	out.Image = in.Image
	return nil
}

// Convert_v1alpha1_Etcd_To_kubeadm_Etcd is an autogenerated conversion function.
func Convert_v1alpha1_Etcd_To_kubeadm_Etcd(in *Etcd, out *kubeadm.Etcd, s conversion.Scope) error {
	return autoConvert_v1alpha1_Etcd_To_kubeadm_Etcd(in, out, s)
}

func autoConvert_kubeadm_Etcd_To_v1alpha1_Etcd(in *kubeadm.Etcd, out *Etcd, s conversion.Scope) error {
	out.Endpoints = *(*[]string)(unsafe.Pointer(&in.Endpoints))
	out.CAFile = in.CAFile
	out.CertFile = in.CertFile
	out.KeyFile = in.KeyFile
	out.DataDir = in.DataDir
	out.ExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.ExtraArgs))
	out.Image = in.Image
	return nil
}

// Convert_kubeadm_Etcd_To_v1alpha1_Etcd is an autogenerated conversion function.
func Convert_kubeadm_Etcd_To_v1alpha1_Etcd(in *kubeadm.Etcd, out *Etcd, s conversion.Scope) error {
	return autoConvert_kubeadm_Etcd_To_v1alpha1_Etcd(in, out, s)
}

func autoConvert_v1alpha1_MasterConfiguration_To_kubeadm_MasterConfiguration(in *MasterConfiguration, out *kubeadm.MasterConfiguration, s conversion.Scope) error {
	if err := Convert_v1alpha1_API_To_kubeadm_API(&in.API, &out.API, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_Etcd_To_kubeadm_Etcd(&in.Etcd, &out.Etcd, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_Networking_To_kubeadm_Networking(&in.Networking, &out.Networking, s); err != nil {
		return err
	}
	out.KubernetesVersion = in.KubernetesVersion
	out.CloudProvider = in.CloudProvider
	out.NodeName = in.NodeName
	out.AuthorizationModes = *(*[]string)(unsafe.Pointer(&in.AuthorizationModes))
	out.Token = in.Token
	out.TokenTTL = (*v1.Duration)(unsafe.Pointer(in.TokenTTL))
	out.APIServerExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.APIServerExtraArgs))
	out.ControllerManagerExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.ControllerManagerExtraArgs))
	out.SchedulerExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.SchedulerExtraArgs))
	out.APIServerCertSANs = *(*[]string)(unsafe.Pointer(&in.APIServerCertSANs))
	out.CertificatesDir = in.CertificatesDir
	out.ImageRepository = in.ImageRepository
	out.UnifiedControlPlaneImage = in.UnifiedControlPlaneImage
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	return nil
}

// Convert_v1alpha1_MasterConfiguration_To_kubeadm_MasterConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_MasterConfiguration_To_kubeadm_MasterConfiguration(in *MasterConfiguration, out *kubeadm.MasterConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_MasterConfiguration_To_kubeadm_MasterConfiguration(in, out, s)
}

func autoConvert_kubeadm_MasterConfiguration_To_v1alpha1_MasterConfiguration(in *kubeadm.MasterConfiguration, out *MasterConfiguration, s conversion.Scope) error {
	if err := Convert_kubeadm_API_To_v1alpha1_API(&in.API, &out.API, s); err != nil {
		return err
	}
	if err := Convert_kubeadm_Etcd_To_v1alpha1_Etcd(&in.Etcd, &out.Etcd, s); err != nil {
		return err
	}
	if err := Convert_kubeadm_Networking_To_v1alpha1_Networking(&in.Networking, &out.Networking, s); err != nil {
		return err
	}
	out.KubernetesVersion = in.KubernetesVersion
	out.CloudProvider = in.CloudProvider
	out.NodeName = in.NodeName
	out.AuthorizationModes = *(*[]string)(unsafe.Pointer(&in.AuthorizationModes))
	out.Token = in.Token
	out.TokenTTL = (*v1.Duration)(unsafe.Pointer(in.TokenTTL))
	out.APIServerExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.APIServerExtraArgs))
	out.ControllerManagerExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.ControllerManagerExtraArgs))
	out.SchedulerExtraArgs = *(*map[string]string)(unsafe.Pointer(&in.SchedulerExtraArgs))
	out.APIServerCertSANs = *(*[]string)(unsafe.Pointer(&in.APIServerCertSANs))
	out.CertificatesDir = in.CertificatesDir
	out.ImageRepository = in.ImageRepository
	// INFO: in.CIImageRepository opted out of conversion generation
	out.UnifiedControlPlaneImage = in.UnifiedControlPlaneImage
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	return nil
}

// Convert_kubeadm_MasterConfiguration_To_v1alpha1_MasterConfiguration is an autogenerated conversion function.
func Convert_kubeadm_MasterConfiguration_To_v1alpha1_MasterConfiguration(in *kubeadm.MasterConfiguration, out *MasterConfiguration, s conversion.Scope) error {
	return autoConvert_kubeadm_MasterConfiguration_To_v1alpha1_MasterConfiguration(in, out, s)
}

func autoConvert_v1alpha1_Networking_To_kubeadm_Networking(in *Networking, out *kubeadm.Networking, s conversion.Scope) error {
	out.ServiceSubnet = in.ServiceSubnet
	out.PodSubnet = in.PodSubnet
	out.DNSDomain = in.DNSDomain
	return nil
}

// Convert_v1alpha1_Networking_To_kubeadm_Networking is an autogenerated conversion function.
func Convert_v1alpha1_Networking_To_kubeadm_Networking(in *Networking, out *kubeadm.Networking, s conversion.Scope) error {
	return autoConvert_v1alpha1_Networking_To_kubeadm_Networking(in, out, s)
}

func autoConvert_kubeadm_Networking_To_v1alpha1_Networking(in *kubeadm.Networking, out *Networking, s conversion.Scope) error {
	out.ServiceSubnet = in.ServiceSubnet
	out.PodSubnet = in.PodSubnet
	out.DNSDomain = in.DNSDomain
	return nil
}

// Convert_kubeadm_Networking_To_v1alpha1_Networking is an autogenerated conversion function.
func Convert_kubeadm_Networking_To_v1alpha1_Networking(in *kubeadm.Networking, out *Networking, s conversion.Scope) error {
	return autoConvert_kubeadm_Networking_To_v1alpha1_Networking(in, out, s)
}

func autoConvert_v1alpha1_NodeConfiguration_To_kubeadm_NodeConfiguration(in *NodeConfiguration, out *kubeadm.NodeConfiguration, s conversion.Scope) error {
	out.CACertPath = in.CACertPath
	out.DiscoveryFile = in.DiscoveryFile
	out.DiscoveryToken = in.DiscoveryToken
	out.DiscoveryTokenAPIServers = *(*[]string)(unsafe.Pointer(&in.DiscoveryTokenAPIServers))
	out.NodeName = in.NodeName
	out.TLSBootstrapToken = in.TLSBootstrapToken
	out.Token = in.Token
	out.DiscoveryTokenCACertHashes = *(*[]string)(unsafe.Pointer(&in.DiscoveryTokenCACertHashes))
	out.DiscoveryTokenUnsafeSkipCAVerification = in.DiscoveryTokenUnsafeSkipCAVerification
	return nil
}

// Convert_v1alpha1_NodeConfiguration_To_kubeadm_NodeConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_NodeConfiguration_To_kubeadm_NodeConfiguration(in *NodeConfiguration, out *kubeadm.NodeConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_NodeConfiguration_To_kubeadm_NodeConfiguration(in, out, s)
}

func autoConvert_kubeadm_NodeConfiguration_To_v1alpha1_NodeConfiguration(in *kubeadm.NodeConfiguration, out *NodeConfiguration, s conversion.Scope) error {
	out.CACertPath = in.CACertPath
	out.DiscoveryFile = in.DiscoveryFile
	out.DiscoveryToken = in.DiscoveryToken
	out.DiscoveryTokenAPIServers = *(*[]string)(unsafe.Pointer(&in.DiscoveryTokenAPIServers))
	out.NodeName = in.NodeName
	out.TLSBootstrapToken = in.TLSBootstrapToken
	out.Token = in.Token
	out.DiscoveryTokenCACertHashes = *(*[]string)(unsafe.Pointer(&in.DiscoveryTokenCACertHashes))
	out.DiscoveryTokenUnsafeSkipCAVerification = in.DiscoveryTokenUnsafeSkipCAVerification
	return nil
}

// Convert_kubeadm_NodeConfiguration_To_v1alpha1_NodeConfiguration is an autogenerated conversion function.
func Convert_kubeadm_NodeConfiguration_To_v1alpha1_NodeConfiguration(in *kubeadm.NodeConfiguration, out *NodeConfiguration, s conversion.Scope) error {
	return autoConvert_kubeadm_NodeConfiguration_To_v1alpha1_NodeConfiguration(in, out, s)
}

func autoConvert_v1alpha1_TokenDiscovery_To_kubeadm_TokenDiscovery(in *TokenDiscovery, out *kubeadm.TokenDiscovery, s conversion.Scope) error {
	out.ID = in.ID
	out.Secret = in.Secret
	out.Addresses = *(*[]string)(unsafe.Pointer(&in.Addresses))
	return nil
}

// Convert_v1alpha1_TokenDiscovery_To_kubeadm_TokenDiscovery is an autogenerated conversion function.
func Convert_v1alpha1_TokenDiscovery_To_kubeadm_TokenDiscovery(in *TokenDiscovery, out *kubeadm.TokenDiscovery, s conversion.Scope) error {
	return autoConvert_v1alpha1_TokenDiscovery_To_kubeadm_TokenDiscovery(in, out, s)
}

func autoConvert_kubeadm_TokenDiscovery_To_v1alpha1_TokenDiscovery(in *kubeadm.TokenDiscovery, out *TokenDiscovery, s conversion.Scope) error {
	out.ID = in.ID
	out.Secret = in.Secret
	out.Addresses = *(*[]string)(unsafe.Pointer(&in.Addresses))
	return nil
}

// Convert_kubeadm_TokenDiscovery_To_v1alpha1_TokenDiscovery is an autogenerated conversion function.
func Convert_kubeadm_TokenDiscovery_To_v1alpha1_TokenDiscovery(in *kubeadm.TokenDiscovery, out *TokenDiscovery, s conversion.Scope) error {
	return autoConvert_kubeadm_TokenDiscovery_To_v1alpha1_TokenDiscovery(in, out, s)
}
