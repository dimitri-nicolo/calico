// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calico

import (
	"sync"

	licClient "github.com/tigera/licensing/client"
	"k8s.io/klog"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// LicenseCache stores LicenseKeys and validates API restrictions
type LicenseCache interface {
	FetchRegisteredFeatures() *licClient.LicenseClaims
	Store(licenseKey libcalicoapi.LicenseKey) bool
	Clear()
}

// LicenseCache caches the LicenseKey resource that is currently stored
// as "default" value
type licenseCache struct {
	// claims extracted from a valid LicenseKey
	claims *licClient.LicenseClaims
	mu     sync.RWMutex
}

// NewLicenseCache returns an implementation of LicenseCache
func NewLicenseCache() LicenseCache {
	return &licenseCache{}
}

func newLicenseCache(claims *licClient.LicenseClaims) LicenseCache {
	return &licenseCache{claims: claims}
}

// Store will store the claims extracted from a LicenseKey.
func (lc *licenseCache) Store(licenseKey libcalicoapi.LicenseKey) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	licClaims, err := licClient.Decode(licenseKey)
	if err != nil {
		klog.Warningf("Failed to store provided license - %v", err)
		return false
	}

	lc.claims = &licClaims
	return true
}

// Clear will remove previous claims extracted from a LicenseKey.
func (lc *licenseCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.claims = nil
}

// FetchRegisteredFeatures returns the features registered
func (lc *licenseCache) FetchRegisteredFeatures() *licClient.LicenseClaims {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	return lc.claims
}

// HasDNSDomains will check an object to see if a DNS domain defined
func HasDNSDomains(gvk string, lcObj resourceObject) bool {
	switch gvk {
	case libcalicoapi.NewNetworkPolicy().GetObjectKind().GroupVersionKind().String():
		policy := *lcObj.(*libcalicoapi.NetworkPolicy)
		return hasDNSDomain(policy.Spec.Egress) || hasDNSDomain(policy.Spec.Ingress)
	case libcalicoapi.NewGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String():
		policy := *lcObj.(*libcalicoapi.GlobalNetworkPolicy)
		return hasDNSDomain(policy.Spec.Egress) || hasDNSDomain(policy.Spec.Ingress)
	case libcalicoapi.NewStagedNetworkPolicy().GetObjectKind().GroupVersionKind().String():
		policy := *lcObj.(*libcalicoapi.StagedNetworkPolicy)
		return hasDNSDomain(policy.Spec.Egress) || hasDNSDomain(policy.Spec.Ingress)
	case libcalicoapi.NewStagedGlobalNetworkPolicy().GetObjectKind().GroupVersionKind().String():
		policy := *lcObj.(*libcalicoapi.StagedGlobalNetworkPolicy)
		return hasDNSDomain(policy.Spec.Egress) || hasDNSDomain(policy.Spec.Ingress)
	case libcalicoapi.NewGlobalNetworkSet().GetObjectKind().GroupVersionKind().String():
		networkSet := *lcObj.(*libcalicoapi.GlobalNetworkSet)
		return len(networkSet.Spec.AllowedEgressDomains) != 0
	default:
		return false
	}
}

func hasDNSDomain(rules []libcalicoapi.Rule) bool {
	for _, r := range rules {
		if len(r.Destination.Domains) != 0 {
			return true
		}
		if len(r.Source.Domains) != 0 {
			return true
		}
	}

	return false
}
