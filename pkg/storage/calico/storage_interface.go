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
	default:
		glog.Fatalf("Unable to create storage for resource %v", opts.RESTOptions.ResourcePrefix)
		return nil, nil
	}
}
