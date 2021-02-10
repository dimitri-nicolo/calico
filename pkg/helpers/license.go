// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package helpers

import (
	"sort"
	"strings"

	licFeatures "github.com/tigera/licensing/client/features"

	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// ConvertToPackageType converts the features array extracted from a license
// to a LicensePackageType
func ConvertToPackageType(features []string) libcalicoapi.LicensePackageType {
	switch strings.Join(features, "|") {
	case licFeatures.CloudCommunity:
		return libcalicoapi.CloudCommunity
	case licFeatures.CloudStarter:
		return libcalicoapi.CloudStarter
	case licFeatures.CloudPro:
		return libcalicoapi.CloudPro
	case licFeatures.Enterprise:
		return libcalicoapi.Enterprise
	default:
		return ""
	}
}

// ExpandFeatureNames expands the license package to the individual
// features that are available
func ExpandFeatureNames(features []string) []string {
	switch strings.Join(features, "|") {
	case licFeatures.CloudCommunity:
		return SortedKeys(licFeatures.CloudCommunityFeatures)
	case licFeatures.CloudStarter:
		return SortedKeys(licFeatures.CloudStarterFeatures)
	case licFeatures.CloudPro:
		return SortedKeys(licFeatures.CloudProFeatures)
	case licFeatures.Enterprise:
		return SortedKeys(licFeatures.EnterpriseFeatures)
	default:
		return []string{}
	}
}

// SortedKeys returns the keys of map sorted in an ascending order
func SortedKeys(set map[string]bool) []string {
	var keys []string
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
