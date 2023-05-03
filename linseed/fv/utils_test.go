// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

// Use a relative path to the client token in the calico-private/linseed/fv directory.
const TokenPath = "./client-token"

func NewLinseedClient() (client.Client, error) {
	// By default, all tests use "tenant-a" - this is because we launch Linseed
	// with EXPECTED_TENANT_ID=tenant-a for the FVs.
	// For tests that want to act as another tenant, use NewLinseedClientForTenant.
	return NewLinseedClientForTenant("tenant-a")
}

func NewLinseedClientForTenant(t string) (client.Client, error) {
	cfg := rest.Config{
		CACertPath:     "cert/RootCA.crt",
		URL:            "https://localhost:8444/",
		ClientCertPath: "cert/localhost.crt",
		ClientKeyPath:  "cert/localhost.key",
	}

	// The token is created as part of FV setup in the Makefile, and mounted into the container that
	// runs the FV binaries.
	return client.NewClient(t, cfg, rest.WithTokenPath(TokenPath))
}
