// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/client-go/kubernetes"

	"github.com/projectcalico/calico/licensing/monitor"
)

const (
	PolicyResource string = "policy"
	TierResource   string = "tier"
)

// ManagedClusterResources contains resources needed by the managed cluster API
// to generate a fully populated manifest. The CA certificate and private key
// will be read from a K8S secret
type ManagedClusterResources struct {
	ManagementClusterAddr   string
	ManagementClusterCAType string
	TunnelSecretName        string
	K8sClient               kubernetes.Interface
}

type Options struct {
	RESTOptions generic.RESTOptions
	*ManagedClusterResources
	LicenseMonitor monitor.LicenseMonitor
}
