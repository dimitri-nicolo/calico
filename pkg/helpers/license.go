// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package helpers

import (
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
