// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"k8s.io/klog"

	"github.com/projectcalico/libcalico-go/lib/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	aapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func aapiError(err error, key string) error {
	switch err.(type) {
	case errors.ErrorResourceAlreadyExists:
		return storage.NewKeyExistsError(key, 0)
	case errors.ErrorResourceDoesNotExist:
		return storage.NewKeyNotFoundError(key, 0)
	case errors.ErrorResourceUpdateConflict:
		return storage.NewResourceVersionConflictsError(key, 0)
	default:
		return err
	}
}

// TODO: convertToAAPI should be same as the ones specific to resources.
// This is common code. Refactor this workflow.
func convertToAAPI(libcalicoObject runtime.Object) (res runtime.Object) {
	switch libcalicoObject.(type) {
	case *api.Tier:
		lcgTier := libcalicoObject.(*api.Tier)
		aapiTier := &aapi.Tier{}
		TierConverter{}.convertToAAPI(lcgTier, aapiTier)
		return aapiTier
	case *api.NetworkPolicy:
		lcgPolicy := libcalicoObject.(*api.NetworkPolicy)
		aapiPolicy := &aapi.NetworkPolicy{}
		NetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *api.StagedKubernetesNetworkPolicy:
		lcgPolicy := libcalicoObject.(*api.StagedKubernetesNetworkPolicy)
		aapiPolicy := &aapi.StagedKubernetesNetworkPolicy{}
		StagedKubernetesNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *api.StagedNetworkPolicy:
		lcgPolicy := libcalicoObject.(*api.StagedNetworkPolicy)
		aapiPolicy := &aapi.StagedNetworkPolicy{}
		StagedNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *api.GlobalNetworkPolicy:
		lcgPolicy := libcalicoObject.(*api.GlobalNetworkPolicy)
		aapiPolicy := &aapi.GlobalNetworkPolicy{}
		GlobalNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *api.StagedGlobalNetworkPolicy:
		lcgPolicy := libcalicoObject.(*api.StagedGlobalNetworkPolicy)
		aapiPolicy := &aapi.StagedGlobalNetworkPolicy{}
		StagedGlobalNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *api.GlobalNetworkSet:
		lcgNetworkSet := libcalicoObject.(*api.GlobalNetworkSet)
		aapiNetworkSet := &aapi.GlobalNetworkSet{}
		GlobalNetworkSetConverter{}.convertToAAPI(lcgNetworkSet, aapiNetworkSet)
		return aapiNetworkSet
	case *api.NetworkSet:
		lcgNetworkSet := libcalicoObject.(*api.NetworkSet)
		aapiNetworkSet := &aapi.NetworkSet{}
		NetworkSetConverter{}.convertToAAPI(lcgNetworkSet, aapiNetworkSet)
		return aapiNetworkSet
	case *api.LicenseKey:
		lcgLicense := libcalicoObject.(*api.LicenseKey)
		aapiLicenseKey := &aapi.LicenseKey{}
		LicenseKeyConverter{}.convertToAAPI(lcgLicense, aapiLicenseKey)
		return aapiLicenseKey
	case *api.GlobalAlert:
		lcg := libcalicoObject.(*api.GlobalAlert)
		aapi := &aapi.GlobalAlert{}
		GlobalAlertConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.GlobalAlertTemplate:
		lcg := libcalicoObject.(*api.GlobalAlertTemplate)
		aapi := &aapi.GlobalAlertTemplate{}
		GlobalAlertTemplateConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.GlobalThreatFeed:
		lcg := libcalicoObject.(*api.GlobalThreatFeed)
		aapi := &aapi.GlobalThreatFeed{}
		GlobalThreatFeedConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.HostEndpoint:
		lcg := libcalicoObject.(*api.HostEndpoint)
		aapi := &aapi.HostEndpoint{}
		HostEndpointConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.GlobalReport:
		lcg := libcalicoObject.(*api.GlobalReport)
		aapi := &aapi.GlobalReport{}
		GlobalReportConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.GlobalReportType:
		lcg := libcalicoObject.(*api.GlobalReportType)
		aapi := &aapi.GlobalReportType{}
		GlobalReportTypeConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.IPPool:
		lcg := libcalicoObject.(*api.IPPool)
		aapi := &aapi.IPPool{}
		IPPoolConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.BGPConfiguration:
		lcg := libcalicoObject.(*api.BGPConfiguration)
		aapi := &aapi.BGPConfiguration{}
		BGPConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.BGPPeer:
		lcg := libcalicoObject.(*api.BGPPeer)
		aapi := &aapi.BGPPeer{}
		BGPPeerConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.Profile:
		lcg := libcalicoObject.(*api.Profile)
		aapi := &aapi.Profile{}
		ProfileConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.RemoteClusterConfiguration:
		lcg := libcalicoObject.(*api.RemoteClusterConfiguration)
		aapi := &aapi.RemoteClusterConfiguration{}
		RemoteClusterConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.FelixConfiguration:
		lcg := libcalicoObject.(*api.FelixConfiguration)
		aapi := &aapi.FelixConfiguration{}
		FelixConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.KubeControllersConfiguration:
		lcg := libcalicoObject.(*api.KubeControllersConfiguration)
		aapi := &aapi.KubeControllersConfiguration{}
		KubeControllersConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.ManagedCluster:
		lcg := libcalicoObject.(*api.ManagedCluster)
		aapi := &aapi.ManagedCluster{}
		ManagedClusterConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.ClusterInformation:
		lcg := libcalicoObject.(*api.ClusterInformation)
		aapi := &aapi.ClusterInformation{}
		ClusterInformationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.PacketCapture:
		lcg := libcalicoObject.(*api.PacketCapture)
		aapi := &aapi.PacketCapture{}
		PacketCaptureConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.DeepPacketInspection:
		lcg := libcalicoObject.(*api.DeepPacketInspection)
		aapi := &aapi.DeepPacketInspection{}
		DeepPacketInspectionConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *api.UISettingsGroup:
		api := libcalicoObject.(*api.UISettingsGroup)
		aapi := &aapi.UISettingsGroup{}
		UISettingsGroupConverter{}.convertToAAPI(api, aapi)
		return aapi
	default:
		klog.Infof("Unrecognized libcalico object (type %v)", reflect.TypeOf(libcalicoObject))
		return nil
	}
}
