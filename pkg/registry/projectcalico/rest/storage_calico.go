/*
Copyright 2016 The Kubernetes Authors.

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

package rest

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"

	calico "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
	calicobgpconfiguration "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/bgpconfiguration"
	calicognetworkset "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/globalnetworkset"
	calicogpolicy "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/globalpolicy"
	calicoglobalreport "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/globalreport"
	calicoglobalreporttype "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/globalreporttype"
	calicogthreatfeed "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/globalthreatfeed"
	calicohostendpoint "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/hostendpoint"
	calicoippool "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/ippool"
	calicolicensekey "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/licensekey"
	calicopolicy "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/networkpolicy"
	"github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/server"
	calicotier "github.com/tigera/calico-k8sapiserver/pkg/registry/projectcalico/tier"
	calicostorage "github.com/tigera/calico-k8sapiserver/pkg/storage/calico"
	"github.com/tigera/calico-k8sapiserver/pkg/storage/etcd"
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: policyRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: tierRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: gpolicyRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: gNetworkSetRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: licenseKeyRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: gThreatFeedRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: hostEndpointRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: globalReportRESTOptions,
		},
		p.StorageType,
		authorizer,
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: globalReportTypeRESTOptions,
		},
		p.StorageType,
		authorizer,
	)

<<<<<<< HEAD
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: ipPoolRESTOptions,
=======
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
			Trigger:       storage.NoTriggerPublisher,
		},
		calicostorage.Options{
			RESTOptions: bgpConfigurationRESTOptions,
>>>>>>> 1f9fbe90... Added BGPConfiguration resource to AAPI server
		},
		p.StorageType,
		authorizer,
	)

	storage := map[string]rest.Storage{}
	storage["networkpolicies"] = calicopolicy.NewREST(scheme, *policyOpts)
	storage["tiers"] = calicotier.NewREST(scheme, *tierOpts)
	storage["globalnetworkpolicies"] = calicogpolicy.NewREST(scheme, *gpolicyOpts)
	storage["globalnetworksets"] = calicognetworkset.NewREST(scheme, *gNetworkSetOpts)
	storage["licensekeys"] = calicolicensekey.NewREST(scheme, *licenseKeysSetOpts)
<<<<<<< HEAD
	storage["ippools"] = calicoippool.NewREST(scheme, *ipPoolSetOpts)
=======
	storage["bgpconfigurations"] = calicobgpconfiguration.NewREST(scheme, *bgpConfigurationOpts)
>>>>>>> 1f9fbe90... Added BGPConfiguration resource to AAPI server

	globalThreatFeedsStorage, globalThreatFeedsStatusStorage := calicogthreatfeed.NewREST(scheme, *gThreatFeedOpts)
	storage["globalthreatfeeds"] = globalThreatFeedsStorage
	storage["globalthreatfeeds/status"] = globalThreatFeedsStatusStorage

	storage["hostendpoints"] = calicohostendpoint.NewREST(scheme, *hostEndpointOpts)

	globalReportsStorage, globalReportsStatusStorage := calicoglobalreport.NewREST(scheme, *globalReportOpts)
	storage["globalreports"] = globalReportsStorage
	storage["globalreports/status"] = globalReportsStatusStorage

	storage["globalreporttypes"] = calicoglobalreporttype.NewREST(scheme, *globalReportTypeOpts)

	return storage, nil
}

// GroupName returns the API group name.
func (p RESTStorageProvider) GroupName() string {
	return calico.GroupName
}
