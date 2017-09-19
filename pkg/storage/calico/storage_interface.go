package calico

import (
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

// NewStorage creates a new libcalico-based storage.Interface implementation
func NewStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	switch opts.resorceType {
	case "policy":
		return NewPolicyStorage(opts)
	case "tier":
		return NewTierStorage(opts)
	default:
		return nil, nil
	}
}
