// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"testing"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

func TestEgressAccessControls(t *testing.T) {
	var data = []struct {
		gvk      string
		obj      resourceObject
		expected bool
	}{
		{libcalicoapi.NewNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.NetworkPolicy{Spec: libcalicoapi.NetworkPolicySpec{
			Egress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.NetworkPolicy{Spec: libcalicoapi.NetworkPolicySpec{
			Ingress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.NetworkPolicy{Spec: libcalicoapi.NetworkPolicySpec{}}, false},
		{libcalicoapi.NewGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.GlobalNetworkPolicy{Spec: libcalicoapi.GlobalNetworkPolicySpec{
			Egress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.GlobalNetworkPolicy{Spec: libcalicoapi.GlobalNetworkPolicySpec{
			Ingress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.GlobalNetworkPolicy{Spec: libcalicoapi.GlobalNetworkPolicySpec{}}, false},
		{libcalicoapi.NewStagedGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.StagedGlobalNetworkPolicy{Spec: libcalicoapi.StagedGlobalNetworkPolicySpec{
			Egress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewStagedGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.StagedGlobalNetworkPolicy{Spec: libcalicoapi.StagedGlobalNetworkPolicySpec{
			Ingress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewStagedGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.StagedGlobalNetworkPolicy{Spec: libcalicoapi.StagedGlobalNetworkPolicySpec{}}, false},
		{libcalicoapi.NewStagedNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.StagedNetworkPolicy{Spec: libcalicoapi.StagedNetworkPolicySpec{
			Egress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewStagedNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.StagedNetworkPolicy{Spec: libcalicoapi.StagedNetworkPolicySpec{
			Ingress: []libcalicoapi.Rule{
				{Destination: libcalicoapi.EntityRule{Domains: []string{"google.com"}}},
			},
		}}, true},
		{libcalicoapi.NewStagedNetworkPolicy().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.StagedNetworkPolicy{Spec: libcalicoapi.StagedNetworkPolicySpec{}}, false},
		{libcalicoapi.NewGlobalNetworkSet().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.GlobalNetworkSet{Spec: libcalicoapi.GlobalNetworkSetSpec{
			AllowedEgressDomains: []string{"google.com"},
		}}, true},
		{libcalicoapi.NewGlobalNetworkSet().GetObjectKind().GroupVersionKind().String(), &libcalicoapi.GlobalNetworkSet{Spec: libcalicoapi.GlobalNetworkSetSpec{}}, false},
	}

	const any = "any"

	for _, test := range data {
		var result = HasDNSDomains(test.gvk, test.obj)

		if result != test.expected {
			t.Fatalf("Check if object %s has a DNS domain defined. Expected check to return %v, but got %v instead", test.gvk, test.expected, result)
		}
	}
}

func TestStore(t *testing.T) {
	const any = "any"

	var data = []struct {
		license  *libcalicoapi.LicenseKey
		expected bool
	}{
		{getLicenseKey("default", validLicenseCertificate, validLicenseToken), true},
		{getLicenseKey("default", any, any), false},
	}

	for _, test := range data {

		var cache = NewLicenseCache()
		var result = cache.Store(*test.license)

		if result != test.expected {
			t.Fatalf("Failed to store license in cache. Expected result to be to be %v, but got %v instead", test.expected, result)
		}
	}
}

func TestClear(t *testing.T) {
	var cache = NewLicenseCache()

	// store a license
	result := cache.Store(*getLicenseKey("default", validLicenseCertificate, validLicenseToken))
	if !result {
		t.Fatalf("Expected storing the license in cache to be %v, but got %v instead", true, result)
	}

	// validate that API can be accessed
	claims := cache.FetchRegisteredFeatures()
	if claims == nil {
		t.Fatalf("Fetching license features from cache should have not returned an empty list, but got %v instead", claims)
	}

	// clear the cache
	cache.Clear()

	// validate that API can be accessed
	claims = cache.FetchRegisteredFeatures()
	if claims != nil {
		t.Fatalf("Fetching license features from cache should have returned an empty list, but got %v instead", claims)
	}
}
