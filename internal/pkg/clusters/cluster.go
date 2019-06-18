// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package clusters

// Cluster contains metadata used to track a specific cluster
type Cluster struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}
