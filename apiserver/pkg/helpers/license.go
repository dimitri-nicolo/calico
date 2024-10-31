// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package helpers

import (
	"sort"
	"strings"

	libcalicoapi "github.com/tigera/api/pkg/apis/projectcalico/v3"

	licFeatures "github.com/projectcalico/calico/licensing/client/features"
)

// ConvertToPackageType converts the features array extracted from a license
// to a LicensePackageType
func ConvertToPackageType(features []string) libcalicoapi.LicensePackageType {
	if len(features) < 2 {
		return ""
	}

	switch strings.Join(features[0:2], "|") {
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
	var actualFeatures []string

	// This is required to maintain backwards compability to any CloudLicense issued for v3.5
	switch strings.Join(features, "|") {
	case licFeatures.CloudCommunity:
		// This maintains backwards compatibility for any license issued with only "cloud|community"
		actualFeatures = licFeatures.Keys(licFeatures.CloudCommunityFeatures)
	case licFeatures.CloudStarter:
		// This maintains backwards compability for any license issued with only "cloud|starter"
		actualFeatures = licFeatures.Keys(licFeatures.CloudStarterFeatures)
	case licFeatures.CloudPro:
		// This maintains backwards compability for any license issued with only "cloud|pro"
		actualFeatures = licFeatures.Keys(licFeatures.CloudProFeatures)
	default:
		// This maintains backwards compability for any license issued with only "cnx|all"
		// Or cloud licenses that have features listed
		actualFeatures = features
	}

	sort.Strings(actualFeatures)
	return actualFeatures

}
