// Copyright (c) 2019-2022 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"k8s.io/klog/v2"

	libapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
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
	case *v3.Tier:
		lcgTier := libcalicoObject.(*v3.Tier)
		aapiTier := &v3.Tier{}
		TierConverter{}.convertToAAPI(lcgTier, aapiTier)
		return aapiTier
	case *v3.NetworkPolicy:
		lcgPolicy := libcalicoObject.(*v3.NetworkPolicy)
		aapiPolicy := &v3.NetworkPolicy{}
		NetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *v3.StagedKubernetesNetworkPolicy:
		lcgPolicy := libcalicoObject.(*v3.StagedKubernetesNetworkPolicy)
		aapiPolicy := &v3.StagedKubernetesNetworkPolicy{}
		StagedKubernetesNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *v3.StagedNetworkPolicy:
		lcgPolicy := libcalicoObject.(*v3.StagedNetworkPolicy)
		aapiPolicy := &v3.StagedNetworkPolicy{}
		StagedNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *v3.GlobalNetworkPolicy:
		lcgPolicy := libcalicoObject.(*v3.GlobalNetworkPolicy)
		aapiPolicy := &v3.GlobalNetworkPolicy{}
		GlobalNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *v3.StagedGlobalNetworkPolicy:
		lcgPolicy := libcalicoObject.(*v3.StagedGlobalNetworkPolicy)
		aapiPolicy := &v3.StagedGlobalNetworkPolicy{}
		StagedGlobalNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *v3.GlobalNetworkSet:
		lcgNetworkSet := libcalicoObject.(*v3.GlobalNetworkSet)
		aapiNetworkSet := &v3.GlobalNetworkSet{}
		GlobalNetworkSetConverter{}.convertToAAPI(lcgNetworkSet, aapiNetworkSet)
		return aapiNetworkSet
	case *v3.NetworkSet:
		lcgNetworkSet := libcalicoObject.(*v3.NetworkSet)
		aapiNetworkSet := &v3.NetworkSet{}
		NetworkSetConverter{}.convertToAAPI(lcgNetworkSet, aapiNetworkSet)
		return aapiNetworkSet
	case *v3.LicenseKey:
		lcgLicense := libcalicoObject.(*v3.LicenseKey)
		aapiLicenseKey := &v3.LicenseKey{}
		LicenseKeyConverter{}.convertToAAPI(lcgLicense, aapiLicenseKey)
		return aapiLicenseKey
	case *v3.AlertException:
		lcg := libcalicoObject.(*v3.AlertException)
		aapi := &v3.AlertException{}
		AlertExceptionConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.GlobalAlert:
		lcg := libcalicoObject.(*v3.GlobalAlert)
		aapi := &v3.GlobalAlert{}
		GlobalAlertConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.GlobalAlertTemplate:
		lcg := libcalicoObject.(*v3.GlobalAlertTemplate)
		aapi := &v3.GlobalAlertTemplate{}
		GlobalAlertTemplateConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.GlobalThreatFeed:
		lcg := libcalicoObject.(*v3.GlobalThreatFeed)
		aapi := &v3.GlobalThreatFeed{}
		GlobalThreatFeedConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.HostEndpoint:
		lcg := libcalicoObject.(*v3.HostEndpoint)
		aapi := &v3.HostEndpoint{}
		HostEndpointConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.GlobalReport:
		lcg := libcalicoObject.(*v3.GlobalReport)
		aapi := &v3.GlobalReport{}
		GlobalReportConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.GlobalReportType:
		lcg := libcalicoObject.(*v3.GlobalReportType)
		aapi := &v3.GlobalReportType{}
		GlobalReportTypeConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.IPPool:
		lcg := libcalicoObject.(*v3.IPPool)
		aapi := &v3.IPPool{}
		IPPoolConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.IPReservation:
		lcg := libcalicoObject.(*v3.IPReservation)
		aapi := &v3.IPReservation{}
		IPReservationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.BGPConfiguration:
		lcg := libcalicoObject.(*v3.BGPConfiguration)
		aapi := &v3.BGPConfiguration{}
		BGPConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.BGPPeer:
		lcg := libcalicoObject.(*v3.BGPPeer)
		aapi := &v3.BGPPeer{}
		BGPPeerConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.BGPFilter:
		lcg := libcalicoObject.(*v3.BGPFilter)
		aapi := &v3.BGPFilter{}
		BGPFilterConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.Profile:
		lcg := libcalicoObject.(*v3.Profile)
		aapi := &v3.Profile{}
		ProfileConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.RemoteClusterConfiguration:
		lcg := libcalicoObject.(*v3.RemoteClusterConfiguration)
		aapi := &v3.RemoteClusterConfiguration{}
		RemoteClusterConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.FelixConfiguration:
		lcg := libcalicoObject.(*v3.FelixConfiguration)
		aapi := &v3.FelixConfiguration{}
		FelixConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.KubeControllersConfiguration:
		lcg := libcalicoObject.(*v3.KubeControllersConfiguration)
		aapi := &v3.KubeControllersConfiguration{}
		KubeControllersConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.ManagedCluster:
		lcg := libcalicoObject.(*v3.ManagedCluster)
		aapi := &v3.ManagedCluster{}
		ManagedClusterConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.ClusterInformation:
		lcg := libcalicoObject.(*v3.ClusterInformation)
		aapi := &v3.ClusterInformation{}
		ClusterInformationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.PacketCapture:
		lcg := libcalicoObject.(*v3.PacketCapture)
		aapi := &v3.PacketCapture{}
		PacketCaptureConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.DeepPacketInspection:
		lcg := libcalicoObject.(*v3.DeepPacketInspection)
		aapi := &v3.DeepPacketInspection{}
		DeepPacketInspectionConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.UISettingsGroup:
		api := libcalicoObject.(*v3.UISettingsGroup)
		aapi := &v3.UISettingsGroup{}
		UISettingsGroupConverter{}.convertToAAPI(api, aapi)
		return aapi
	case *v3.UISettings:
		api := libcalicoObject.(*v3.UISettings)
		aapi := &v3.UISettings{}
		UISettingsConverter{}.convertToAAPI(api, aapi)
		return aapi
	case *v3.CalicoNodeStatus:
		lcg := libcalicoObject.(*v3.CalicoNodeStatus)
		aapi := &v3.CalicoNodeStatus{}
		CalicoNodeStatusConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libapi.IPAMConfig:
		lcg := libcalicoObject.(*libapi.IPAMConfig)
		aapi := &v3.IPAMConfiguration{}
		IPAMConfigConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	// BlockAffinity works off of the libapi objects since
	// the v3 client is used for mostly internal operations.
	case *libapi.BlockAffinity:
		lcg := libcalicoObject.(*libapi.BlockAffinity)
		aapi := &v3.BlockAffinity{}
		BlockAffinityConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *v3.ExternalNetwork:
		lcg := libcalicoObject.(*v3.ExternalNetwork)
		aapi := &v3.ExternalNetwork{}
		ExternalNetworkConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	default:
		klog.Infof("Unrecognized libcalico object (type %v)", reflect.TypeOf(libcalicoObject))
		return nil
	}
}
