// Copyright 2021 Tigera Inc. All rights reserved.

//go:build !tesla
// +build !tesla

package managedcluster

func getVariantSpecificClusterName(name string) string {
	return name
}
