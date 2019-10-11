// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"reflect"

	"github.com/golang/glog"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/errors"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
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
	case *libcalicoapi.Tier:
		lcgTier := libcalicoObject.(*libcalicoapi.Tier)
		aapiTier := &aapi.Tier{}
		TierConverter{}.convertToAAPI(lcgTier, aapiTier)
		return aapiTier
	case *libcalicoapi.NetworkPolicy:
		lcgPolicy := libcalicoObject.(*libcalicoapi.NetworkPolicy)
		aapiPolicy := &aapi.NetworkPolicy{}
		NetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *libcalicoapi.GlobalNetworkPolicy:
		lcgPolicy := libcalicoObject.(*libcalicoapi.GlobalNetworkPolicy)
		aapiPolicy := &aapi.GlobalNetworkPolicy{}
		GlobalNetworkPolicyConverter{}.convertToAAPI(lcgPolicy, aapiPolicy)
		return aapiPolicy
	case *libcalicoapi.GlobalNetworkSet:
		lcgNetworkSet := libcalicoObject.(*libcalicoapi.GlobalNetworkSet)
		aapiNetworkSet := &aapi.GlobalNetworkSet{}
		GlobalNetworkSetConverter{}.convertToAAPI(lcgNetworkSet, aapiNetworkSet)
		return aapiNetworkSet
	case *libcalicoapi.NetworkSet:
		lcgNetworkSet := libcalicoObject.(*libcalicoapi.NetworkSet)
		aapiNetworkSet := &aapi.NetworkSet{}
		NetworkSetConverter{}.convertToAAPI(lcgNetworkSet, aapiNetworkSet)
		return aapiNetworkSet
	case *libcalicoapi.LicenseKey:
		lcgLicense := libcalicoObject.(*libcalicoapi.LicenseKey)
		aapiLicenseKey := &aapi.LicenseKey{}
		LicenseKeyConverter{}.convertToAAPI(lcgLicense, aapiLicenseKey)
		return aapiLicenseKey
	case *libcalicoapi.GlobalAlert:
		lcg := libcalicoObject.(*libcalicoapi.GlobalAlert)
		aapi := &aapi.GlobalAlert{}
		GlobalAlertConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.GlobalThreatFeed:
		lcg := libcalicoObject.(*libcalicoapi.GlobalThreatFeed)
		aapi := &aapi.GlobalThreatFeed{}
		GlobalThreatFeedConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.HostEndpoint:
		lcg := libcalicoObject.(*libcalicoapi.HostEndpoint)
		aapi := &aapi.HostEndpoint{}
		HostEndpointConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.GlobalReport:
		lcg := libcalicoObject.(*libcalicoapi.GlobalReport)
		aapi := &aapi.GlobalReport{}
		GlobalReportConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.GlobalReportType:
		lcg := libcalicoObject.(*libcalicoapi.GlobalReportType)
		aapi := &aapi.GlobalReportType{}
		GlobalReportTypeConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.IPPool:
		lcg := libcalicoObject.(*libcalicoapi.IPPool)
		aapi := &aapi.IPPool{}
		IPPoolConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.BGPConfiguration:
		lcg := libcalicoObject.(*libcalicoapi.BGPConfiguration)
		aapi := &aapi.BGPConfiguration{}
		BGPConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.BGPPeer:
		lcg := libcalicoObject.(*libcalicoapi.BGPPeer)
		aapi := &aapi.BGPPeer{}
		BGPPeerConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.Profile:
		lcg := libcalicoObject.(*libcalicoapi.Profile)
		aapi := &aapi.Profile{}
		ProfileConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.RemoteClusterConfiguration:
		lcg := libcalicoObject.(*libcalicoapi.RemoteClusterConfiguration)
		aapi := &aapi.RemoteClusterConfiguration{}
		RemoteClusterConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.FelixConfiguration:
		lcg := libcalicoObject.(*libcalicoapi.FelixConfiguration)
		aapi := &aapi.FelixConfiguration{}
		FelixConfigurationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.ManagedCluster:
		lcg := libcalicoObject.(*libcalicoapi.ManagedCluster)
		aapi := &aapi.ManagedCluster{}
		ManagedClusterConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	case *libcalicoapi.ClusterInformation:
		lcg := libcalicoObject.(*libcalicoapi.ClusterInformation)
		aapi := &aapi.ClusterInformation{}
		ClusterInformationConverter{}.convertToAAPI(lcg, aapi)
		return aapi
	default:
		glog.Infof("Unrecognized libcalico object (type %v)", reflect.TypeOf(libcalicoObject))
		return nil
	}
}
