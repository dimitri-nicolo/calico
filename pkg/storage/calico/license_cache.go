// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package calico

import (
	"sync"

	licClient "github.com/tigera/licensing/client"
	"k8s.io/klog"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// LicenseCache stores LicenseKeys and validates API restrictions
type LicenseCache interface {
	IsAPIRestricted(gvk string) bool
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

// IsAPIRestricted determines whether a projectcalico API can be used without
// any restrictions based on the license package
func (lc *licenseCache) IsAPIRestricted(gvk string) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	if len(gvk) == 0 {
		klog.Warningf("Group/Version/Kind is not defined. Resource cannot be identified.")
		return false
	}

	if licClient.IsOpenSourceAPI(gvk) {
		return false
	}

	if lc.claims == nil {
		klog.Warningf("LicenseCache has not been initialised with a valid license.")
		return true
	}

	return !lc.claims.ValidateAPIUsage(gvk)
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
