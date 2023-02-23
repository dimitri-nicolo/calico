// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package main

import "github.com/projectcalico/calico/crypto/pkg/tls"

func main() {
	// Just making sure some crypto is present so the build is based on boring crypto. Please remove when the real code is merged.
	tls.NewTLSConfig(true)
}
