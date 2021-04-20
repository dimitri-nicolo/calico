// Copyright (c) 2016-2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"

	calico "github.com/projectcalico/apiserver/pkg/apis/projectcalico"
	"github.com/projectcalico/apiserver/pkg/rbac"
	calicoauthenticationreview "github.com/projectcalico/apiserver/pkg/registry/projectcalico/authenticationreview"
	calicoauthorizationreview "github.com/projectcalico/apiserver/pkg/registry/projectcalico/authorizationreview"
	calicobgpconfiguration "github.com/projectcalico/apiserver/pkg/registry/projectcalico/bgpconfiguration"
	calicobgppeer "github.com/projectcalico/apiserver/pkg/registry/projectcalico/bgppeer"
	calicoclusterinformation "github.com/projectcalico/apiserver/pkg/registry/projectcalico/clusterinformation"
	calicofelixconfig "github.com/projectcalico/apiserver/pkg/registry/projectcalico/felixconfig"
	calicogalert "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalalert"
	calicogalerttemplate "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalalerttemplate"
	calicognetworkset "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalnetworkset"
	calicogpolicy "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalpolicy"
	calicoglobalreport "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalreport"
	calicoglobalreporttype "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalreporttype"
	calicogthreatfeed "github.com/projectcalico/apiserver/pkg/registry/projectcalico/globalthreatfeed"
	calicohostendpoint "github.com/projectcalico/apiserver/pkg/registry/projectcalico/hostendpoint"
	calicoippool "github.com/projectcalico/apiserver/pkg/registry/projectcalico/ippool"
	calicokubecontrollersconfig "github.com/projectcalico/apiserver/pkg/registry/projectcalico/kubecontrollersconfig"
	calicolicensekey "github.com/projectcalico/apiserver/pkg/registry/projectcalico/licensekey"
	calicomanagedcluster "github.com/projectcalico/apiserver/pkg/registry/projectcalico/managedcluster"
	calicopolicy "github.com/projectcalico/apiserver/pkg/registry/projectcalico/networkpolicy"
	caliconetworkset "github.com/projectcalico/apiserver/pkg/registry/projectcalico/networkset"
	calicopacketcapture "github.com/projectcalico/apiserver/pkg/registry/projectcalico/packetcapture"
	calicoprofile "github.com/projectcalico/apiserver/pkg/registry/projectcalico/profile"
	calicoremoteclusterconfig "github.com/projectcalico/apiserver/pkg/registry/projectcalico/remoteclusterconfig"
	"github.com/projectcalico/apiserver/pkg/registry/projectcalico/server"
	calicostagedgpolicy "github.com/projectcalico/apiserver/pkg/registry/projectcalico/stagedglobalnetworkpolicy"
	calicostagedk8spolicy "github.com/projectcalico/apiserver/pkg/registry/projectcalico/stagedkubernetesnetworkpolicy"
	calicostagedpolicy "github.com/projectcalico/apiserver/pkg/registry/projectcalico/stagednetworkpolicy"
	calicotier "github.com/projectcalico/apiserver/pkg/registry/projectcalico/tier"
	calicostorage "github.com/projectcalico/apiserver/pkg/storage/calico"
	"github.com/projectcalico/apiserver/pkg/storage/etcd"
)

// RESTStorageProvider provides a factory method to create a new APIGroupInfo for
// the calico API group. It implements (./pkg/apiserver).RESTStorageProvider
type RESTStorageProvider struct {
	StorageType server.StorageType
}

