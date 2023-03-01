// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api

// ClusterInfo defines the user who made the request.
type ClusterInfo struct {
	Cluster string
	Tenant  string
}
