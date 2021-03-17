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
	var featuresNames = strings.Join(features, "|")
	if strings.Contains(featuresNames, licFeatures.CloudCommunity) {
		return libcalicoapi.CloudCommunity
	}

	if strings.Contains(featuresNames, licFeatures.CloudStarter) {
		return libcalicoapi.CloudStarter
	}

	if strings.Contains(featuresNames, licFeatures.CloudPro) {
		return libcalicoapi.CloudPro
	}

	if strings.Contains(featuresNames, licFeatures.Enterprise) {
		return libcalicoapi.Enterprise
	}

	return ""
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
