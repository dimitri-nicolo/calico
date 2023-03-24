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
	cfg := rest.Config{
		CACertPath:     "cert/RootCA.crt",
		URL:            "https://localhost:8444/",
		ClientCertPath: "cert/localhost.crt",
		ClientKeyPath:  "cert/localhost.key",
	}

	// The token is created as part of FV setup in the Makefile, and mounted into the container that
	// runs the FV binaries.
	return client.NewClient("", cfg, rest.WithTokenPath(TokenPath))
}
