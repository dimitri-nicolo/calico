// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"testing"
	"time"

	"github.com/tigera/licensing/client"
	"gopkg.in/square/go-jose.v2/jwt"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

func TestIsAPIRestricted(t *testing.T) {
	var data = []struct {
		featureLicense []string
		gvk            string
		expected       bool
	}{
		{[]string{}, "", false},
		{[]string{}, libcalicoapi.NewNetworkPolicy().GetObjectKind().GroupVersionKind().String(), false},
		{[]string{"cnx", "all"}, "", false},
		{[]string{"cloud", "community"}, libcalicoapi.NewLicenseKey().GetObjectKind().GroupVersionKind().String(), false},
		{[]string{"cloud", "community"}, libcalicoapi.NewGlobalAlert().GetObjectKind().GroupVersionKind().String(), true},
		{[]string{"cloud", "pro"}, libcalicoapi.NewGlobalAlert().GetObjectKind().GroupVersionKind().String(), false},
	}

	const any = "any"

	for _, test := range data {
		var numNodes = 5
		var claims = client.LicenseClaims{
			LicenseID:   any,
			Nodes:       &numNodes,
			Customer:    any,
			GracePeriod: 90,
			Features:    test.featureLicense,
			Claims: jwt.Claims{
				Expiry: jwt.NumericDate(time.Now().Add(72 * time.Hour).UTC().Unix()),
				Issuer: any,
			},
		}

		var cache = newLicenseCache(&claims)
		var result = cache.IsAPIRestricted(test.gvk)

		if result != test.expected {
			t.Fatalf("API restriction for %s is not enforced. Expect check to return %v, but got %v instead", test.gvk, test.expected, result)
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
			t.Fatalf("Expected storing the license in cache to be %v, but got %v instead", test.expected, result)
		}
	}
}

func TestClear(t *testing.T) {
	var data = []struct {
		api                       string
		isRestricted              bool
		isRestrictedAfterClearing bool
	}{
		{libcalicoapi.NewGlobalAlert().GetObjectKind().GroupVersionKind().String(), false, true},
		{libcalicoapi.NewLicenseKey().GetObjectKind().GroupVersionKind().String(), false, false},
	}

	for _, test := range data {
		var cache = NewLicenseCache()

		// store a license
		result := cache.Store(*getLicenseKey("default", validLicenseCertificate, validLicenseToken))
		if !result {
			t.Fatalf("Expected storing the license in cache to be %v, but got %v instead", true, result)
		}

		// validate that API can be accessed
		result = cache.IsAPIRestricted(test.api)
		if result != test.isRestricted {
			t.Fatalf("API restriction for %s is not enforced. Expect check to return %v, but got %v instead", test.api, test.isRestricted, result)
		}

		// clear the cache
		cache.Clear()

		// validate that API can no longer be accessed
		result = cache.IsAPIRestricted(test.api)
		if result != test.isRestrictedAfterClearing {
			t.Fatalf("API restriction for %s is not enforced. Expect check to return %v, but got %v instead", test.api, test.isRestrictedAfterClearing, result)
		}
	}
}
