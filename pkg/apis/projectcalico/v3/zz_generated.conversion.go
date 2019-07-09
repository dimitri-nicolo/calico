// +build !ignore_autogenerated

// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Code generated by conversion-gen. DO NOT EDIT.

package v3

import (
	unsafe "unsafe"

	projectcalico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedConversionFuncs(
		Convert_v3_BGPConfiguration_To_projectcalico_BGPConfiguration,
		Convert_projectcalico_BGPConfiguration_To_v3_BGPConfiguration,
		Convert_v3_BGPConfigurationList_To_projectcalico_BGPConfigurationList,
		Convert_projectcalico_BGPConfigurationList_To_v3_BGPConfigurationList,
		Convert_v3_BGPPeer_To_projectcalico_BGPPeer,
		Convert_projectcalico_BGPPeer_To_v3_BGPPeer,
		Convert_v3_BGPPeerList_To_projectcalico_BGPPeerList,
		Convert_projectcalico_BGPPeerList_To_v3_BGPPeerList,
		Convert_v3_GlobalNetworkPolicy_To_projectcalico_GlobalNetworkPolicy,
		Convert_projectcalico_GlobalNetworkPolicy_To_v3_GlobalNetworkPolicy,
		Convert_v3_GlobalNetworkPolicyList_To_projectcalico_GlobalNetworkPolicyList,
		Convert_projectcalico_GlobalNetworkPolicyList_To_v3_GlobalNetworkPolicyList,
		Convert_v3_GlobalNetworkSet_To_projectcalico_GlobalNetworkSet,
		Convert_projectcalico_GlobalNetworkSet_To_v3_GlobalNetworkSet,
		Convert_v3_GlobalNetworkSetList_To_projectcalico_GlobalNetworkSetList,
		Convert_projectcalico_GlobalNetworkSetList_To_v3_GlobalNetworkSetList,
		Convert_v3_GlobalReport_To_projectcalico_GlobalReport,
		Convert_projectcalico_GlobalReport_To_v3_GlobalReport,
		Convert_v3_GlobalReportList_To_projectcalico_GlobalReportList,
		Convert_projectcalico_GlobalReportList_To_v3_GlobalReportList,
		Convert_v3_GlobalReportType_To_projectcalico_GlobalReportType,
		Convert_projectcalico_GlobalReportType_To_v3_GlobalReportType,
		Convert_v3_GlobalReportTypeList_To_projectcalico_GlobalReportTypeList,
		Convert_projectcalico_GlobalReportTypeList_To_v3_GlobalReportTypeList,
		Convert_v3_GlobalThreatFeed_To_projectcalico_GlobalThreatFeed,
		Convert_projectcalico_GlobalThreatFeed_To_v3_GlobalThreatFeed,
		Convert_v3_GlobalThreatFeedList_To_projectcalico_GlobalThreatFeedList,
		Convert_projectcalico_GlobalThreatFeedList_To_v3_GlobalThreatFeedList,
		Convert_v3_HostEndpoint_To_projectcalico_HostEndpoint,
		Convert_projectcalico_HostEndpoint_To_v3_HostEndpoint,
		Convert_v3_HostEndpointList_To_projectcalico_HostEndpointList,
		Convert_projectcalico_HostEndpointList_To_v3_HostEndpointList,
		Convert_v3_IPPool_To_projectcalico_IPPool,
		Convert_projectcalico_IPPool_To_v3_IPPool,
		Convert_v3_IPPoolList_To_projectcalico_IPPoolList,
		Convert_projectcalico_IPPoolList_To_v3_IPPoolList,
		Convert_v3_LicenseKey_To_projectcalico_LicenseKey,
		Convert_projectcalico_LicenseKey_To_v3_LicenseKey,
		Convert_v3_LicenseKeyList_To_projectcalico_LicenseKeyList,
		Convert_projectcalico_LicenseKeyList_To_v3_LicenseKeyList,
		Convert_v3_NetworkPolicy_To_projectcalico_NetworkPolicy,
		Convert_projectcalico_NetworkPolicy_To_v3_NetworkPolicy,
		Convert_v3_NetworkPolicyList_To_projectcalico_NetworkPolicyList,
		Convert_projectcalico_NetworkPolicyList_To_v3_NetworkPolicyList,
		Convert_v3_Tier_To_projectcalico_Tier,
		Convert_projectcalico_Tier_To_v3_Tier,
		Convert_v3_TierList_To_projectcalico_TierList,
		Convert_projectcalico_TierList_To_v3_TierList,
	)
}

