package calico

import (
	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

// NewStorage creates a new libcalico-based storage.Interface implementation
func NewStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	glog.V(4).Infoln("Constructing Calico Storage")

	return NewNetworkPolicyStorage(opts)
}
