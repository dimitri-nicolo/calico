// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package calico

import (
	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

// NewStorage creates a new libcalico-based storage.Interface implementation
func NewStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	glog.V(4).Infoln("Constructing Calico Storage")

	switch opts.RESTOptions.ResourcePrefix {
	case "projectcalico.org/networkpolicies":
		return NewNetworkPolicyStorage(opts)
	case "projectcalico.org/tiers":
		return NewTierStorage(opts)
	case "projectcalico.org/globalnetworkpolicies":
		return NewGlobalNetworkPolicyStorage(opts)
	case "projectcalico.org/globalnetworksets":
		return NewGlobalNetworkSetStorage(opts)
	case "projectcalico.org/licensekeys":
		return NewLicenseKeyStorage(opts)
	case "projectcalico.org/globalthreatfeeds":
		return NewGlobalThreatFeedStorage(opts)
	case "projectcalico.org/hostendpoints":
		return NewHostEndpointStorage(opts)
	case "projectcalico.org/globalreports":
		return NewGlobalReportStorage(opts)
	case "projectcalico.org/globalreporttypes":
		return NewGlobalReportTypeStorage(opts)
<<<<<<< HEAD
	case "projectcalico.org/ippools":
		return NewIPPoolStorage(opts)
=======
	case "projectcalico.org/bgpconfigurations":
		return NewBGPConfigurationStorage(opts)
>>>>>>> 1f9fbe90... Added BGPConfiguration resource to AAPI server
	default:
		glog.Fatalf("Unable to create storage for resource %v", opts.RESTOptions.ResourcePrefix)
		return nil, nil
	}
}
