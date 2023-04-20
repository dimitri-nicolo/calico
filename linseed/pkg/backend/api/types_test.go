// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/projectcalico/calico/linseed/pkg/backend/api"
)

func TestClusterInfo(t *testing.T) {
	// Missing a cluster name.
	info := api.ClusterInfo{}
	assert.Error(t, info.Valid())

	// Cluster has invalid symbol.
	info = api.ClusterInfo{Cluster: "one,two"}
	assert.Error(t, info.Valid())
	info.Cluster = "-one"
	assert.Error(t, info.Valid())
	info.Cluster = "sneaky*"
	assert.Error(t, info.Valid())

	// Valid - has a cluster.
	info = api.ClusterInfo{Cluster: "cloister"}
	assert.NoError(t, info.Valid())

	// Tenant has invalid symbol.
	info = api.ClusterInfo{Cluster: "cloister", Tenant: "some,value"}
	assert.Error(t, info.Valid())
	info.Cluster = "-nomatch"
	assert.Error(t, info.Valid())
	info.Cluster = "furtive*"
	assert.Error(t, info.Valid())
}
