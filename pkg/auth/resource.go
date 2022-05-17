// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package auth

import (
	authzv1 "k8s.io/api/authorization/v1"
)

// CreateLMAResourceAttributes returns an authzv1.ResourceAttributes for the lma.tiger.io api group, setting the
// Resource to the given cluster and the Name to the given resourceName.
func CreateLMAResourceAttributes(cluster, resourceName string) *authzv1.ResourceAttributes {
	return &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "lma.tigera.io",
		Resource: cluster,
		Name:     resourceName,
	}
}
