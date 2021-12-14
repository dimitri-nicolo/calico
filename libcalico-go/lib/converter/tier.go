// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package converter

const (
	DefaultTierName = "default"
)

// TierOrDefault returns the tier name, or the default if blank.
func TierOrDefault(tier string) string {
	if len(tier) == 0 {
		return DefaultTierName
	} else {
		return tier
	}
}