// NewV3Storage constructs v3 api storage.
func (p RESTStorageProvider) NewV3Storage(
	scheme *runtime.Scheme,
	restOptionsGetter generic.RESTOptionsGetter,
	authorizer authorizer.Authorizer,
	resources *calicostorage.ManagedClusterResources,
	calculator rbac.Calculator,
	licenseCache calicostorage.LicenseCache,
) (map[string]rest.Storage, error) {
	policyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("networkpolicies"))
	if err != nil {
		return nil, err
	}
	policyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   policyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicopolicy.EmptyObject(),
			ScopeStrategy: calicopolicy.NewStrategy(scheme),
			NewListFunc:   calicopolicy.NewList,
			GetAttrsFunc:  calicopolicy.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  policyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"cnp", "caliconetworkpolicy", "caliconetworkpolicies"},
	)

	networksetRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("networksets"))
	if err != nil {
		return nil, err
	}
	networksetOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   networksetRESTOptions,
			Capacity:      1000,
			ObjectType:    caliconetworkset.EmptyObject(),
			ScopeStrategy: caliconetworkset.NewStrategy(scheme),
			NewListFunc:   caliconetworkset.NewList,
			GetAttrsFunc:  caliconetworkset.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  networksetRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"netsets"},
	)

	stagedk8spolicyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("stagedkubernetesnetworkpolicies"))
	if err != nil {
		return nil, err
	}
	stagedk8spolicyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   stagedk8spolicyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicostagedk8spolicy.EmptyObject(),
			ScopeStrategy: calicostagedk8spolicy.NewStrategy(scheme),
			NewListFunc:   calicostagedk8spolicy.NewList,
			GetAttrsFunc:  calicostagedk8spolicy.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  stagedk8spolicyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	stagedpolicyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("stagednetworkpolicies"))
	if err != nil {
		return nil, err
	}
	stagedpolicyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   stagedpolicyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicostagedpolicy.EmptyObject(),
			ScopeStrategy: calicostagedpolicy.NewStrategy(scheme),
			NewListFunc:   calicostagedpolicy.NewList,
			GetAttrsFunc:  calicostagedpolicy.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  stagedpolicyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	tierRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("tiers"))
	if err != nil {
		return nil, err
	}
	tierOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   tierRESTOptions,
			Capacity:      1000,
			ObjectType:    calicotier.EmptyObject(),
			ScopeStrategy: calicotier.NewStrategy(scheme),
			NewListFunc:   calicotier.NewList,
			GetAttrsFunc:  calicotier.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  tierRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	gpolicyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalnetworkpolicies"))
	if err != nil {
		return nil, err
	}
	gpolicyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   gpolicyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicogpolicy.EmptyObject(),
			ScopeStrategy: calicogpolicy.NewStrategy(scheme),
			NewListFunc:   calicogpolicy.NewList,
			GetAttrsFunc:  calicogpolicy.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  gpolicyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"gnp", "cgnp", "calicoglobalnetworkpolicies"},
	)

	stagedgpolicyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("stagedglobalnetworkpolicies"))
	if err != nil {
		return nil, err
	}
	stagedgpolicyOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   stagedgpolicyRESTOptions,
			Capacity:      1000,
			ObjectType:    calicostagedgpolicy.EmptyObject(),
			ScopeStrategy: calicostagedgpolicy.NewStrategy(scheme),
			NewListFunc:   calicostagedgpolicy.NewList,
			GetAttrsFunc:  calicostagedgpolicy.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  stagedgpolicyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	gNetworkSetRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalnetworksets"))
	if err != nil {
		return nil, err
	}
	gNetworkSetOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   gNetworkSetRESTOptions,
			Capacity:      1000,
			ObjectType:    calicognetworkset.EmptyObject(),
			ScopeStrategy: calicognetworkset.NewStrategy(scheme),
			NewListFunc:   calicognetworkset.NewList,
			GetAttrsFunc:  calicognetworkset.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  gNetworkSetRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	licenseKeyRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("licensekeys"))
	if err != nil {
		return nil, err
	}
	licenseKeysSetOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   licenseKeyRESTOptions,
			Capacity:      10,
			ObjectType:    calicolicensekey.EmptyObject(),
			ScopeStrategy: calicolicensekey.NewStrategy(scheme),
			NewListFunc:   calicolicensekey.NewList,
			GetAttrsFunc:  calicolicensekey.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  licenseKeyRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	gAlertRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalalerts"))
	if err != nil {
		return nil, err
	}
	gAlertOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   gAlertRESTOptions,
			Capacity:      1000,
			ObjectType:    calicogalert.EmptyObject(),
			ScopeStrategy: calicogalert.NewStrategy(scheme),
			NewListFunc:   calicogalert.NewList,
			GetAttrsFunc:  calicogalert.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  gAlertRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	gAlertTemplateRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalalerttemplates"))
	if err != nil {
		return nil, err
	}
	gAlertTemplateOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   gAlertTemplateRESTOptions,
			Capacity:      1000,
			ObjectType:    calicogalert.EmptyObject(),
			ScopeStrategy: calicogalert.NewStrategy(scheme),
			NewListFunc:   calicogalert.NewList,
			GetAttrsFunc:  calicogalert.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  gAlertTemplateRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	gThreatFeedRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalthreatfeeds"))
	if err != nil {
		return nil, err
	}
	gThreatFeedOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   gThreatFeedRESTOptions,
			Capacity:      1000,
			ObjectType:    calicogthreatfeed.EmptyObject(),
			ScopeStrategy: calicogthreatfeed.NewStrategy(scheme),
			NewListFunc:   calicogthreatfeed.NewList,
			GetAttrsFunc:  calicogthreatfeed.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  gThreatFeedRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	hostEndpointRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("hostendpoints"))
	if err != nil {
		return nil, err
	}
	hostEndpointOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   hostEndpointRESTOptions,
			Capacity:      1000,
			ObjectType:    calicohostendpoint.EmptyObject(),
			ScopeStrategy: calicohostendpoint.NewStrategy(scheme),
			NewListFunc:   calicohostendpoint.NewList,
			GetAttrsFunc:  calicohostendpoint.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  hostEndpointRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"hep", "heps"},
	)

	globalReportRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalreports"))
	if err != nil {
		return nil, err
	}
	globalReportOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   globalReportRESTOptions,
			Capacity:      1000,
			ObjectType:    calicoglobalreport.EmptyObject(),
			ScopeStrategy: calicoglobalreport.NewStrategy(scheme),
			NewListFunc:   calicoglobalreport.NewList,
			GetAttrsFunc:  calicoglobalreport.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  globalReportRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	globalReportTypeRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("globalreporttypes"))
	if err != nil {
		return nil, err
	}
	globalReportTypeOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   globalReportTypeRESTOptions,
			Capacity:      1000,
			ObjectType:    calicoglobalreporttype.EmptyObject(),
			ScopeStrategy: calicoglobalreporttype.NewStrategy(scheme),
			NewListFunc:   calicoglobalreporttype.NewList,
			GetAttrsFunc:  calicoglobalreporttype.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  globalReportTypeRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	ipPoolRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("ippools"))
	if err != nil {
		return nil, err
	}
	ipPoolSetOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   ipPoolRESTOptions,
			Capacity:      10,
			ObjectType:    calicoippool.EmptyObject(),
			ScopeStrategy: calicoippool.NewStrategy(scheme),
			NewListFunc:   calicoippool.NewList,
			GetAttrsFunc:  calicoippool.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  ipPoolRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	bgpConfigurationRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("bgpconfigurations"))
	if err != nil {
		return nil, err
	}
	bgpConfigurationOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   bgpConfigurationRESTOptions,
			Capacity:      1000,
			ObjectType:    calicobgpconfiguration.EmptyObject(),
			ScopeStrategy: calicobgpconfiguration.NewStrategy(scheme),
			NewListFunc:   calicobgpconfiguration.NewList,
			GetAttrsFunc:  calicobgpconfiguration.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  bgpConfigurationRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"bgpconfig", "bgpconfigs"},
	)

	bgpPeerRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("bgppeers"))
	if err != nil {
		return nil, err
	}
	bgpPeerOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   bgpPeerRESTOptions,
			Capacity:      1000,
			ObjectType:    calicobgppeer.EmptyObject(),
			ScopeStrategy: calicobgppeer.NewStrategy(scheme),
			NewListFunc:   calicobgppeer.NewList,
			GetAttrsFunc:  calicobgppeer.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  bgpPeerRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	profileRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("profiles"))
	if err != nil {
		return nil, err
	}
	profileOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   profileRESTOptions,
			Capacity:      1000,
			ObjectType:    calicoprofile.EmptyObject(),
			ScopeStrategy: calicoprofile.NewStrategy(scheme),
			NewListFunc:   calicoprofile.NewList,
			GetAttrsFunc:  calicoprofile.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  profileRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	remoteclusterconfigRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("remoteclusterconfigurations"))
	if err != nil {
		return nil, err
	}
	remoteclusterconfigOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   remoteclusterconfigRESTOptions,
			Capacity:      1000,
			ObjectType:    calicoremoteclusterconfig.EmptyObject(),
			ScopeStrategy: calicoremoteclusterconfig.NewStrategy(scheme),
			NewListFunc:   calicoremoteclusterconfig.NewList,
			GetAttrsFunc:  calicoremoteclusterconfig.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  remoteclusterconfigRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	felixConfigRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("felixconfigurations"))
	if err != nil {
		return nil, err
	}
	felixConfigOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   felixConfigRESTOptions,
			Capacity:      1000,
			ObjectType:    calicofelixconfig.EmptyObject(),
			ScopeStrategy: calicofelixconfig.NewStrategy(scheme),
			NewListFunc:   calicofelixconfig.NewList,
			GetAttrsFunc:  calicofelixconfig.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  felixConfigRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"felixconfig", "felixconfigs"},
	)

	kubeControllersConfigsRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("kubecontrollersconfigurations"))
	if err != nil {
		return nil, err
	}
	kubeControllersConfigsOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   kubeControllersConfigsRESTOptions,
			Capacity:      1000,
			ObjectType:    calicokubecontrollersconfig.EmptyObject(),
			ScopeStrategy: calicokubecontrollersconfig.NewStrategy(scheme),
			NewListFunc:   calicokubecontrollersconfig.NewList,
			GetAttrsFunc:  calicokubecontrollersconfig.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  kubeControllersConfigsRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"kcconfig"},
	)

	managedClusterRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("managedclusters"))
	if err != nil {
		return nil, err
	}
	managedClusterOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   managedClusterRESTOptions,
			Capacity:      1000,
			ObjectType:    calicomanagedcluster.EmptyObject(),
			ScopeStrategy: calicomanagedcluster.NewStrategy(scheme),
			NewListFunc:   calicomanagedcluster.NewList,
			GetAttrsFunc:  calicomanagedcluster.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:             managedClusterRESTOptions,
			ManagedClusterResources: resources,
			LicenseCache:            licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	clusterInformationRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("clusterinformations"))
	if err != nil {
		return nil, err
	}
	clusterInformationOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   clusterInformationRESTOptions,
			Capacity:      1000,
			ObjectType:    calicoclusterinformation.EmptyObject(),
			ScopeStrategy: calicoclusterinformation.NewStrategy(scheme),
			NewListFunc:   calicoclusterinformation.NewList,
			GetAttrsFunc:  calicoclusterinformation.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  clusterInformationRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{"clusterinfo"},
	)

	packetCaptureRESTOptions, err := restOptionsGetter.GetRESTOptions(calico.Resource("packetcaptures"))
	if err != nil {
		return nil, err
	}
	packetCaptureOpts := server.NewOptions(
		etcd.Options{
			RESTOptions:   packetCaptureRESTOptions,
			Capacity:      1000,
			ObjectType:    calicopacketcapture.EmptyObject(),
			ScopeStrategy: calicopacketcapture.NewStrategy(scheme),
			NewListFunc:   calicopacketcapture.NewList,
			GetAttrsFunc:  calicopacketcapture.GetAttrs,
			Trigger:       nil,
		},
		calicostorage.Options{
			RESTOptions:  packetCaptureRESTOptions,
			LicenseCache: licenseCache,
		},
		p.StorageType,
		authorizer,
		[]string{},
	)

	storage := map[string]rest.Storage{}
	storage["networkpolicies"] = rESTInPeace(calicopolicy.NewREST(scheme, *policyOpts))
	storage["stagednetworkpolicies"] = rESTInPeace(calicostagedpolicy.NewREST(scheme, *stagedpolicyOpts))
	storage["stagedkubernetesnetworkpolicies"] = rESTInPeace(calicostagedk8spolicy.NewREST(scheme, *stagedk8spolicyOpts))
	storage["tiers"] = rESTInPeace(calicotier.NewREST(scheme, *tierOpts))
	storage["globalnetworkpolicies"] = rESTInPeace(calicogpolicy.NewREST(scheme, *gpolicyOpts))
	storage["stagedglobalnetworkpolicies"] = rESTInPeace(calicostagedgpolicy.NewREST(scheme, *stagedgpolicyOpts))
	storage["globalnetworksets"] = rESTInPeace(calicognetworkset.NewREST(scheme, *gNetworkSetOpts))
	storage["networksets"] = rESTInPeace(caliconetworkset.NewREST(scheme, *networksetOpts))
	licenseStorage, licenseStatusStorage, err := calicolicensekey.NewREST(scheme, *licenseKeysSetOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}

	storage["licensekeys"] = licenseStorage
	storage["licensekeys/status"] = licenseStatusStorage

	globalAlertsStorage, globalAlertsStatusStorage, err := calicogalert.NewREST(scheme, *gAlertOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	storage["globalalerts"] = globalAlertsStorage
	storage["globalalerts/status"] = globalAlertsStatusStorage
	storage["globalalerttemplates"] = rESTInPeace(calicogalerttemplate.NewREST(scheme, *gAlertTemplateOpts))

	globalThreatFeedsStorage, globalThreatFeedsStatusStorage, err := calicogthreatfeed.NewREST(scheme, *gThreatFeedOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	storage["globalthreatfeeds"] = globalThreatFeedsStorage
	storage["globalthreatfeeds/status"] = globalThreatFeedsStatusStorage

	storage["hostendpoints"] = rESTInPeace(calicohostendpoint.NewREST(scheme, *hostEndpointOpts))

	globalReportsStorage, globalReportsStatusStorage, err := calicoglobalreport.NewREST(scheme, *globalReportOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	storage["globalreports"] = globalReportsStorage
	storage["globalreports/status"] = globalReportsStatusStorage

	storage["globalreporttypes"] = rESTInPeace(calicoglobalreporttype.NewREST(scheme, *globalReportTypeOpts))
	storage["ippools"] = rESTInPeace(calicoippool.NewREST(scheme, *ipPoolSetOpts))
	storage["bgpconfigurations"] = rESTInPeace(calicobgpconfiguration.NewREST(scheme, *bgpConfigurationOpts))
	storage["bgppeers"] = rESTInPeace(calicobgppeer.NewREST(scheme, *bgpPeerOpts))
	storage["profiles"] = rESTInPeace(calicoprofile.NewREST(scheme, *profileOpts))
	storage["remoteclusterconfigurations"] = rESTInPeace(calicoremoteclusterconfig.NewREST(scheme, *remoteclusterconfigOpts))
	storage["felixconfigurations"] = rESTInPeace(calicofelixconfig.NewREST(scheme, *felixConfigOpts))

	kubeControllersConfigsStorage, kubeControllersConfigsStatusStorage, err := calicokubecontrollersconfig.NewREST(scheme, *kubeControllersConfigsOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	storage["kubecontrollersconfigurations"] = kubeControllersConfigsStorage
	storage["kubecontrollersconfigurations/status"] = kubeControllersConfigsStatusStorage

	managedClusterStorage, managedClusterStatusStorage, err := calicomanagedcluster.NewREST(scheme, *managedClusterOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	storage["managedclusters"] = managedClusterStorage
	storage["managedclusters/status"] = managedClusterStatusStorage

	storage["clusterinformations"] = rESTInPeace(calicoclusterinformation.NewREST(scheme, *clusterInformationOpts))
	storage["authenticationreviews"] = calicoauthenticationreview.NewREST()
	storage["authorizationreviews"] = calicoauthorizationreview.NewREST(calculator)

	packetCaptureStorage, err := calicopacketcapture.NewREST(scheme, *packetCaptureOpts)
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	storage["packetcaptures"] = packetCaptureStorage

	return storage, nil
}

// GroupName returns the API group name.
func (p RESTStorageProvider) GroupName() string {
	return calico.GroupName
}

// rESTInPeace is just a simple function that panics on error.
// Otherwise returns the given storage object. It is meant to be
// a wrapper for projectcalico registries.
func rESTInPeace(storage rest.Storage, err error) rest.Storage {
	if err != nil {
		err = fmt.Errorf("unable to create REST storage for a resource due to %v, will die", err)
		panic(err)
	}
	return storage
}
