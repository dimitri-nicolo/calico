// Copyright 2021 Tigera Inc. All rights reserved.

// +build !tesla

package managedcluster

func getVariantSpecificClusterName(name string) string {
	return name
}