func autoConvert_v3_BGPConfiguration_To_projectcalico_BGPConfiguration(in *BGPConfiguration, out *projectcalico.BGPConfiguration, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_BGPConfiguration_To_projectcalico_BGPConfiguration is an autogenerated conversion function.
func Convert_v3_BGPConfiguration_To_projectcalico_BGPConfiguration(in *BGPConfiguration, out *projectcalico.BGPConfiguration, s conversion.Scope) error {
	return autoConvert_v3_BGPConfiguration_To_projectcalico_BGPConfiguration(in, out, s)
}

func autoConvert_projectcalico_BGPConfiguration_To_v3_BGPConfiguration(in *projectcalico.BGPConfiguration, out *BGPConfiguration, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_BGPConfiguration_To_v3_BGPConfiguration is an autogenerated conversion function.
func Convert_projectcalico_BGPConfiguration_To_v3_BGPConfiguration(in *projectcalico.BGPConfiguration, out *BGPConfiguration, s conversion.Scope) error {
	return autoConvert_projectcalico_BGPConfiguration_To_v3_BGPConfiguration(in, out, s)
}

func autoConvert_v3_BGPConfigurationList_To_projectcalico_BGPConfigurationList(in *BGPConfigurationList, out *projectcalico.BGPConfigurationList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.BGPConfiguration)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_BGPConfigurationList_To_projectcalico_BGPConfigurationList is an autogenerated conversion function.
func Convert_v3_BGPConfigurationList_To_projectcalico_BGPConfigurationList(in *BGPConfigurationList, out *projectcalico.BGPConfigurationList, s conversion.Scope) error {
	return autoConvert_v3_BGPConfigurationList_To_projectcalico_BGPConfigurationList(in, out, s)
}

func autoConvert_projectcalico_BGPConfigurationList_To_v3_BGPConfigurationList(in *projectcalico.BGPConfigurationList, out *BGPConfigurationList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]BGPConfiguration)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_BGPConfigurationList_To_v3_BGPConfigurationList is an autogenerated conversion function.
func Convert_projectcalico_BGPConfigurationList_To_v3_BGPConfigurationList(in *projectcalico.BGPConfigurationList, out *BGPConfigurationList, s conversion.Scope) error {
	return autoConvert_projectcalico_BGPConfigurationList_To_v3_BGPConfigurationList(in, out, s)
}

func autoConvert_v3_BGPPeer_To_projectcalico_BGPPeer(in *BGPPeer, out *projectcalico.BGPPeer, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_BGPPeer_To_projectcalico_BGPPeer is an autogenerated conversion function.
func Convert_v3_BGPPeer_To_projectcalico_BGPPeer(in *BGPPeer, out *projectcalico.BGPPeer, s conversion.Scope) error {
	return autoConvert_v3_BGPPeer_To_projectcalico_BGPPeer(in, out, s)
}

func autoConvert_projectcalico_BGPPeer_To_v3_BGPPeer(in *projectcalico.BGPPeer, out *BGPPeer, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_BGPPeer_To_v3_BGPPeer is an autogenerated conversion function.
func Convert_projectcalico_BGPPeer_To_v3_BGPPeer(in *projectcalico.BGPPeer, out *BGPPeer, s conversion.Scope) error {
	return autoConvert_projectcalico_BGPPeer_To_v3_BGPPeer(in, out, s)
}

func autoConvert_v3_BGPPeerList_To_projectcalico_BGPPeerList(in *BGPPeerList, out *projectcalico.BGPPeerList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.BGPPeer)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_BGPPeerList_To_projectcalico_BGPPeerList is an autogenerated conversion function.
func Convert_v3_BGPPeerList_To_projectcalico_BGPPeerList(in *BGPPeerList, out *projectcalico.BGPPeerList, s conversion.Scope) error {
	return autoConvert_v3_BGPPeerList_To_projectcalico_BGPPeerList(in, out, s)
}

func autoConvert_projectcalico_BGPPeerList_To_v3_BGPPeerList(in *projectcalico.BGPPeerList, out *BGPPeerList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]BGPPeer)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_BGPPeerList_To_v3_BGPPeerList is an autogenerated conversion function.
func Convert_projectcalico_BGPPeerList_To_v3_BGPPeerList(in *projectcalico.BGPPeerList, out *BGPPeerList, s conversion.Scope) error {
	return autoConvert_projectcalico_BGPPeerList_To_v3_BGPPeerList(in, out, s)
}

func autoConvert_v3_GlobalNetworkPolicy_To_projectcalico_GlobalNetworkPolicy(in *GlobalNetworkPolicy, out *projectcalico.GlobalNetworkPolicy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_GlobalNetworkPolicy_To_projectcalico_GlobalNetworkPolicy is an autogenerated conversion function.
func Convert_v3_GlobalNetworkPolicy_To_projectcalico_GlobalNetworkPolicy(in *GlobalNetworkPolicy, out *projectcalico.GlobalNetworkPolicy, s conversion.Scope) error {
	return autoConvert_v3_GlobalNetworkPolicy_To_projectcalico_GlobalNetworkPolicy(in, out, s)
}

func autoConvert_projectcalico_GlobalNetworkPolicy_To_v3_GlobalNetworkPolicy(in *projectcalico.GlobalNetworkPolicy, out *GlobalNetworkPolicy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_GlobalNetworkPolicy_To_v3_GlobalNetworkPolicy is an autogenerated conversion function.
func Convert_projectcalico_GlobalNetworkPolicy_To_v3_GlobalNetworkPolicy(in *projectcalico.GlobalNetworkPolicy, out *GlobalNetworkPolicy, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalNetworkPolicy_To_v3_GlobalNetworkPolicy(in, out, s)
}

func autoConvert_v3_GlobalNetworkPolicyList_To_projectcalico_GlobalNetworkPolicyList(in *GlobalNetworkPolicyList, out *projectcalico.GlobalNetworkPolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.GlobalNetworkPolicy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_GlobalNetworkPolicyList_To_projectcalico_GlobalNetworkPolicyList is an autogenerated conversion function.
func Convert_v3_GlobalNetworkPolicyList_To_projectcalico_GlobalNetworkPolicyList(in *GlobalNetworkPolicyList, out *projectcalico.GlobalNetworkPolicyList, s conversion.Scope) error {
	return autoConvert_v3_GlobalNetworkPolicyList_To_projectcalico_GlobalNetworkPolicyList(in, out, s)
}

func autoConvert_projectcalico_GlobalNetworkPolicyList_To_v3_GlobalNetworkPolicyList(in *projectcalico.GlobalNetworkPolicyList, out *GlobalNetworkPolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]GlobalNetworkPolicy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_GlobalNetworkPolicyList_To_v3_GlobalNetworkPolicyList is an autogenerated conversion function.
func Convert_projectcalico_GlobalNetworkPolicyList_To_v3_GlobalNetworkPolicyList(in *projectcalico.GlobalNetworkPolicyList, out *GlobalNetworkPolicyList, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalNetworkPolicyList_To_v3_GlobalNetworkPolicyList(in, out, s)
}

func autoConvert_v3_GlobalNetworkSet_To_projectcalico_GlobalNetworkSet(in *GlobalNetworkSet, out *projectcalico.GlobalNetworkSet, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_GlobalNetworkSet_To_projectcalico_GlobalNetworkSet is an autogenerated conversion function.
func Convert_v3_GlobalNetworkSet_To_projectcalico_GlobalNetworkSet(in *GlobalNetworkSet, out *projectcalico.GlobalNetworkSet, s conversion.Scope) error {
	return autoConvert_v3_GlobalNetworkSet_To_projectcalico_GlobalNetworkSet(in, out, s)
}

func autoConvert_projectcalico_GlobalNetworkSet_To_v3_GlobalNetworkSet(in *projectcalico.GlobalNetworkSet, out *GlobalNetworkSet, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_GlobalNetworkSet_To_v3_GlobalNetworkSet is an autogenerated conversion function.
func Convert_projectcalico_GlobalNetworkSet_To_v3_GlobalNetworkSet(in *projectcalico.GlobalNetworkSet, out *GlobalNetworkSet, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalNetworkSet_To_v3_GlobalNetworkSet(in, out, s)
}

func autoConvert_v3_GlobalNetworkSetList_To_projectcalico_GlobalNetworkSetList(in *GlobalNetworkSetList, out *projectcalico.GlobalNetworkSetList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.GlobalNetworkSet)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_GlobalNetworkSetList_To_projectcalico_GlobalNetworkSetList is an autogenerated conversion function.
func Convert_v3_GlobalNetworkSetList_To_projectcalico_GlobalNetworkSetList(in *GlobalNetworkSetList, out *projectcalico.GlobalNetworkSetList, s conversion.Scope) error {
	return autoConvert_v3_GlobalNetworkSetList_To_projectcalico_GlobalNetworkSetList(in, out, s)
}

func autoConvert_projectcalico_GlobalNetworkSetList_To_v3_GlobalNetworkSetList(in *projectcalico.GlobalNetworkSetList, out *GlobalNetworkSetList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]GlobalNetworkSet)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_GlobalNetworkSetList_To_v3_GlobalNetworkSetList is an autogenerated conversion function.
func Convert_projectcalico_GlobalNetworkSetList_To_v3_GlobalNetworkSetList(in *projectcalico.GlobalNetworkSetList, out *GlobalNetworkSetList, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalNetworkSetList_To_v3_GlobalNetworkSetList(in, out, s)
}

func autoConvert_v3_GlobalReport_To_projectcalico_GlobalReport(in *GlobalReport, out *projectcalico.GlobalReport, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_v3_GlobalReport_To_projectcalico_GlobalReport is an autogenerated conversion function.
func Convert_v3_GlobalReport_To_projectcalico_GlobalReport(in *GlobalReport, out *projectcalico.GlobalReport, s conversion.Scope) error {
	return autoConvert_v3_GlobalReport_To_projectcalico_GlobalReport(in, out, s)
}

func autoConvert_projectcalico_GlobalReport_To_v3_GlobalReport(in *projectcalico.GlobalReport, out *GlobalReport, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_projectcalico_GlobalReport_To_v3_GlobalReport is an autogenerated conversion function.
func Convert_projectcalico_GlobalReport_To_v3_GlobalReport(in *projectcalico.GlobalReport, out *GlobalReport, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalReport_To_v3_GlobalReport(in, out, s)
}

func autoConvert_v3_GlobalReportList_To_projectcalico_GlobalReportList(in *GlobalReportList, out *projectcalico.GlobalReportList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.GlobalReport)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_GlobalReportList_To_projectcalico_GlobalReportList is an autogenerated conversion function.
func Convert_v3_GlobalReportList_To_projectcalico_GlobalReportList(in *GlobalReportList, out *projectcalico.GlobalReportList, s conversion.Scope) error {
	return autoConvert_v3_GlobalReportList_To_projectcalico_GlobalReportList(in, out, s)
}

func autoConvert_projectcalico_GlobalReportList_To_v3_GlobalReportList(in *projectcalico.GlobalReportList, out *GlobalReportList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]GlobalReport)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_GlobalReportList_To_v3_GlobalReportList is an autogenerated conversion function.
func Convert_projectcalico_GlobalReportList_To_v3_GlobalReportList(in *projectcalico.GlobalReportList, out *GlobalReportList, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalReportList_To_v3_GlobalReportList(in, out, s)
}

func autoConvert_v3_GlobalReportType_To_projectcalico_GlobalReportType(in *GlobalReportType, out *projectcalico.GlobalReportType, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_GlobalReportType_To_projectcalico_GlobalReportType is an autogenerated conversion function.
func Convert_v3_GlobalReportType_To_projectcalico_GlobalReportType(in *GlobalReportType, out *projectcalico.GlobalReportType, s conversion.Scope) error {
	return autoConvert_v3_GlobalReportType_To_projectcalico_GlobalReportType(in, out, s)
}

func autoConvert_projectcalico_GlobalReportType_To_v3_GlobalReportType(in *projectcalico.GlobalReportType, out *GlobalReportType, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_GlobalReportType_To_v3_GlobalReportType is an autogenerated conversion function.
func Convert_projectcalico_GlobalReportType_To_v3_GlobalReportType(in *projectcalico.GlobalReportType, out *GlobalReportType, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalReportType_To_v3_GlobalReportType(in, out, s)
}

func autoConvert_v3_GlobalReportTypeList_To_projectcalico_GlobalReportTypeList(in *GlobalReportTypeList, out *projectcalico.GlobalReportTypeList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.GlobalReportType)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_GlobalReportTypeList_To_projectcalico_GlobalReportTypeList is an autogenerated conversion function.
func Convert_v3_GlobalReportTypeList_To_projectcalico_GlobalReportTypeList(in *GlobalReportTypeList, out *projectcalico.GlobalReportTypeList, s conversion.Scope) error {
	return autoConvert_v3_GlobalReportTypeList_To_projectcalico_GlobalReportTypeList(in, out, s)
}

func autoConvert_projectcalico_GlobalReportTypeList_To_v3_GlobalReportTypeList(in *projectcalico.GlobalReportTypeList, out *GlobalReportTypeList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]GlobalReportType)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_GlobalReportTypeList_To_v3_GlobalReportTypeList is an autogenerated conversion function.
func Convert_projectcalico_GlobalReportTypeList_To_v3_GlobalReportTypeList(in *projectcalico.GlobalReportTypeList, out *GlobalReportTypeList, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalReportTypeList_To_v3_GlobalReportTypeList(in, out, s)
}

func autoConvert_v3_GlobalThreatFeed_To_projectcalico_GlobalThreatFeed(in *GlobalThreatFeed, out *projectcalico.GlobalThreatFeed, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_v3_GlobalThreatFeed_To_projectcalico_GlobalThreatFeed is an autogenerated conversion function.
func Convert_v3_GlobalThreatFeed_To_projectcalico_GlobalThreatFeed(in *GlobalThreatFeed, out *projectcalico.GlobalThreatFeed, s conversion.Scope) error {
	return autoConvert_v3_GlobalThreatFeed_To_projectcalico_GlobalThreatFeed(in, out, s)
}

func autoConvert_projectcalico_GlobalThreatFeed_To_v3_GlobalThreatFeed(in *projectcalico.GlobalThreatFeed, out *GlobalThreatFeed, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
	return nil
}

// Convert_projectcalico_GlobalThreatFeed_To_v3_GlobalThreatFeed is an autogenerated conversion function.
func Convert_projectcalico_GlobalThreatFeed_To_v3_GlobalThreatFeed(in *projectcalico.GlobalThreatFeed, out *GlobalThreatFeed, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalThreatFeed_To_v3_GlobalThreatFeed(in, out, s)
}

func autoConvert_v3_GlobalThreatFeedList_To_projectcalico_GlobalThreatFeedList(in *GlobalThreatFeedList, out *projectcalico.GlobalThreatFeedList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.GlobalThreatFeed)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_GlobalThreatFeedList_To_projectcalico_GlobalThreatFeedList is an autogenerated conversion function.
func Convert_v3_GlobalThreatFeedList_To_projectcalico_GlobalThreatFeedList(in *GlobalThreatFeedList, out *projectcalico.GlobalThreatFeedList, s conversion.Scope) error {
	return autoConvert_v3_GlobalThreatFeedList_To_projectcalico_GlobalThreatFeedList(in, out, s)
}

func autoConvert_projectcalico_GlobalThreatFeedList_To_v3_GlobalThreatFeedList(in *projectcalico.GlobalThreatFeedList, out *GlobalThreatFeedList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]GlobalThreatFeed)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_GlobalThreatFeedList_To_v3_GlobalThreatFeedList is an autogenerated conversion function.
func Convert_projectcalico_GlobalThreatFeedList_To_v3_GlobalThreatFeedList(in *projectcalico.GlobalThreatFeedList, out *GlobalThreatFeedList, s conversion.Scope) error {
	return autoConvert_projectcalico_GlobalThreatFeedList_To_v3_GlobalThreatFeedList(in, out, s)
}

func autoConvert_v3_HostEndpoint_To_projectcalico_HostEndpoint(in *HostEndpoint, out *projectcalico.HostEndpoint, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_HostEndpoint_To_projectcalico_HostEndpoint is an autogenerated conversion function.
func Convert_v3_HostEndpoint_To_projectcalico_HostEndpoint(in *HostEndpoint, out *projectcalico.HostEndpoint, s conversion.Scope) error {
	return autoConvert_v3_HostEndpoint_To_projectcalico_HostEndpoint(in, out, s)
}

func autoConvert_projectcalico_HostEndpoint_To_v3_HostEndpoint(in *projectcalico.HostEndpoint, out *HostEndpoint, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_HostEndpoint_To_v3_HostEndpoint is an autogenerated conversion function.
func Convert_projectcalico_HostEndpoint_To_v3_HostEndpoint(in *projectcalico.HostEndpoint, out *HostEndpoint, s conversion.Scope) error {
	return autoConvert_projectcalico_HostEndpoint_To_v3_HostEndpoint(in, out, s)
}

func autoConvert_v3_HostEndpointList_To_projectcalico_HostEndpointList(in *HostEndpointList, out *projectcalico.HostEndpointList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.HostEndpoint)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_HostEndpointList_To_projectcalico_HostEndpointList is an autogenerated conversion function.
func Convert_v3_HostEndpointList_To_projectcalico_HostEndpointList(in *HostEndpointList, out *projectcalico.HostEndpointList, s conversion.Scope) error {
	return autoConvert_v3_HostEndpointList_To_projectcalico_HostEndpointList(in, out, s)
}

func autoConvert_projectcalico_HostEndpointList_To_v3_HostEndpointList(in *projectcalico.HostEndpointList, out *HostEndpointList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]HostEndpoint)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_HostEndpointList_To_v3_HostEndpointList is an autogenerated conversion function.
func Convert_projectcalico_HostEndpointList_To_v3_HostEndpointList(in *projectcalico.HostEndpointList, out *HostEndpointList, s conversion.Scope) error {
	return autoConvert_projectcalico_HostEndpointList_To_v3_HostEndpointList(in, out, s)
}

func autoConvert_v3_IPPool_To_projectcalico_IPPool(in *IPPool, out *projectcalico.IPPool, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_IPPool_To_projectcalico_IPPool is an autogenerated conversion function.
func Convert_v3_IPPool_To_projectcalico_IPPool(in *IPPool, out *projectcalico.IPPool, s conversion.Scope) error {
	return autoConvert_v3_IPPool_To_projectcalico_IPPool(in, out, s)
}

func autoConvert_projectcalico_IPPool_To_v3_IPPool(in *projectcalico.IPPool, out *IPPool, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_IPPool_To_v3_IPPool is an autogenerated conversion function.
func Convert_projectcalico_IPPool_To_v3_IPPool(in *projectcalico.IPPool, out *IPPool, s conversion.Scope) error {
	return autoConvert_projectcalico_IPPool_To_v3_IPPool(in, out, s)
}

func autoConvert_v3_IPPoolList_To_projectcalico_IPPoolList(in *IPPoolList, out *projectcalico.IPPoolList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.IPPool)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_IPPoolList_To_projectcalico_IPPoolList is an autogenerated conversion function.
func Convert_v3_IPPoolList_To_projectcalico_IPPoolList(in *IPPoolList, out *projectcalico.IPPoolList, s conversion.Scope) error {
	return autoConvert_v3_IPPoolList_To_projectcalico_IPPoolList(in, out, s)
}

func autoConvert_projectcalico_IPPoolList_To_v3_IPPoolList(in *projectcalico.IPPoolList, out *IPPoolList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]IPPool)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_IPPoolList_To_v3_IPPoolList is an autogenerated conversion function.
func Convert_projectcalico_IPPoolList_To_v3_IPPoolList(in *projectcalico.IPPoolList, out *IPPoolList, s conversion.Scope) error {
	return autoConvert_projectcalico_IPPoolList_To_v3_IPPoolList(in, out, s)
}

func autoConvert_v3_LicenseKey_To_projectcalico_LicenseKey(in *LicenseKey, out *projectcalico.LicenseKey, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_LicenseKey_To_projectcalico_LicenseKey is an autogenerated conversion function.
func Convert_v3_LicenseKey_To_projectcalico_LicenseKey(in *LicenseKey, out *projectcalico.LicenseKey, s conversion.Scope) error {
	return autoConvert_v3_LicenseKey_To_projectcalico_LicenseKey(in, out, s)
}

func autoConvert_projectcalico_LicenseKey_To_v3_LicenseKey(in *projectcalico.LicenseKey, out *LicenseKey, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_LicenseKey_To_v3_LicenseKey is an autogenerated conversion function.
func Convert_projectcalico_LicenseKey_To_v3_LicenseKey(in *projectcalico.LicenseKey, out *LicenseKey, s conversion.Scope) error {
	return autoConvert_projectcalico_LicenseKey_To_v3_LicenseKey(in, out, s)
}

func autoConvert_v3_LicenseKeyList_To_projectcalico_LicenseKeyList(in *LicenseKeyList, out *projectcalico.LicenseKeyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.LicenseKey)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_LicenseKeyList_To_projectcalico_LicenseKeyList is an autogenerated conversion function.
func Convert_v3_LicenseKeyList_To_projectcalico_LicenseKeyList(in *LicenseKeyList, out *projectcalico.LicenseKeyList, s conversion.Scope) error {
	return autoConvert_v3_LicenseKeyList_To_projectcalico_LicenseKeyList(in, out, s)
}

func autoConvert_projectcalico_LicenseKeyList_To_v3_LicenseKeyList(in *projectcalico.LicenseKeyList, out *LicenseKeyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]LicenseKey)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_LicenseKeyList_To_v3_LicenseKeyList is an autogenerated conversion function.
func Convert_projectcalico_LicenseKeyList_To_v3_LicenseKeyList(in *projectcalico.LicenseKeyList, out *LicenseKeyList, s conversion.Scope) error {
	return autoConvert_projectcalico_LicenseKeyList_To_v3_LicenseKeyList(in, out, s)
}

func autoConvert_v3_NetworkPolicy_To_projectcalico_NetworkPolicy(in *NetworkPolicy, out *projectcalico.NetworkPolicy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_NetworkPolicy_To_projectcalico_NetworkPolicy is an autogenerated conversion function.
func Convert_v3_NetworkPolicy_To_projectcalico_NetworkPolicy(in *NetworkPolicy, out *projectcalico.NetworkPolicy, s conversion.Scope) error {
	return autoConvert_v3_NetworkPolicy_To_projectcalico_NetworkPolicy(in, out, s)
}

func autoConvert_projectcalico_NetworkPolicy_To_v3_NetworkPolicy(in *projectcalico.NetworkPolicy, out *NetworkPolicy, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_NetworkPolicy_To_v3_NetworkPolicy is an autogenerated conversion function.
func Convert_projectcalico_NetworkPolicy_To_v3_NetworkPolicy(in *projectcalico.NetworkPolicy, out *NetworkPolicy, s conversion.Scope) error {
	return autoConvert_projectcalico_NetworkPolicy_To_v3_NetworkPolicy(in, out, s)
}

func autoConvert_v3_NetworkPolicyList_To_projectcalico_NetworkPolicyList(in *NetworkPolicyList, out *projectcalico.NetworkPolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.NetworkPolicy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_NetworkPolicyList_To_projectcalico_NetworkPolicyList is an autogenerated conversion function.
func Convert_v3_NetworkPolicyList_To_projectcalico_NetworkPolicyList(in *NetworkPolicyList, out *projectcalico.NetworkPolicyList, s conversion.Scope) error {
	return autoConvert_v3_NetworkPolicyList_To_projectcalico_NetworkPolicyList(in, out, s)
}

func autoConvert_projectcalico_NetworkPolicyList_To_v3_NetworkPolicyList(in *projectcalico.NetworkPolicyList, out *NetworkPolicyList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]NetworkPolicy)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_NetworkPolicyList_To_v3_NetworkPolicyList is an autogenerated conversion function.
func Convert_projectcalico_NetworkPolicyList_To_v3_NetworkPolicyList(in *projectcalico.NetworkPolicyList, out *NetworkPolicyList, s conversion.Scope) error {
	return autoConvert_projectcalico_NetworkPolicyList_To_v3_NetworkPolicyList(in, out, s)
}

func autoConvert_v3_Tier_To_projectcalico_Tier(in *Tier, out *projectcalico.Tier, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_v3_Tier_To_projectcalico_Tier is an autogenerated conversion function.
func Convert_v3_Tier_To_projectcalico_Tier(in *Tier, out *projectcalico.Tier, s conversion.Scope) error {
	return autoConvert_v3_Tier_To_projectcalico_Tier(in, out, s)
}

func autoConvert_projectcalico_Tier_To_v3_Tier(in *projectcalico.Tier, out *Tier, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	return nil
}

// Convert_projectcalico_Tier_To_v3_Tier is an autogenerated conversion function.
func Convert_projectcalico_Tier_To_v3_Tier(in *projectcalico.Tier, out *Tier, s conversion.Scope) error {
	return autoConvert_projectcalico_Tier_To_v3_Tier(in, out, s)
}

func autoConvert_v3_TierList_To_projectcalico_TierList(in *TierList, out *projectcalico.TierList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]projectcalico.Tier)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_v3_TierList_To_projectcalico_TierList is an autogenerated conversion function.
func Convert_v3_TierList_To_projectcalico_TierList(in *TierList, out *projectcalico.TierList, s conversion.Scope) error {
	return autoConvert_v3_TierList_To_projectcalico_TierList(in, out, s)
}

func autoConvert_projectcalico_TierList_To_v3_TierList(in *projectcalico.TierList, out *TierList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	out.Items = *(*[]Tier)(unsafe.Pointer(&in.Items))
	return nil
}

// Convert_projectcalico_TierList_To_v3_TierList is an autogenerated conversion function.
func Convert_projectcalico_TierList_To_v3_TierList(in *projectcalico.TierList, out *TierList, s conversion.Scope) error {
	return autoConvert_projectcalico_TierList_To_v3_TierList(in, out, s)
}
