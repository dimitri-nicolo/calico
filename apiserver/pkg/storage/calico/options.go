// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/projectcalico/calico/licensing/monitor"

	"k8s.io/apiserver/pkg/registry/generic"
)

const (
	PolicyResource string = "policy"
	TierResource   string = "tier"
)

// ManagedClusterResources contains resources needed by the managed cluster API
// to generate a fully populated manifest. The CA certificate and private key
// will be read from a local volume mounted using a K8S secret
type ManagedClusterResources struct {
	CACert                  *x509.Certificate
	CAKey                   *rsa.PrivateKey
	ManagementClusterAddr   string
	ManagementClusterCAType string
}

type Options struct {
	RESTOptions generic.RESTOptions
	*ManagedClusterResources
	LicenseMonitor monitor.LicenseMonitor
}
