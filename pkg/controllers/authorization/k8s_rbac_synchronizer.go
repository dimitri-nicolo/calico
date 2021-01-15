// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package authorization

const (
	roleMappingPrefix = "tigera-k8s"
	nativeUserPrefix  = "tigera-k8s"
)

// k8sRBACSynchronizer is an interface to sync k8s RBCA to backend.
type k8sRBACSynchronizer interface {
	// resync removes stale items in the backend that are not in cache and also
	// creates/overwrites items that are in cache to the backend.
	resync() error

	// synchronize updates/deletes the item in cache and backend for the given resource.
	synchronize(update resourceUpdate) error
}
